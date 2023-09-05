package workflow

import (
	"fmt"
	"os"
	"strings"
)

type Type int

const (
	Path Type = iota
	Named
)

type Workflow struct {
	WorkFlowType Type
	Workflows    []string
}

func (w *Workflow) RunWorkflow() bool {
	if w.WorkFlowType == Path {
		fmt.Println("Starting up a path workflow")
	} else {
		fmt.Println("Starting up a named workflow")
		fmt.Println(w.Workflows)
	}
	return true
}

func NewWorkflow(workflowArg string) *Workflow {
	var workflowType Type
	var workflowSteps []string

	if isPath(workflowArg) {
		workflowType = Path
	} else {
		workflowType = Named
		workflowSteps = strings.Split(workflowArg, ",")
	}
	return &Workflow{
		WorkFlowType: workflowType,
		Workflows:    workflowSteps,
	}
}

func isPath(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
