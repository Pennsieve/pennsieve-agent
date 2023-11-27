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
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type WorkOrder struct {
	ProcessID       guuid.UUID                       `json:"ProcessID"`
	FilePath        string                           `json:"FilePath"`
	ManifestId      int32                            `json:"ManifestId"`
	WorkFlowType    pb.WorkflowResponse_WorkflowType `json:"WorkFlowType"`
	WorkOrderInput  string                           `json:"WorkOrderInput"`
	WorkOrderFiles  string                           `json:"WorkOrderFiles"`
	WorkOrderStatus bool                             `json:"WorkOrderStatus"`
	WorkFlowOutput  string                           `json:"WorkFlowOutput"`
	ManifestRoots   []string                         `json:"ManifestRoots"`
}

func isPath(path string) bool {
	if _, err := os.Stat(path); err == nil {
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
	workOrder.WorkOrderStatus = false
	workOrder.ManifestId = request.ManifestId
	workOrder.WorkOrderInput = request.WorkflowFlag
	offset := int32(0)
	limit := int32(100)

	switch isPath(workOrder.WorkOrderInput) {

	case true:
		fmt.Println("Path workflow")
		workOrder.WorkFlowType = pb.WorkflowResponse_PATH

		newJobFolder := createWorkflowFolder(workOrder)
		workOrder.FilePath = newJobFolder

		req := api.ListManifestFilesRequest{
			ManifestId: workOrder.ManifestId,
			Offset:     offset,
			Limit:      limit,
		}

		port := viper.GetString("agent.port")
		conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))

		if err != nil {
			log.Fatal(err)

		}
		defer conn.Close()

		client := api.NewAgentClient(conn)
		listFilesResponse, err := client.ListManifestFiles(context.Background(), &req)

		fmt.Println("START  CSV Gen")
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
		os.WriteFile(filepath.Dir(workOrder.WorkOrderInput)+"/nextflow.config", []byte(nextflowConfigContent), 0644)
		fmt.Println("Manifest roots")
		fmt.Println(workOrder.ManifestRoots)

		runWorkflow(&workOrder)

	case false:
		fmt.Println("Named Workflow")
		workOrder.WorkFlowType = pb.WorkflowResponse_NAMED
		workflowSteps := strings.Split(workOrder.WorkOrderInput, ",")

		for _, workflow := range workflowSteps {
			fmt.Println(workflow)
		}
	}

	response := pb.WorkflowResponse{
		Success:      workOrder.WorkOrderStatus,
		Derivatives:  "test/path",
		WorkflowType: workflowType,
	}

	return &response, nil
}

func runWorkflow(workOrder *WorkOrder) {

	writeWorkOrder(workOrder)
	app := "nextflow"
	cmd := exec.Command(app, workOrder.WorkOrderInput, "--workflowJobId", workOrder.ProcessID.String())

	output, err := cmd.Output()
	if err != nil {
		fmt.Println("failed on command execution")
		fmt.Println(string(output))

		workOrder.WorkOrderStatus = false
	} else {
		// Command completed successfully
		fmt.Println(string(output))
		workOrder.WorkOrderStatus = true
		fmt.Println("Command completed successfully")
	}

	writeWorkOrder(workOrder)
}

func writeWorkOrder(workOrder *WorkOrder) {
	// Marshal the struct into JSON
	jsonData, err := json.Marshal(workOrder)
	if err != nil {
		log.Fatal("Error marshaling JSON:", err)
	}

	// Write JSON data to a file
	err = os.WriteFile(workOrder.FilePath+"/workflow/work_order.json", jsonData, 0644)
	if err != nil {
		log.Fatal("Error writing JSON to file:", err)
	}
	fmt.Println("JSON data written to work_order.json")
}

func createWorkflowFolder(workOrder WorkOrder) string {
	// 2. Create Folder structure
	pennsieveFolder := filepath.Dir(workOrder.WorkOrderInput)
	jobsFolder := filepath.Join(pennsieveFolder, ".jobs")
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
	fmt.Println("Creating Input CSV file")

	f, err := os.Create(workOrder.FilePath + "/workflow/input.csv")
	workOrder.WorkOrderFiles = f.Name()
	w := csv.NewWriter(f)
	record := []string{"id", "source_path", "target_path"}
	err = w.Write(record)
	if err != nil {
		log.Println("Unable to write header")
	}
	for _, file := range listFilesResponse.File {
		record := []string{strconv.Itoa(int(file.Id)), file.SourcePath, file.TargetPath}
		if err := w.Write(record); err != nil {
			log.Fatalln("error writing record to file", err)
		}
	}
	w.Flush()
	f.Close()
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
