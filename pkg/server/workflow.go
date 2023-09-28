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
	ManifestId      int32                            `json:"ManifestId"`
	WorkFlowType    pb.WorkflowResponse_WorkflowType `json:"WorkFlowType"`
	WorkOrderInput  string                           `json:"WorkOrderInput"`
	WorkOrderStatus bool                             `json:"WorkOrderStatus"`
}

func isPath(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func (s *server) StartWorkflow(ctx context.Context, request *pb.StartWorkflowRequest) (*pb.WorkflowResponse, error) {

	fmt.Println("\nStarting workflow")

	var workOrder WorkOrder
	var workflowType pb.WorkflowResponse_WorkflowType

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

		req := api.ListManifestFilesRequest{
			ManifestId: workOrder.ManifestId,
			Offset:     offset,
			Limit:      limit,
		}
		createInputCSV(workOrder, req, newJobFolder)

		runWorkflow(workOrder)

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

func runWorkflow(workOrder WorkOrder) {
	app := "nextflow"
	cmd := exec.Command(app, workOrder.WorkOrderInput)

	//err := cmd.Run()
	output, err := cmd.Output()
	if err != nil {
		// Check if the error is of type ExitError, which contains exit status information
		if exitErr, ok := err.(*exec.ExitError); ok {
			// The command exited with a non-zero status
			workOrder.WorkOrderStatus = false
			fmt.Printf("Command finished with error: %v\n", exitErr)
			fmt.Printf("Exit status: %d\n", exitErr.ExitCode())
			fmt.Println(err.Error())
			fmt.Print(string(output))
		} else {
			// Some other error occurred
			fmt.Printf("Command finished with error: %v\n", err)
		}
	} else {
		// Command completed successfully
		fmt.Println("Command completed successfully")
	}
}

func createWorkflowFolder(workOrder WorkOrder) string {
	workOrder.ProcessID = guuid.New()
	// 2. Create Folder structure
	home, _ := os.UserHomeDir()
	pennsieveFolder := filepath.Join(home, ".pennsieve")
	jobsFolder := filepath.Join(pennsieveFolder, ".jobs")
	// Create '~/.pennsieve/jobs' folder if it does not exist.
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

	// 3. Create workorder.json file

	// Marshal the struct into JSON
	jsonData, err := json.Marshal(workOrder)
	if err != nil {
		log.Fatal("Error marshaling JSON:", err)
		return ""
	}

	// Write JSON data to a file
	err = os.WriteFile(newJobFolder+"/workflow/work_order.json", jsonData, 0644)
	if err != nil {
		log.Fatal("Error writing JSON to file:", err)
		return ""
	}
	fmt.Println("JSON data written to work_order.json")
	return newJobFolder
}

func createInputCSV(workOrder WorkOrder, req api.ListManifestFilesRequest, newJobFolder string) {
	fmt.Println("Creating Input CSV file")

	port := viper.GetString("agent.port")
	conn, err := grpc.Dial(":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatal(err)

	}
	defer conn.Close()

	client := api.NewAgentClient(conn)
	listFilesResponse, err := client.ListManifestFiles(context.Background(), &req)

	f, err := os.Create(newJobFolder + "/workflow/input.csv")
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

func isCommandAvailable(command string) bool {
	// Run the "command --version" to check if it's available
	cmd := exec.Command(command, "--version")

	// Capture the output of the command
	output, err := cmd.CombinedOutput()

	// Check if the command executed successfully
	if err != nil || strings.Contains(string(output), "not found") {
		return false
	}

	return true
}
