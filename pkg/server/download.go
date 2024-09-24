package server

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/google/uuid"
    api "github.com/pennsieve/pennsieve-agent/api/v1"
    "github.com/pennsieve/pennsieve-go-core/pkg/models/workspaceManifest"
    "github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/ps_package"
    log "github.com/sirupsen/logrus"
    "io"
    "net/http"
    "os"
    "path"
    "sync"
    "time"
)

type ProgressReader struct {
    Reader io.Reader
    Size   int64
    Pos    int64
    s      *server
    Name   string
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
    n, err := pr.Reader.Read(p)
    if err == nil {
        pr.Pos += int64(n)
        pr.s.updateDownloadSubscribers(pr.Size, pr.Pos, pr.Name, api.SubscribeResponse_DownloadStatusResponse_IN_PROGRESS)
    }
    return n, err
}

func (s *server) Download(ctx context.Context, req *api.DownloadRequest) (*api.DownloadResponse, error) {

    res := &ps_package.GetPresignedUrlResponse{}

    var err error

    responseType := api.DownloadResponse_PRESIGNED_URL

    switch req.Type {
    case api.DownloadRequest_PACKAGE:
        log.Debug("download request for package")

        // Download single Package
        requestData := req.GetPackage()

        // Need to get a presigned URL for the package.
        // This can return an array of results in case a package has multiple source files.
        client := s.client
        res, err = client.Package.GetPresignedUrl(ctx, requestData.PackageId, true)

        if !requestData.GetPresignedUrl {
            log.Debug("Downloading the package.")
            go func() {
                // Iterate over the files in a package and download serially
                // TODO: can optimize by concurrency
                for _, f := range res.Files {
                    err = s.downloadFileFromPresignedUrl(ctx, f.URL, f.Name, requestData.PackageId)
                    if err != nil {
                        log.Errorf("Download failed: %v", err)
                    }
                }

            }()
        }

    case api.DownloadRequest_DATASET:

        // Request Data Should contain dataset-id
        requestData := req.GetDataset()

        //
        client := s.client
        manifestResponse, err := client.Dataset.GetManifest(ctx, requestData.DatasetId)
        if err != nil {
            log.Errorf("Download failed: %v", err)

            return nil, err
        }

        // Create folder (and include hidden .pennsieve folder for manifest)
        err = os.MkdirAll(path.Join(requestData.TargetFolder, ".pennsieve"), os.ModePerm)
        if err != nil {
            log.Errorf("Failed to create target path: %v", err)
            return nil, err
        }

        // Download Manifest to hidden .pennsieve folder in target path
        manifestLocation := path.Join(requestData.TargetFolder, ".pennsieve", "manifest.json")
        err = s.downloadFileFromPresignedUrl(ctx, manifestResponse.URL, manifestLocation, uuid.New().String())
        if err != nil {
            log.Errorf("Download failed: %v", err)
        }

        // Now read in manifest
        manifestFile, err := os.Open(manifestLocation)
        if err != nil {
            fmt.Printf("failed to open manifest file: %s, error: %v", manifestLocation, err)
            return nil, err
        }
        defer func(manifestFile *os.File) {
            err := manifestFile.Close()
            if err != nil {
                log.Warnf("Failed to close manifest file: %s, error: %v", manifestLocation, err)
            }
        }(manifestFile)

        jsonData, err := io.ReadAll(manifestFile)
        if err != nil {
            fmt.Printf("failed to read json file, error: %v", err)
            return nil, err
        }

        data := models.WorkspaceManifest{}
        if err := json.Unmarshal(jsonData, &data); err != nil {
            fmt.Printf("failed to unmarshal json file, error: %v", err)
            return nil, err
        }

        // Go routines for downloading data in parallel
        nrWorkers := 5
        walker := make(chan models.ManifestDTO, nrWorkers)
        results := make(chan int, nrWorkers)
        var downloadWg sync.WaitGroup

        go func() {
            defer close(walker)
            for _, file := range data.Files {
                walker <- file
            }
        }()

        for w := 0; w < nrWorkers; w++ {
            downloadWg.Add(1)
            log.Println("Starting download worker: ", w)

            w := w
            go func() {
                defer func() {
                    log.Println("Closing download worker: ", w)
                    downloadWg.Done()
                }()

                s.downloadWorker(ctx, w, walker, results, requestData.TargetFolder)

            }()

        }

        downloadWg.Wait()

    }

    resp := &api.DownloadResponse{
        Type:   responseType,
        Status: "Success",
        Url:    []string{""},
    }

    return resp, nil
}

func (s *server) downloadWorker(ctx context.Context, workerId int,
    jobs <-chan models.ManifestDTO, result <-chan int, targetFolder string,
) {

    for record := range jobs {
        err := os.MkdirAll(path.Join(targetFolder, record.Path), os.ModePerm)

        res, err := s.client.Package.GetPresignedUrl(ctx, record.PackageNodeId, false)
        if err != nil {
            log.Errorf("Download failed: %v", err)
            continue
        }

        // We are iterating over list of files, but getPresignedUrl works over package so
        // will need to figure out which file in package is current iteration
        preURL := ""
        for _, f := range res.Files {
            preURL = f.URL
            if f.Name == record.FileName.String {
                break
            }
        }
        if preURL == "" {
            log.Error("Cannot find file in returned presigned url array")
        }

        fileLocation := path.Join(targetFolder, record.FileName.String)
        err = s.downloadFileFromPresignedUrl(ctx, preURL, fileLocation, record.PackageNodeId)
        if err != nil {
            log.Errorf("Download failed: %v", err)
        }

    }

}

func (s *server) CancelDownload(ctx context.Context, req *api.CancelDownloadRequest) (*api.SimpleStatusResponse, error) {
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
func (s *server) downloadFileFromPresignedUrl(ctx context.Context, url string, targetLocation string, downloadId string) error {

    start := time.Now().UnixMilli()

    prefix, err := os.UserHomeDir()
    tempPath := path.Join(prefix, ".pennsieve", fmt.Sprintf(".%s_download", uuid.NewString()))

    ctx, cancelFnc := context.WithCancel(context.Background())
    session := downloadSession{
        id:        downloadId,
        cancelFnc: cancelFnc,
    }

    s.downloadCancelFncs.Store(downloadId, session)
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := http.DefaultClient.Do(req)

    if resp.StatusCode != 200 {
        log.Infof("Error while downloading: %v", resp.StatusCode)
        fmt.Println(" - Download cancelled")
        _ = os.Remove(tempPath)
        return err
    }
    defer func(Body io.ReadCloser) {
        err := Body.Close()
        if err != nil {
            log.Warnf("Failed to close response body: %v", err)
        }
    }(resp.Body)

    f, _ := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY, 0644)
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
    }

    if _, err = io.Copy(f, progressReader); err != nil {
        log.Infof("Error while downloading: %v", err)
        fmt.Println(" - Download cancelled")
        _ = os.Remove(tempPath)
        return err
    }

    err = os.Rename(tempPath, targetLocation)
    if err != nil {
        return err
    }

    s.updateDownloadSubscribers(resp.ContentLength, resp.ContentLength, targetLocation, api.SubscribeResponse_DownloadStatusResponse_COMPLETE)

    fmt.Println(" - Download completed!")
    fmt.Printf("Took: %.2fs\n", float64(time.Now().UnixMilli()-start)/1000)

    return nil

}

// updateDownloadSubscribers sends download-progress updates to all grpc-update subscribers.
func (s *server) updateDownloadSubscribers(total int64, current int64, name string,
    status api.SubscribeResponse_DownloadStatusResponse_DownloadStatus) {
    // A list of clients to unsubscribe in case of error
    var unsubscribe []int32

    // Iterate over all subscribers and send data to each client
    s.subscribers.Range(func(k, v interface{}) bool {
        id, ok := k.(int32)
        if !ok {
            log.Error(fmt.Sprintf("Failed to cast subscriber key: %T", k))
            return false
        }
        sub, ok := v.(sub)
        if !ok {
            log.Error(fmt.Sprintf("Failed to cast subscriber value: %T", v))
            return false
        }
        // Send data over the gRPC stream to the client
        if err := sub.stream.Send(&api.SubscribeResponse{
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
            case sub.finished <- true:
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
        s.subscribers.Delete(id)
    }
}
