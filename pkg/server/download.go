package server

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/ps_package"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
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
		fmt.Printf("\rDownloading... %.2f%%", float64(pr.Pos)/float64(pr.Size)*100)
		pr.s.updateDownloadSubscribers(pr.Size, pr.Pos, pr.Name, api.SubscribeResponse_DownloadStatusResponse_IN_PROGRESS)

	}
	return n, err
}

func (s *server) Download(ctx context.Context, req *api.DownloadFileRequest) (*api.DownloadFileResponse, error) {

	res := &ps_package.GetPresignedUrlResponse{}
	var err error

	responseType := api.DownloadFileResponse_PRESIGNED_URL

	// Need to get a presigned URL for the package
	client := s.client
	if req.GetPresignedUrl {
		res, err = client.Package.GetPresignedUrl(ctx, req.PackageId, true)
	} else {

		go func() {
			start := time.Now().UnixMilli()
			tempPath := fmt.Sprintf(".%s_download", uuid.NewString())

			responseType = api.DownloadFileResponse_DOWNLOAD
			ctx, cancelFnc := context.WithCancel(context.Background())
			session := downloadSession{
				packageId: req.PackageId,
				cancelFnc: cancelFnc,
			}

			s.downloadCancelFncs.Store(req.PackageId, session)
			res, err = client.Package.GetPresignedUrl(ctx, req.PackageId, false)
			outPath := res.Files[0].Name

			req, _ := http.NewRequestWithContext(ctx, "GET", res.Files[0].URL, nil)
			resp, _ := http.DefaultClient.Do(req)

			if resp.StatusCode != 200 {
				log.Infof("Error while downloading: %v", resp.StatusCode)
				fmt.Println(" - Download cancelled")
				_ = os.Remove(tempPath)
				return
			}
			defer resp.Body.Close()

			f, _ := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY, 0644)
			defer f.Close()

			progressReader := &ProgressReader{
				Reader: resp.Body,
				Size:   resp.ContentLength,
				s:      s,
				Name:   outPath,
			}

			if _, err := io.Copy(f, progressReader); err != nil {
				log.Infof("Error while downloading: %v", err)
				fmt.Println(" - Download cancelled")
				_ = os.Remove(tempPath)
				return
			}

			os.Rename(tempPath, outPath)
			fmt.Println(" - Download completed!")
			fmt.Printf("Took: %.2fs\n", float64(time.Now().UnixMilli()-start)/1000)
		}()

	}

	if err != nil {
		log.Println("Error downloading file:", err)
	}

	resp := &api.DownloadFileResponse{
		Type:   responseType,
		Status: "Success",
		Url:    []string{""},
	}

	return resp, nil
}

func (s *server) CancelDownload(ctx context.Context, req *api.CancelDownloadRequest) (*api.SimpleStatusResponse, error) {
	cancelCount := 0
	s.downloadCancelFncs.Range(func(k, v interface{}) bool {

		session := v.(downloadSession)
		if !req.CancelAll {
			// Only cancel if the package_id matches requested id
			if session.packageId == req.PackageId {
				session.cancelFnc()
				s.sendCancelSubscribers(fmt.Sprintf("Cancelling all downloads."))
				cancelCount += 1
				return false
			}
		} else {
			// Cancel all upload sessions.
			session.cancelFnc()
			s.sendCancelSubscribers(fmt.Sprintf("Cancelling download package: %s", session.packageId))
			cancelCount += 1
		}

		return true
	})

	return &api.SimpleStatusResponse{
		Status: fmt.Sprintf("Succesfully cancelled %d download sessions", cancelCount)}, nil
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
