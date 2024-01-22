package server

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	guuid "github.com/google/uuid"
	api "github.com/pennsieve/pennsieve-agent/api/v1"
	pb "github.com/pennsieve/pennsieve-agent/api/v1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

type WorkOrder struct {
	ProcessID          guuid.UUID                       `json:"ProcessID"`
	FilePath           string                           `json:"FilePath"`
	ManifestID         int32                            `json:"ManifestID"`
	WorkFlowType       pb.WorkflowResponse_WorkflowType `json:"WorkFlowType"`
	Input              string                           `json:"Input"`
	Files              string                           `json:"Files"`
	Status             bool                             `json:"Status"`
	WorkFlowOutput     string                           `json:"WorkFlowOutput"`
	ManifestRoots      []string                         `json:"ManifestRoots"`
	NextflowConfigFile string                           `json:"NextflowConfigFile"`
}

func isPath(path string) bool {
	_, err := os.Stat(path)

	if errors.Is(err, fs.ErrNotExist) {
		fmt.Printf("Path error: %v", err)
	}
	if err == nil {
		return true
	}

	return false
}

func (s *server) StartWorkflow(ctx context.Context, request *pb.StartWorkflowRequest) (*pb.WorkflowResponse, error) {

	id := guuid.New()
	fmt.Println("\nStarting workflow")
	fmt.Println("ID:" + id.String())

	var workOrder WorkOrder
	var workflowType pb.WorkflowResponse_WorkflowType

	workOrder.ProcessID = id
	workOrder.Status = false
	workOrder.ManifestID = request.ManifestId
	workOrder.Input = request.WorkflowFlag
	offset := int32(0)
	limit := int32(100)

	switch isPath(workOrder.Input) {

	case true:
		workOrder.WorkFlowType = pb.WorkflowResponse_PATH

		newJobFolder := createWorkflowFolder(workOrder)
		workOrder.FilePath = newJobFolder

		req := api.ListManifestFilesRequest{
			ManifestId: workOrder.ManifestID,
			Offset:     offset,
			Limit:      limit,
		}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))

		if err != nil {
			fmt.Printf("%v", err)

		}
		defer conn.Close()

		client := api.NewAgentClient(conn)
		listFilesResponse, err := client.ListManifestFiles(context.Background(), &req)

		createInputCSV(&workOrder, listFilesResponse)
		workOrder.ManifestRoots = getRootDirectories(listFilesResponse)

		nextflowConfigContent := "" +
			"process.failFast = true\n" +
			"process.containerOptions = '--platform linux/amd64 --rm " +
			"-v " + workOrder.ManifestRoots[0] + ":/data " +
			"-v " + workOrder.FilePath + ":/job'" +
			"\ndocker{" +
			"\n    enabled = true" +
			"\n}"

		workOrder.NextflowConfigFile = filepath.Dir(newJobFolder) + "/" + workOrder.ProcessID.String() + "/nextflow.config"
		err = os.WriteFile(workOrder.NextflowConfigFile, []byte(nextflowConfigContent), 0644)

		runWorkflow(&workOrder)

	case false:
		fmt.Println("Named Workflow")
		workOrder.WorkFlowType = pb.WorkflowResponse_NAMED
		workflowSteps := strings.Split(workOrder.Input, ",")

		for _, workflow := range workflowSteps {
			log.Println(workflow)
		}
	}

	response := pb.WorkflowResponse{
		Success:      workOrder.Status,
		Derivatives:  "~/.pennsieve/derivatives",
		WorkflowType: workflowType,
	}

	return &response, nil
}

func runWorkflow(workOrder *WorkOrder) {

	writeWorkOrder(workOrder)

	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current working directory: %v", err)
		return
	}

	targetFolder := "workflow"
	targetFile := "master_workflow.nf"
	targetPath := filepath.Join(currentDir, targetFolder, targetFile)

	app := "nextflow"
	cmd := exec.Command(app, targetPath, "--workflowJobId", workOrder.ProcessID.String(), "--userJob", workOrder.Input, "-c", workOrder.NextflowConfigFile)

	output, err := cmd.Output()
	fmt.Println(string(output))
	if err != nil {
		fmt.Println("failed on command execution")

		workOrder.Status = false
	} else {
		// Command completed successfully
		workOrder.Status = true
		fmt.Println("Command completed successfully")
	}

	writeWorkOrder(workOrder)
}

func writeWorkOrder(workOrder *WorkOrder) {
	// Marshal the struct into JSON
	jsonData, err := json.Marshal(workOrder)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v", err)
	}

	// Write JSON data to a file
	err = os.WriteFile(workOrder.FilePath+"/workflow/work_order.json", jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing JSON to file: %v", err)
	}
}

func createWorkflowFolder(workOrder WorkOrder) string {
	// 2. Create Folder structure
	currentUserHomeFolder, err := user.Current()
	if err != nil {
		fmt.Printf("Error in getting home folder: %v", err)
	}

	jobsFolder := filepath.Join(currentUserHomeFolder.HomeDir, ".pennsieve/.jobs")
	if _, err := os.Stat(jobsFolder); errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(jobsFolder, os.ModePerm); err != nil {
			log.Fatal(err)
		}
	}
	newJobFolder := filepath.Join(jobsFolder, workOrder.ProcessID.String())
	if err := os.Mkdir(newJobFolder, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	os.Mkdir(filepath.Join(newJobFolder, "workflow"), os.ModePerm)
	os.Mkdir(filepath.Join(newJobFolder, ".derivatives"), os.ModePerm)

	return newJobFolder
}

func createInputCSV(workOrder *WorkOrder, listFilesResponse *api.ListManifestFilesResponse) {

	f, err := os.Create(workOrder.FilePath + "/workflow/input.csv")
	workOrder.Files = f.Name()
	w := csv.NewWriter(f)
	record := []string{"id", "source_path", "target_path"}
	err = w.Write(record)
	if err != nil {
		fmt.Println("Unable to write header")
	}
	for _, file := range listFilesResponse.File {
		record := []string{strconv.Itoa(int(file.Id)), file.SourcePath, file.TargetPath}
		if err := w.Write(record); err != nil {
			fmt.Printf("error writing record to file: %v", err)
		}
	}
	w.Flush()
	err = f.Close()
	if err != nil {
		fmt.Printf("Error closing file stream: %v", err)
		return
	}
}

func getRootDirectories(listFilesResponse *api.ListManifestFilesResponse) []string {

	var dirs []string
	// Push all paths into array
	for _, file := range listFilesResponse.File {
		dirs = append(dirs, filepath.Dir(file.SourcePath))
	}
	// Remove duplicates
	var uniqueDirPaths []string
	seen := map[string]bool{}
	for _, dir := range dirs {
		if seen[dir] == false {
			seen[dir] = true
			uniqueDirPaths = append(uniqueDirPaths, dir)
		}
	}

	// Check for highest level folder
	var skips []int
	rootsMap := map[string]bool{}
	for i := 0; i < len(uniqueDirPaths); i++ {
		if slices.Contains(skips, i) {
			continue
		}
		for j := 0; j < len(uniqueDirPaths); j++ {
			if strings.Contains(uniqueDirPaths[j], uniqueDirPaths[i]) {
				rootsMap[uniqueDirPaths[i]] = true
				skips = append(skips, j)
			}
		}
	}

	// Pull dirs out of keys
	rootDirs := make([]string, len(rootsMap))
	i := 0
	for k := range rootsMap {
		rootDirs[i] = k
		i++
	}
	return rootDirs
}

// find common parts of path between 2 file paths
func commonPathParts(path1 string, path2 string) []string {
	parts1 := strings.Split(path1, "/")
	parts2 := strings.Split(path2, "/")

	var commonParts []string
	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		if parts1[i] == parts2[i] {
			commonParts = append(commonParts, parts1[i])
		} else {
			continue
		}
	}
	return commonParts
}
