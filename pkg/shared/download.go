package shared

import (
    "context"
    "fmt"
    api "github.com/pennsieve/pennsieve-agent/v2/api/v1"
    "github.com/pennsieve/pennsieve-go/pkg/pennsieve"
    "github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/download"
    log "github.com/sirupsen/logrus"
    "hash/crc32"
    "io"
    "net/http"
    "os"
    "sync"
    "time"
)

type downloadSession struct {
    id        string
    cancelFnc context.CancelFunc
}

type Downloader interface {
    DownloadFileFromPresignedUrl(
        ctx context.Context,
        url string,
        targetLocation string,
        downloadId string) (uint32, error)
    DownloadManifest(ctx context.Context, d ManifestDownload, first *download.ManifestPage) (ManifestResult, error)
}

// StatusError is a download the storage server refused, such as an expired
// link (403).
type StatusError struct {
    StatusCode int
}

func (e *StatusError) Error() string {
    return fmt.Sprintf("download refused: HTTP %d", e.StatusCode)
}

type Subscriber interface {
    GetSubscribers() sync.Map
}

type downloader struct {
    server             Subscriber
    pennsieveClient    *pennsieve.Client
    downloadCancelFncs sync.Map // downloadCancelFncs is a map that holds cancel functions for download routines.
}

func NewDownloader(s Subscriber, client *pennsieve.Client) downloader {
    return downloader{server: s, pennsieveClient: client}
}

type ProgressReader struct {
    Reader io.Reader
    Size   int64
    Pos    int64
    s      *downloader
    Name   string
    crc32  uint32
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
    crc := crc32.Update(pr.crc32, crc32.IEEETable, p)
    n, err := pr.Reader.Read(p)
    if err == nil {
        pr.Pos += int64(n)
        pr.crc32 = crc
        pr.s.updateDownloadSubscribers(pr.Size, pr.Pos, pr.Name, api.SubscribeResponse_DownloadStatusResponse_IN_PROGRESS)
    }
    return n, err
}

func (s *downloader) CancelDownload(ctx context.Context, req *api.CancelDownloadRequest) (*api.SimpleStatusResponse, error) {
    cancelCount := 0
    s.downloadCancelFncs.Range(func(k, v interface{}) bool {

        session := v.(downloadSession)
        if !req.CancelAll {

            // Only cancel if the package_id matches requested id
            if req.Id != nil {
                if session.id == *req.Id {
                    session.cancelFnc()
                    s.sendCancelSubscribers(fmt.Sprintf("Cancelling all downloads."))
                    cancelCount += 1
                    return false
                }
            }

        } else {
            // Cancel all upload sessions.
            session.cancelFnc()
            s.sendCancelSubscribers(fmt.Sprintf("Cancelling download package: %s", session.id))
            cancelCount += 1
        }

        return true
    })

    return &api.SimpleStatusResponse{
        Status: fmt.Sprintf("Succesfully cancelled %d download sessions", cancelCount)}, nil
}

// downloadFileFromPresignedUrl downloads a file from a presigned URL
//
//	downloadId is a unique id that is associated with the download (i.e. packageId, or manifestId)
//	targetLocation is the absolute path and file-name where the downloaded content is stored
func (s *downloader) DownloadFileFromPresignedUrl(ctx context.Context, url string, targetLocation string, downloadId string) (uint32, error) {

    start := time.Now().UnixMilli()

    // Cancelled with ctx: the caller's download, or CancelDownload.
    ctx, cancelFnc := context.WithCancel(ctx)
    defer cancelFnc()
    session := downloadSession{
        id:        downloadId,
        cancelFnc: cancelFnc,
    }

    s.downloadCancelFncs.Store(downloadId, session)
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return 0, err
    }

    // Not the URL: it carries a signature.
    log.Infof("Downloading %s", targetLocation)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return 0, err
    }
    defer func(Body io.ReadCloser) {
        err := Body.Close()
        if err != nil {
            log.Warnf("Failed to close response body: %v", err)
        }
    }(resp.Body)

    if resp.StatusCode != http.StatusOK {
        log.Infof("Error while downloading: %v", resp.StatusCode)
        fmt.Println(" - Download cancelled")
        return 0, &StatusError{StatusCode: resp.StatusCode}
    }

    f, err := os.OpenFile(targetLocation, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
    if err != nil {
        return 0, err
    }
    defer func(f *os.File) {
        err := f.Close()
        if err != nil {
            log.Warnf("Failed to close response body: %v", err)
        }
    }(f)

    progressReader := &ProgressReader{
        Reader: resp.Body,
        Size:   resp.ContentLength,
        s:      s,
        Name:   targetLocation,
        crc32:  0,
    }

    if _, err = io.Copy(f, progressReader); err != nil {
        log.Infof("Error while downloading: %v", err)
        fmt.Println(" - Download cancelled")

        // Repopulate dummy file with the packageID
        f, openErr := os.OpenFile(targetLocation, os.O_TRUNC|os.O_WRONLY, 0644)
        if openErr != nil {
            log.Warnf("Failed to write PackageID: %v", openErr)
            return 0, err
        }
        defer func(f *os.File) {
            _ = f.Close()
        }(f)

        if _, writeErr := f.WriteString(downloadId); writeErr != nil {
            log.Warnf("Failed to write PackageID to file: %v", writeErr)
            return 0, err
        }
        return 0, err
    }

    s.updateDownloadSubscribers(resp.ContentLength, resp.ContentLength, targetLocation, api.SubscribeResponse_DownloadStatusResponse_COMPLETE)

    fmt.Println(" - Download completed!")
    fmt.Printf("Took: %.2fs\n", float64(time.Now().UnixMilli()-start)/1000)

    return progressReader.crc32, nil

}

// updateDownloadSubscribers sends download-progress updates to all grpc-update subscribers.
func (s *downloader) updateDownloadSubscribers(total int64, current int64, name string,
    status api.SubscribeResponse_DownloadStatusResponse_DownloadStatus) {
    // A list of clients to unsubscribe in case of error
    var unsubscribe []int32

    // Iterate over all subscribers and send data to each client
    subscribers := s.server.GetSubscribers()
    subscribers.Range(func(k, v interface{}) bool {
        id, ok := k.(int32)
        if !ok {
            log.Error(fmt.Sprintf("Failed to cast subscriber key: %T", k))
            return false
        }
        sub, ok := v.(Sub)
        if !ok {
            log.Error(fmt.Sprintf("Failed to cast subscriber value: %T", v))
            return false
        }
        // Send data over the gRPC stream to the client
        if err := sub.Stream.Send(&api.SubscribeResponse{
            Type: 4,
            MessageData: &api.SubscribeResponse_DownloadStatus{
                DownloadStatus: &api.SubscribeResponse_DownloadStatusResponse{
                    FileId:  name,
                    Total:   total,
                    Current: current,
                    Status:  status,
                }},
        }); err != nil {

            select {
            case sub.Finished <- true:
                log.Info(fmt.Sprintf("Unsubscribed client: %d", id))
            default:
                log.Warn(fmt.Sprintf("Failed to send data to client: %v", err))
                // Default case is to avoid blocking in case client has already unsubscribed
            }
            // In case of error the client would re-subscribe so close the subscriber stream
            unsubscribe = append(unsubscribe, id)
        }
        return true
    })

    // Unsubscribe erroneous client streams
    for _, id := range unsubscribe {
        subscribers := s.server.GetSubscribers()
        subscribers.Delete(id)
    }
}

// sendCancelSubscribers Send Cancel Message to subscribers
func (s *downloader) sendCancelSubscribers(message string) {
    // A list of clients to unsubscribe in case of error
    var unsubscribe []int32

    // Iterate over all subscribers and send data to each client
    subscribers := s.server.GetSubscribers()
    subscribers.Range(func(k, v interface{}) bool {
        id, ok := k.(int32)
        if !ok {
            log.Error(fmt.Sprintf("Failed to cast subscriber key: %T", k))
            return false
        }
        sub, ok := v.(Sub)
        if !ok {
            log.Error(fmt.Sprintf("Failed to cast subscriber value: %T", v))
            return false
        }
        // Send data over the gRPC stream to the client
        if err := sub.Stream.Send(&api.SubscribeResponse{
            Type: api.SubscribeResponse_DOWNLOAD_CANCEL,
            MessageData: &api.SubscribeResponse_EventInfo{
                EventInfo: &api.SubscribeResponse_EventResponse{Details: message}},
        }); err != nil {
            select {
            case sub.Finished <- true:
                log.Info(fmt.Sprintf("Unsubscribed client: %d", id))
            default:
                log.Warn(fmt.Sprintf("Failed to send data to client: %v", err))
                // Default case is to avoid blocking in case client has already unsubscribed
            }
            // In case of error the client would re-subscribe so close the subscriber stream
            unsubscribe = append(unsubscribe, id)
        }
        return true
    })

    // Unsubscribe erroneous client streams
    for _, id := range unsubscribe {
        subscribers := s.server.GetSubscribers()
        subscribers.Delete(id)
    }
}
