package server

import (
	"context"
	"errors"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	models2 "github.com/pennsieve/pennsieve-agent/pkg/models"
	"github.com/pennsieve/pennsieve-agent/pkg/shared"
	models "github.com/pennsieve/pennsieve-go-core/pkg/models/workspaceManifest"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
)

func (s *server) GetMapDiff(_ context.Context, req *api.MapDiffRequest) (*api.MapDiffResponse, error) {

	// Check if the provided path is part of a mapped dataset
	datasetRoot, found, err := findMappedDatasetRoot(req.Path)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("cannot find root of the dataset")
	}

	files, err := createFolderManifest(datasetRoot)
	if err != nil {
		return nil, err
	}

	manifest, err := shared.ReadWorkspaceManifest(filepath.Join(datasetRoot, ".pennsieve", "manifest.json"))
	if err != nil {
		return nil, err
	}

	result, err := compareManifestToFolder(datasetRoot, manifest.Files, files)

	log.Warn(result)

	if err != nil {
		return nil, err
	}

	// Map result into response
	var response api.MapDiffResponse
	for _, r := range result {

		content := api.FileInfo{}
		switch r.Type {
		case api.PackageStatus_MOVED_RENAMED:
			fallthrough
		case api.PackageStatus_ADDED:
			fallthrough
		case api.PackageStatus_RENAMED:
			fallthrough
		case api.PackageStatus_MOVED:
			content = api.FileInfo{
				PackageId: r.Old.PackageNodeId,
				Path:      r.New.Path,
				Name:      r.New.FileName,
				Message:   "",
			}
		case api.PackageStatus_DELETED:
			content = api.FileInfo{
				PackageId: r.Old.PackageNodeId,
				Path:      r.Old.Path,
				Name:      r.Old.FileName,
				Message:   "",
			}
		case api.PackageStatus_CHANGED:
			content = api.FileInfo{
				PackageId: r.Changed.from.PackageNodeId,
				Path:      r.Changed.from.Path,
				Name:      r.Changed.from.PackageName,
				Message:   "",
			}

		}

		record := api.PackageStatus{
			Content:    &content,
			ChangeType: r.Type,
		}

		response.Files = append(response.Files, &record)

	}

	return &response, nil
}

// CrcSize is the length of the buffer that is used to calculate the CRC32 for the files.
// Files are considered the same if files match for the length of the buffer.
// This is only tested for new files and compared to other files in manifest that are considered deleted.
const CrcSize = 1024 * 1024

type crcOrFileId struct {
	hasFileId bool
	FileId    string
	Crc32     uint32
}

type folderFile struct {
	FileName string
	Path     string
	Size     int64
}

type addedFile struct {
	FileName  string
	Path      string
	Size      int64
	hasFileId bool
	Crc32     uint32
	FileId    string
}

type deletedFile struct {
	PackageNodeId string
	FileName      string
	Path          string
	Size          int64
	FileId        string
	hasMatched    bool // Boolean indicating whether deleted file matched
}

type changedFile struct {
	Size  int64
	Crc32 uint32
	from  models.ManifestDTO
}

type renamedMovedFile struct {
	Type api.PackageStatus_StatusType
	Old  deletedFile
	New  addedFile
}

type diffResult struct {
	Type     api.PackageStatus_StatusType
	FilePath string
	Old      deletedFile
	New      addedFile
	Changed  changedFile
}

func getFileIdOrCrc32(path string, maxBytes int) (*crcOrFileId, error) {

	// Try to read file id from file
	fileId, err := shared.ReadFileIDFromFile(path)
	if err != nil {
		// Return CRC
		crc32, err := shared.GetFileCrc32(path, maxBytes)
		log.Info("GOT CRC32: ", crc32)
		if err != nil {
			return nil, err
		}

		return &crcOrFileId{
			hasFileId: false,
			FileId:    "",
			Crc32:     crc32,
		}, nil

	}
	log.Info("GOT FILEID: ", fileId)

	return &crcOrFileId{
		hasFileId: true,
		FileId:    fileId,
		Crc32:     0,
	}, nil

}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func createFolderManifest(datasetRoot string) ([]folderFile, error) {

	skipFiles := []string{
		".DS_Store",
		".pennsieve_package",
	}

	var files []folderFile
	err := filepath.WalkDir(datasetRoot, func(p string, d os.DirEntry, err error) error {

		_, f := filepath.Split(p)
		if f == ".pennsieve" {
			return filepath.SkipDir
		} else if stringInSlice(f, skipFiles) {
			return nil
		}

		info, err := os.Stat(p)
		if err != nil {
			return err
		}
		if info.IsDir() {
			// skipping folders
			return nil
		}

		directory, fileName := filepath.Split(p)
		cleanDir := filepath.Clean(
			strings.TrimPrefix(directory, datasetRoot+string(os.PathSeparator)))

		// Clean automatically returns '.' when path is empty. In this case we do not
		// want that to happen as we are comparing to the manifest from the server
		// which does not do that.
		if cleanDir == "." {
			cleanDir = ""
		}

		curFile := folderFile{
			FileName: fileName,
			Path:     cleanDir,
			Size:     info.Size(),
		}

		files = append(files, curFile)
		return nil
	})

	return files, err

}

// compareManifestToFolder returns a list of files that are ADDED, CHANGED, MOVED, RENAMED or DELETED
// since fetching the dataset from the Pennsieve server (compare to the manifest.json file)
func compareManifestToFolder(datasetRoot string, manifest []models.ManifestDTO, files []folderFile) ([]diffResult, error) {

	//var result = api.MapDiffResponse{}
	var addedFiles []addedFile
	var deletedFiles []deletedFile
	var changedFiles []changedFile

	// Read State File which is used to determine if files are synced with server
	datasetState, err := shared.ReadStateFile(filepath.Join(datasetRoot, ".pennsieve", "state.json"))
	if err != nil {
		return nil, err
	}

	// Iterate over folder and find files that are added
FindAdded:
	for _, f := range files {
		fPath := filepath.Join(f.Path, f.FileName)
		fPathFull := filepath.Join(datasetRoot, fPath)
		for _, m := range manifest {
			if m.FileName.Valid {
				mPath := filepath.Join(m.Path, m.PackageName)
				if fPath == mPath {
					// At this point, we have a file with the expected name at a location,
					// we will check the expected size to see if something changed.
					// If the size of the file is the same as expected, we assume the file
					// is untouched.
					fi, err := os.Stat(fPathFull)
					if err != nil {
						log.Error("Cannot stat file ", fPath)
						continue
					}

					// If the size is the same as the expected size, the file is downloaded
					// and is in the expected place. No action needed.
					if fi.Size() == m.Size.Int64 {
						continue FindAdded
					}

					// At this point, we have a file with the same name at the same location,
					// but with a different size. This can either indicate a change in the
					// file, or that the file has not been downloaded yet.
					// If it is local --> something changed as file-size changed
					for _, s := range datasetState.Files {
						if s.Path == fPath && s.IsLocal {

							// File is different size at the same location and file is local
							crc32, err := shared.GetFileCrc32(fPathFull, CrcSize)
							if err != nil {
								log.Error(err)
								log.Error("Cannot get crc32 for changed file: ", fPath)
								continue
							}

							log.Warn("FIND CHANGED FILE")

							changedFiles = append(changedFiles, changedFile{
								Size:  fi.Size(),
								Crc32: crc32,
								from:  m,
							})
							continue FindAdded
						}
					}

					// At this point, we have a file with expected name,
					// at the expected location, with an unknown size as file represents remote file.
					// We assume this represents the same file as the remote file. Let's check.
					// In the small chance that we have a file with expected name, but we cannot read the
					// FileID, we know that this is an added file that has replaced the expected file with
					// that name.

					crcOrFileInfo, err := getFileIdOrCrc32(fPathFull, CrcSize)
					if err != nil {
						log.Info("Cannot find file ", err)
						return nil, err
					}

					if crcOrFileInfo.FileId != m.FileNodeId.String {

						addedF := addedFile{
							FileName:  f.FileName,
							Path:      f.Path,
							Size:      f.Size,
							hasFileId: crcOrFileInfo.hasFileId,
							Crc32:     crcOrFileInfo.Crc32,
							FileId:    crcOrFileInfo.FileId,
						}

						addedFiles = append(addedFiles, addedF)
					}

					// File represents the expected remote file on Pennsieve.
					continue FindAdded
				}
			}

		}

		// Getting CRC32 for each added file. This will be used to figure out
		// if the file was moved, renamed or truly was added to the dataset.
		// The CRC32 could be of the "empty" file, so we need to check later,
		crcOrFileInfo, err := getFileIdOrCrc32(fPathFull, CrcSize)
		if err != nil {
			return nil, err
		}

		addedF := addedFile{
			FileName:  f.FileName,
			Path:      f.Path,
			Size:      f.Size,
			hasFileId: crcOrFileInfo.hasFileId,
			Crc32:     crcOrFileInfo.Crc32,
			FileId:    crcOrFileInfo.FileId,
		}

		addedFiles = append(addedFiles, addedF)
	}

	// Iterate over manifest and find files that are deleted
FindDeleted:
	for _, m := range manifest {
		if m.FileName.Valid {
			mPath := filepath.Join(m.Path, m.PackageName)
			for _, f := range files {
				fPath := filepath.Join(f.Path, f.FileName)
				if fPath == mPath {
					continue FindDeleted
				}
			}

			log.Info("DELETED: ", mPath, "  :  ", m.FileName)

			// File in manifest is not present in the actual folder structure
			deletedFiles = append(deletedFiles, deletedFile{
				PackageNodeId: m.PackageNodeId,
				FileId:        m.FileNodeId.String,
				FileName:      m.PackageName,
				Path:          m.Path,
				Size:          m.Size.Int64,
			})
		}
	}

	// Now we need to match-up Added and Deleted entries that
	// might have been because of a:
	// 1) Move: Same name, same size, different path, same crc
	// 2) Rename: Same path, different name, same size, same crc

	var movedFiles []renamedMovedFile
	var renamedFiles []renamedMovedFile

FindMovedRenamed:
	for _, fAdded := range addedFiles {
		log.Warn("FILE:  ", fAdded.Path, "/", fAdded.FileName)

		for deletedIndex, fDeleted := range deletedFiles {

			// Skip any files in the deleted array if they have previously been matched
			if fDeleted.hasMatched {
				log.Info("SKIP MATCHED DELETED FILE")
				continue
			}

			log.Warn("DELETE:  ", fDeleted.Path, "/", fDeleted.FileName)

			if fAdded.Path == fDeleted.Path {

				log.Warn("SAME PATH BETWEEN ADDED AND DELETED: ", fDeleted.Path)
				// At this point, we have a pair of added/deleted files at in the same folder.
				// The added and deleted files have different names (as otherwise, it would have registered as Changed,
				// or Unchanged in 'FindAdded').
				//
				// We need to check the size, or fileID to figure out if this is a renamed file.

				// sanity check...
				if fAdded.FileName == fDeleted.FileName {
					log.Error("Something went wrong as file should not have been marked as deleted or added.")
				}

				// Check the state to see if the deleted entry was locally available.
				// If so, we can check to see if there is a match by file-size.
				relLocation := strings.TrimPrefix(fDeleted.Path, datasetRoot+string(os.PathSeparator))
				log.Info("Relative location: ", relLocation)
				log.Info()
				if fileIsLocalAndNotMoved(filepath.Join(relLocation, fDeleted.FileName), *datasetState) {
					log.Warn("FileLocalAndNotMoved: ", relLocation)

					// If the file size is the same, we assume that this is a renamed file.
					if fAdded.Size == fDeleted.Size {
						deletedFiles[deletedIndex].hasMatched = true
						renamedFiles = append(renamedFiles, renamedMovedFile{
							Type: api.PackageStatus_RENAMED,
							Old:  deletedFiles[deletedIndex],
							New:  fAdded,
						})

						continue FindMovedRenamed
					}

				} else {
					// File is not local
					log.Warn("File not local")

					// Try to read the fileID from the file. If this fails, we probably have a moved file.
					log.Info("read from id: ", filepath.Join(datasetRoot, fAdded.Path, fAdded.FileName))

					fileID, err := shared.ReadFileIDFromFile(filepath.Join(datasetRoot, fAdded.Path, fAdded.FileName))

					if err != nil {
						// In this case, we have a file that was marked as added,
						// and a file that was marked as deleted
						// they are in the same folder
						// the actual file is not a pulled file
						// the actual file does not contain a file ID
						// therefore, there is no action between ADDED and DELETED entries.

						log.Warn("Cannot get file id that was expected at location ", err)
						continue

					}

					if fileID == fDeleted.FileId {
						// The file was renamed
						deletedFiles[deletedIndex].hasMatched = true
						renamedFiles = append(renamedFiles, renamedMovedFile{
							Type: api.PackageStatus_RENAMED,
							Old:  deletedFiles[deletedIndex],
							New:  fAdded,
						})

						continue FindMovedRenamed
					}
				}

				continue
			}

			if fAdded.FileName == fDeleted.FileName {
				// At this point, we have a pair of files with the same name
				// but different location.

				log.Warn("FILENAME SAME -- LOCATION DIFFERENT")

				if fAdded.Path == fDeleted.Path {
					log.Error("Something went wrong as file should not have been marked as deleted or added.")
				}

				// Check state to see if added file is locally available
				// If so, we can match on file-size, if not, we match on file-id
				relLocation := strings.TrimPrefix(fDeleted.Path, datasetRoot+string(os.PathSeparator))
				local := fileIsLocalAndNotMoved(filepath.Join(relLocation, fAdded.FileName), *datasetState)
				if local {
					if fAdded.Size == fDeleted.Size {
						// The file was moved
						deletedFiles[deletedIndex].hasMatched = true
						movedFiles = append(movedFiles, renamedMovedFile{
							Type: api.PackageStatus_MOVED,
							Old:  deletedFiles[deletedIndex],
							New:  fAdded,
						})

						continue FindMovedRenamed
					}
				} else {
					fileID, err := shared.ReadFileIDFromFile(filepath.Join(datasetRoot, fAdded.Path, fAdded.FileName))

					if err != nil {
						// Likely the file is actually downloaded and moved, which is why
						// it is not found in the state file. Need to check the CRC against the
						// pulled files.

						crc, err := shared.GetFileCrc32(filepath.Join(datasetRoot, fAdded.Path, fAdded.FileName), CrcSize)
						if err != nil {
							log.Error("Unable to read ID from file", fAdded.Path)
							continue FindMovedRenamed
						}

						for _, m := range datasetState.Files {
							if m.Crc32 == crc && fDeleted.Path == m.Path {
								deletedFiles[deletedIndex].hasMatched = true
								movedFiles = append(movedFiles, renamedMovedFile{
									Type: api.PackageStatus_MOVED,
									Old:  deletedFiles[deletedIndex],
									New:  fAdded,
								})

								continue FindMovedRenamed
							}

						}

					}

					if fileID == fDeleted.FileId {
						// The file was moved
						deletedFiles[deletedIndex].hasMatched = true
						movedFiles = append(movedFiles, renamedMovedFile{
							Type: api.PackageStatus_MOVED,
							Old:  deletedFiles[deletedIndex],
							New:  fAdded,
						})

						continue FindMovedRenamed
					}

				}

			} else {
				// At this point we have a file with different names and different location
				// Potentially a RENAMED/MOVED combo.
				// We want to check the CRC to determine if these
				// files are most likely the same.

				log.Warn("POTENTIAL RENAME/MOVE combo: ", fAdded.Path, "/", fAdded.FileName)

				// If the added file has a fileId (not local), then compare fileId to deleted entries
				// If the added file has a crc32 (local), then compare to state to get packageID and
				// then deleted based on package ID.
				if fAdded.hasFileId {
					if fAdded.FileId == fDeleted.FileId {
						deletedFiles[deletedIndex].hasMatched = true
						movedFiles = append(movedFiles, renamedMovedFile{
							Type: api.PackageStatus_MOVED_RENAMED,
							Old:  deletedFiles[deletedIndex],
							New:  fAdded,
						})

						continue FindMovedRenamed
					}
				} else {
					for _, m := range datasetState.Files {
						if m.Crc32 == fAdded.Crc32 && fDeleted.FileId == m.FileId {
							deletedFiles[deletedIndex].hasMatched = true
							movedFiles = append(movedFiles, renamedMovedFile{
								Type: api.PackageStatus_MOVED_RENAMED,
								Old:  deletedFiles[deletedIndex],
								New:  fAdded,
							})
							continue FindMovedRenamed
						}
					}
				}
			}
		}
	}

	var difResults []diffResult
	// Now create resulting map.
MergeStep1:
	// FIND ADDED
	for _, aFile := range addedFiles {
		for _, rFile := range renamedFiles {

			// If a renamed file matches the current added file,
			// then add to result as a renamed file.
			if rFile.New == aFile {
				d := diffResult{
					FilePath: filepath.Join(rFile.New.Path, rFile.New.FileName),
					Type:     rFile.Type,
					Old:      rFile.Old,
					New:      rFile.New,
				}
				difResults = append(difResults, d)
				continue MergeStep1
			}
		}

		for _, mFile := range movedFiles {

			// If a moved file matches the current added File,
			// add the added file as a moved file.
			if mFile.New == aFile {
				r := diffResult{
					FilePath: filepath.Join(mFile.New.Path, mFile.New.FileName),
					Type:     mFile.Type,
					Old:      mFile.Old,
					New:      mFile.New,
				}

				difResults = append(difResults, r)
				continue MergeStep1
			}
		}

		// If the current added file is not renamed, or moved
		// then add as an added file.
		r := diffResult{
			FilePath: filepath.Join(aFile.Path, aFile.FileName),
			Type:     api.PackageStatus_ADDED,
			New:      aFile,
		}

		difResults = append(difResults, r)
	}

MergeStep2:
	// FIND DELETED
	for _, dFile := range deletedFiles {

		// Exclude deleted files that are listed in renamed
		for _, rFile := range renamedFiles {
			if rFile.Old == dFile {
				continue MergeStep2
			}
		}

		// Exclude deleted files that are listed in moved
		for _, mFile := range movedFiles {
			if mFile.Old == dFile {
				continue MergeStep2
			}
		}

		// Add other deleted files as deleted.
		r := diffResult{
			FilePath: filepath.Join(dFile.Path, dFile.FileName),
			Type:     api.PackageStatus_DELETED,
			Old:      dFile,
		}

		difResults = append(difResults, r)
	}

	// FIND CHANGED
	for _, cFile := range changedFiles {

		r := diffResult{
			FilePath: filepath.Join(cFile.from.Path, cFile.from.FileName.String),
			Type:     api.PackageStatus_CHANGED,
			Changed:  cFile,
		}

		difResults = append(difResults, r)
	}

	return difResults, nil
}

func fileIsLocalAndNotMoved(filePath string, state models2.MapState) bool {

	log.Info("FILEISLOCALANDNOTMOVED: ", filePath)
	log.Info(state)

	for _, f := range state.Files {
		log.Warn("LOCAL_NOT_MOVED: ", f.Path, "   ", filePath)
		if f.Path == filePath && f.IsLocal {
			return true
		}
	}

	return false
}
