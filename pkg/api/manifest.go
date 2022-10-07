package api

import (
	"fmt"
	"github.com/pennsieve/pennsieve-agent/models"
	dbconfig "github.com/pennsieve/pennsieve-agent/pkg/db"
	"github.com/pennsieve/pennsieve-go-api/pkg/models/manifest/manifestFile"
	"log"
)

// SyncResponse returns summary info from ManifestSync method.
type SyncResponse struct {
	ManifestNodeId string
	NrFilesUpdated int
	NrFilesRemoved int
	FailedFiles    []string
}

// VerifyFinalizedStatus checks if files are in "Finalized" state on server and sets to "Verified"
func VerifyFinalizedStatus(m *models.Manifest) error {
	log.Println("Verifying files")

	response, err := PennsieveClient.Manifest.GetFilesForStatus(nil, m.NodeId.String, manifestFile.Finalized, "", true)
	if err != nil {
		log.Println("Error getting files for status, here is why: ", err)
		return err
	}

	var mf models.ManifestFile
	log.Println("Number of responses: ", len(response.Files))
	if len(response.Files) > 0 {
		if len(response.Files) == 1 {
			mf.SetStatus(dbconfig.DB, manifestFile.Verified, response.Files[0])
		} else {
			mf.BatchSetStatus(dbconfig.DB, manifestFile.Verified, response.Files)
		}
	}

	fmt.Println(len(response.ContinuationToken))
	for {
		if len(response.ContinuationToken) > 0 {
			log.Println("Getting another set of files ")
			response, err = PennsieveClient.Manifest.GetFilesForStatus(nil, m.NodeId.String, manifestFile.Finalized, response.ContinuationToken, true)
			if err != nil {
				log.Println("Error getting files for status, here is why: ", err)
				return err
			}
			if len(response.Files) > 0 {
				if len(response.Files) == 1 {
					mf.SetStatus(dbconfig.DB, manifestFile.Verified, response.Files[0])
				} else {
					mf.BatchSetStatus(dbconfig.DB, manifestFile.Verified, response.Files)
				}
			}
		} else {
			break
		}
	}

	return nil
}
