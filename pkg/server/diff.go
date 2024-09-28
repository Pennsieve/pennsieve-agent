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
	"path"
	"path/filepath"
	"strings"
)

func (s *server) GetMapDiff(ctx context.Context, req *api.MapDiffRequest) (*api.MapDiffResponse, error) {

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

	manifest, err := shared.ReadWorkspaceManifest(path.Join(datasetRoot, ".pennsieve", "manifest.json"))
	if err != nil {
		return nil, err
	}

	result, err := compareManifestToFolder(datasetRoot, manifest.Files, files)
	if err != nil {
		return nil, err
	}

	return result, nil
}

type folderManifestFile struct {
	PackageNodeId string
	FileId        string
	FileName      string
	Path          string
	Size          int64
	Crc32         uint32
}

type renamedMovedFile struct {
	Old folderManifestFile
	New folderManifestFile
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func createFolderManifest(datasetRoot string) ([]folderManifestFile, error) {

	skipFiles := []string{
		".DS_Store",
		".pennsieve_package",
	}

	var files []folderManifestFile
	err := filepath.WalkDir(datasetRoot, func(p string, d os.DirEntry, err error) error {

		_, f := path.Split(p)
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

		directory, fileName := path.Split(p)
		cleanDir := path.Clean(
			strings.TrimPrefix(directory, datasetRoot+string(os.PathSeparator)))

		// Clean automatically returns '.' when path is empty. In this case we do not
		// want that to happen as we are comparing to the manifest from the server
		// which does not do that.
		if cleanDir == "." {
			cleanDir = ""
		}

		curFile := folderManifestFile{
			PackageNodeId: "",
			FileId:        "",
			FileName:      fileName,
			Path:          cleanDir,
			Size:          info.Size(),
		}

		files = append(files, curFile)
		return nil
	})

	return files, err

}

// compareManifestToFolder returns a list of files that are ADDED, CHANGED, MOVED, RENAMED or DELETED
// since fetching the dataset from the Pennsieve server (compare to the manifest.json file)
func compareManifestToFolder(datasetRoot string, manifest []models.ManifestDTO, files []folderManifestFile) (*api.MapDiffResponse, error) {

	var result = api.MapDiffResponse{}
	var addedFiles []folderManifestFile
	var deletedFiles []folderManifestFile
	var changedFiles []folderManifestFile

	// Read State File which is used to determine if files are synced with server
	datasetState, err := shared.ReadStateFile(path.Join(datasetRoot, ".pennsieve", "state.json"))
	if err != nil {
		return nil, err
	}

	// Iterate over folder and find files that are added
FindAdded:
	for _, f := range files {
		fPath := path.Join(f.Path, f.FileName)
		fPathFull := path.Join(datasetRoot, fPath)
		for _, m := range manifest {
			if m.FileName.Valid {
				mPath := path.Join(m.Path, m.PackageName)
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

					// If the size is the same as the expected size, the file is downloadeded
					// and is in the expected place. No action needed.
					if fi.Size() == m.Size.Int64 {
						continue FindAdded
					}

					// At this point, we have a file with the same name at the same location,
					// but with a different size. This can either indicate a change in the
					// file, or that the file has not been downloaded yet.
					// If it is local --> something changed as file-size changed
					for _, s := range datasetState.Files {
						if s.Path == fPathFull && s.IsLocal {

							// File is different size at the same location and file is local
							changedFiles = append(changedFiles, f)
							continue FindAdded
						}
					}

					// At this point, we have a file with expected name,
					// at the expected location, with an unknown size as file represents remote file.
					// We assume this represents the same file as the remote file. Let's check.
					// In the small chance that we have a file with expected name but we cannot read the
					// FileID, we know that this is an added file that has replaced the expected file with
					// that name.

					fileId, err := shared.ReadFileIDFromFile(fPathFull)
					if fileId != m.FileNodeId.String || err != nil {
						log.Info("Cannot find file ", err)
						addedFiles = append(addedFiles, f)
					}

					// File represents the expected remote file on Pennsieve.
					continue FindAdded
				}
			}

		}

		// Getting CRC32 for each added file. This will be used to figure out
		// if the file was moved, renamed or truly was added to the datset.
		// The CRC32 could be of the "empty" file so we need to check later,
		f.Crc32, _ = shared.GetFileCrc32(fPath, 1024*1024)
		addedFiles = append(addedFiles, f)
	}

	// Iterate over manifest and find files that are deleted
FindDeleted:
	for _, m := range manifest {
		if m.FileName.Valid {
			mPath := path.Join(m.Path, m.PackageName)
			for _, f := range files {
				fPath := path.Join(f.Path, f.FileName)
				if fPath == mPath {
					continue FindDeleted
				}
			}

			// File in manifest is not present in the actual folder structure
			deletedFiles = append(deletedFiles, folderManifestFile{
				PackageNodeId: m.PackageNodeId,
				FileId:        m.FileNodeId.String,
				FileName:      m.PackageName,
				Path:          m.Path,
				Size:          m.Size.Int64,
				Crc32:         0,
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
		for _, fDeleted := range deletedFiles {
			if fAdded.Path == fDeleted.Path {
				// At this point, we have a pair of added/deleted files at in the same folder.
				// The added and deleted files have different names (as otherwise, it would have registered as Changed,
				// or Unchanged in 'FindAdded').
				//
				// We need to check the size, or fileID to figure out if this is a renamed file.

				// sanity check...
				if fAdded.FileName == fDeleted.FileName {
					log.Error("Something went wrong as file should not have been marked as deleted or added.")
				}

				// Check state to see if added file is locally available
				// If so, we can match on file-size, if not, we match on file-id
				if fileIsLocalAndNotMoved(fAdded.Path, *datasetState) {

					// If the file size is the same, we assume that this is a renamed file.
					if fAdded.Size == fDeleted.Size {
						renamedFiles = append(renamedFiles, renamedMovedFile{
							Old: fDeleted,
							New: fAdded,
						})
						continue FindMovedRenamed
					}
				} else {
					// File is not local, or is a local file that has been moved.

					// Try to read the fileID from the file. If this fails, we probably have a moved file.
					fileID, err := shared.ReadFileIDFromFile(path.Join(datasetRoot, fAdded.Path, fAdded.FileName))

					if err != nil {

						// We failed to get the fileID from the file. Now, let's get the CRC32 and see if we can
						// match this against known downloaded files in the DatasetState.
						//crc, err := shared.GetFileCrc32(path.Join(datasetRoot, fAdded.Path, fAdded.FileName), 1024*1024)
						//if err != nil {
						//	log.Error("Unable to read ID from file", fAdded.Path)
						//	continue
						//}

						for _, stateFile := range datasetState.Files {
							if stateFile.Crc32 == fAdded.Crc32 && stateFile.Path == fDeleted.Path {
								renamedFiles = append(renamedFiles, renamedMovedFile{
									Old: fDeleted,
									New: fAdded,
								})
								continue FindMovedRenamed
							}
						}
					} else if fileID == fDeleted.FileId {
						// The file was renamed
						renamedFiles = append(renamedFiles, renamedMovedFile{
							Old: fDeleted,
							New: fAdded,
						})

						continue FindMovedRenamed
					}

				}

			} else if fAdded.FileName == fDeleted.FileName {
				// At this point, we have a pair of files with the same name
				// but different location.

				if fAdded.Path == fDeleted.Path {
					log.Error("Something went wrong as file should not have been marked as deleted or added.")
				}

				// Check state to see if added file is locally available
				// If so, we can match on file-size, if not, we match on file-id
				local := fileIsLocalAndNotMoved(fAdded.Path, *datasetState)
				if local {
					if fAdded.Size == fDeleted.Size {
						// The file was moved
						movedFiles = append(movedFiles, renamedMovedFile{
							Old: fDeleted,
							New: fAdded,
						})

						continue FindMovedRenamed
					}
				} else {
					fileID, err := shared.ReadFileIDFromFile(path.Join(datasetRoot, fAdded.Path, fAdded.FileName))

					if err != nil {
						// Likely the file is actually downloaded and moved, which is why
						// it is not found in the state file. Need to check the CRC against the
						// pulled files.

						crc, err := shared.GetFileCrc32(path.Join(datasetRoot, fAdded.Path, fAdded.FileName), 1024*1024)
						if err != nil {
							log.Error("Unable to read ID from file", fAdded.Path)
							continue FindMovedRenamed
						}

						for _, m := range datasetState.Files {
							if m.Crc32 == crc {
								movedFiles = append(movedFiles, renamedMovedFile{
									Old: fDeleted,
									New: fAdded,
								})
								continue FindMovedRenamed
							}

						}

					}

					if fileID == fDeleted.FileId {
						// The file was renamed
						movedFiles = append(movedFiles, renamedMovedFile{
							Old: fDeleted,
							New: fAdded,
						})

						continue FindMovedRenamed
					}

				}

			} else {
				// At this point we have a file with different names and different location
				// We want to check the CRC to determine if these
				// files are most likely the same.

				crc, err := shared.GetFileCrc32(path.Join(datasetRoot, fAdded.Path, fAdded.FileName), 1024*1024)
				if err != nil {
					log.Error("Unable to read ID from file", fAdded.Path)
					continue FindMovedRenamed
				}
				//
				for _, m := range datasetState.Files {
					if m.Crc32 == crc {
						log.Info("MOVED: ", fAdded.Path)
						movedFiles = append(movedFiles, renamedMovedFile{
							Old: fDeleted,
							New: fAdded,
						})
						continue FindMovedRenamed
					}
				}

			}
		}
	}

	// Now create resulting map.
MergeStep1:
	// FIND ADDED
	for _, aFile := range addedFiles {
		for _, rFile := range renamedFiles {

			// If a renamed file matches the current added file,
			// then add to result as a renamed file.
			if rFile.New == aFile {
				result.Files = append(result.Files, &api.PackageStatus{
					Content: &api.FileInfo{
						PackageId: aFile.PackageNodeId,
						Path:      aFile.Path,
						Name:      aFile.FileName,
						Message:   "",
					},
					ChangeType: api.PackageStatus_RENAMED,
				})
				continue MergeStep1
			}
		}

		for _, mFile := range movedFiles {

			// If a moved file matches the current added File,
			// add the added file as a moved file.
			if mFile.New == aFile {
				result.Files = append(result.Files, &api.PackageStatus{
					Content: &api.FileInfo{
						PackageId: aFile.PackageNodeId,
						Path:      aFile.Path,
						Name:      aFile.FileName,
						Message:   "",
					},
					ChangeType: api.PackageStatus_MOVED,
				})
				continue MergeStep1
			}
		}

		// If the current added file is not renamed, or moved
		// then add as an added file.
		result.Files = append(result.Files, &api.PackageStatus{
			Content: &api.FileInfo{
				PackageId: aFile.PackageNodeId,
				Path:      aFile.Path,
				Name:      aFile.FileName,
				Message:   "",
			},
			ChangeType: api.PackageStatus_ADDED,
		})
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
		result.Files = append(result.Files, &api.PackageStatus{
			Content: &api.FileInfo{
				PackageId: dFile.PackageNodeId,
				Path:      dFile.Path,
				Name:      dFile.FileName,
				Message:   "",
			},
			ChangeType: api.PackageStatus_DELETED,
		})
	}

	// FIND CHANGED
	for _, dFile := range changedFiles {

		// Add other deleted files as deleted.
		result.Files = append(result.Files, &api.PackageStatus{
			Content: &api.FileInfo{
				PackageId: dFile.PackageNodeId,
				Path:      dFile.Path,
				Name:      dFile.FileName,
				Message:   "",
			},
			ChangeType: api.PackageStatus_CHANGED,
		})
	}

	return &result, nil
}

func fileIsLocalAndNotMoved(filePath string, state models2.MapState) bool {

	for _, f := range state.Files {
		if f.Path == filePath && f.IsLocal {
			return true
		}
	}

	return false
}
