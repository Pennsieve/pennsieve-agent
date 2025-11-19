package _map

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	api "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/cmd/shared"
	workspace "github.com/pennsieve/pennsieve-agent/pkg/shared"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var PushCmd = &cobra.Command{
	Use:   "push [target_path]",
	Short: "Push local changes to the remote Pennsieve Dataset",
	Long: `
  [BETA] This feature is in Beta mode and is currently still undergoing
  testing and optimization.

  `,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		// Files to ignore during push
		ignoredFiles := []string{
			".DS_Store",
		}

		// Determine the target folder
		var folder string
		if len(args) > 0 {
			folder = args[0]
		} else {
			folder = "."
		}

		// Check and make path absolute
		absPath, err := shared.GetAbsolutePath(folder)
		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, fmt.Sprintf("Error: Unable to parse provided path: %v", err))
			return
		}

		//1. Lookup manifest.json in .pennsieve folder
		manifestPath, err := workspace.FindManifestFile(absPath)
		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, "Error: Unable to find .pennsieve/manifest.json file in this or parent directories")
			return
		}

		// Get the dataset root directory
		datasetRoot := filepath.Dir(filepath.Dir(manifestPath))

		//2. Build up a list of files from manifest.json (files that exist in the dataset)
		workspaceManifest, err := workspace.ReadWorkspaceManifest(manifestPath)
		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, "Error: Unable to read manifest.json file")
			return
		}

		// Build a map of files that exist in the manifest for quick lookup
		manifestFiles := make(map[string]bool)
		for _, file := range workspaceManifest.Files {
			if file.FileName.Valid {
				// Construct the full absolute path for this file
				absFilePath := filepath.Join(datasetRoot, file.Path, file.FileName.String)
				manifestFiles[absFilePath] = true
			}
		}

		//3. Check for any new files / folders
		var newFiles []string
		err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip the .pennsieve folder
			if strings.Contains(path, ".pennsieve") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip ignored files
			for _, ignored := range ignoredFiles {
				if info.Name() == ignored {
					return nil
				}
			}

			// Only process files (not directories)
			if !info.IsDir() {
				// Check if this file exists in the manifest
				if !manifestFiles[path] {
					newFiles = append(newFiles, path)
				}
			}

			return nil
		})

		if err != nil {
			fmt.Println(err)
			shared.HandleAgentError(err, "Error: Unable to scan directory for new files")
			return
		}

		if len(newFiles) == 0 {
			fmt.Println("No new files to push.")
			return
		}

		// Display files to user
		fmt.Printf("Found %d new file(s) to push:\n", len(newFiles))
		for _, file := range newFiles {
			relPath, err := filepath.Rel(datasetRoot, file)
			if err != nil {
				log.Errorf("Error computing relative path for %s: %v", file, err)
				continue
			}
			fmt.Printf("  - %s\n", relPath)
		}

		//4. Create a manifest and add files to it
		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to GRPC Server: ", err)
			return
		}
		defer conn.Close()

		client := api.NewAgentClient(conn)

		// Create an empty manifest first
		// Note: Cannot specify both BasePath and Files - must create empty manifest then add files
		createReq := api.CreateManifestRequest{
			BasePath:       "",
			TargetBasePath: "",
			Recursive:      false,
			Files:          nil,
		}

		manifestResponse, err := client.CreateManifest(context.Background(), &createReq)
		if err != nil {
			log.Errorf("Error creating manifest: %v", err)
			shared.HandleAgentError(err, "Error: Unable to create manifest for new files")
			return
		}

		fmt.Printf("\nCreated manifest %d\n", manifestResponse.ManifestId)

		// Add each file to the manifest individually with its correct target path
		// We add each file with its directory as the target to preserve structure
		for _, filePath := range newFiles {
			// Get the relative path from dataset root
			relPath, err := filepath.Rel(datasetRoot, filePath)
			if err != nil {
				log.Errorf("Error computing relative path for %s: %v", filePath, err)
				continue
			}

			// The target path is the directory part of the relative path
			targetPath := filepath.Dir(relPath)
			if targetPath == "." {
				targetPath = ""
			}
			// Convert to forward slashes for target path
			targetPath = filepath.ToSlash(targetPath)

			// Add this file to the manifest
			addReq := api.AddToManifestRequest{
				ManifestId:     manifestResponse.ManifestId,
				BasePath:       filePath,
				TargetBasePath: targetPath,
				Recursive:      false,
				Files:          nil,
			}

			_, err = client.AddToManifest(context.Background(), &addReq)
			if err != nil {
				log.Errorf("Error adding file %s to manifest: %v", relPath, err)
				continue
			}
		}

		fmt.Printf("Added %d file(s) to manifest %d\n", len(newFiles), manifestResponse.ManifestId)

		// Upload the manifest
		uploadReq := api.UploadManifestRequest{
			ManifestId: manifestResponse.ManifestId,
		}

		_, err = client.UploadManifest(context.Background(), &uploadReq)
		if err != nil {
			log.Errorf("Error uploading manifest %d: %v", manifestResponse.ManifestId, err)
			shared.HandleAgentError(err, "Error: Unable to upload manifest")
			return
		}

		fmt.Printf("Upload initiated for manifest %d\n", manifestResponse.ManifestId)
		fmt.Println("\nPush complete. Files are being uploaded in the background.")
		fmt.Println("Use \"pennsieve agent subscribe\" to track progress of the uploaded files.")
	},
}

func init() {

}
