package models

import (
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
	"time"
)

type UploadRecord struct {
	Id              int       `json:"id"`
	OrganizationID  string    `json:"organization_id"`
	DatasetID       string    `json:"dataset_id"`
	PackageID       string    `json:"package_id"`
	SourcePath      string    `json:"source_path"`
	TargetPath      string    `json:"target_path"`
	ImportID        string    `json:"import_id"`
	ImportSessionID string    `json:"import_session_id"`
	progress        int       `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UploadRecordParams struct {
	OrganizationID  string `json:"organization_id"`
	DatasetID       string `json:"dataset_id"`
	SourcePath      string `json:"source_path"`
	TargetPath      string `json:"target_path"`
	ImportSessionID string `json:"import_session_id"`
}

// AddToUploadSession adds files to upload manifest in local DB
func AddToUploadSession(session string, path string, recursive bool, targetPath string) {
	walker := make(fileWalk)

	// Use absolute path to support '.' as the initial path without ignoring it
	absPath, _ := filepath.Abs(path)

	go func() {
		// Gather the files to upload by walking the path recursively
		if err := filepath.WalkDir(absPath, walker.Walk); err != nil {
			log.Fatalln("Walk failed:", err)
		}
		close(walker)
	}()

	for path := range walker {
		fmt.Println(path)
	}
}

type fileWalk chan string

//Walk provides walker function and skips hidden files/folders.
func (f fileWalk) Walk(path string, info fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if !info.IsDir() {
		if !strings.HasPrefix(info.Name(), ".") {
			f <- path
		}
	} else if strings.HasPrefix(info.Name(), ".") {
		return filepath.SkipDir
	}

	return nil
}
