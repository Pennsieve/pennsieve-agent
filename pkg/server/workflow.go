package server

import (
	"context"
	"fmt"
	pb "github.com/pennsieve/pennsieve-agent/api/v1"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
)

type Type int

const (
	Path Type = iota
	Named
)

//	type Workflow struct {
//		WorkFlowType Type
//		Workflows    []string
//	}
//
//	func (w *Workflow) RunWorkflow() bool {
//		if w.WorkFlowType == Path {
//			fmt.Println("Starting up a path workflow")
//		} else {
//			fmt.Println("Starting up a named workflow")
//			fmt.Println(w.Workflows)
//		}
//		return true
//	}
//
//	func NewWorkflow(workflowArg string) *Workflow {
//		var workflowType Type
//		var workflowSteps []string
//
//		if isPath(workflowArg) {
//			workflowType = Path
//		} else {
//			workflowType = Named
//			workflowSteps = strings.Split(workflowArg, ",")
//		}
//		return &Workflow{
//			WorkFlowType: workflowType,
//			Workflows:    workflowSteps,
//		}
//	}
func isPath(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func (s *server) StartWorkflow(ctx context.Context, request *pb.StartWorkflowRequest) (*pb.WorkflowResponse, error) {

	fmt.Println("\nStarting workflow")
	var successStatus bool

	var workflowSteps []string
	var workflowType Type

	manifestId := request.ManifestId
	workflowFlag := request.WorkflowFlag

	switch isPath(workflowFlag) {

	case true:
		fmt.Println("Path workflow")
		workflowType = Path

		var out strings.Builder

		cmd := exec.Command("nextflow")
		cmd.Args = append(cmd.Args, workflowFlag)
		cmd.Stdout = &out
		err := cmd.Run()

		if err != nil {
			log.Fatal(err)
			successStatus = false
		}

		successStatus = true
		fmt.Printf("Result: %q\n", out.String())

	case false:
		fmt.Println("Named Workflow")
		workflowType = Named
		workflowSteps = strings.Split(workflowFlag, ",")

		for _, workflow := range workflowSteps {
			fmt.Println(workflow)
		}
	}

	fmt.Println(manifestId)
	fmt.Println(workflowType)

	response := pb.WorkflowResponse{
		Success:     successStatus,
		Derivatives: "test/path",
	}

	return &response, nil
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
