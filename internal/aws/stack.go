package aws

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// StackRegion is where the compute role's stack is created: IAM is global,
// so the region only places the stack, and Pennsieve's role setups name this
// one.
const StackRegion = "us-east-1"

// runAWS runs the aws CLI and returns its stdout, or its stderr as the error.
// Tests replace it.
var runAWS = func(args ...string) (string, error) {
	cmd := exec.Command("aws", args...)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}
	return strings.TrimSpace(outb.String()), nil
}

// StackManager creates, updates and deletes the compute role's CloudFormation
// stack in the user's account, with their aws CLI profile.
type StackManager struct {
	Profile string
	Region  string
	// Poll is how often Wait checks the stack, and Timeout how long it waits.
	Poll    time.Duration
	Timeout time.Duration
	// Progress hears each status the stack goes through.
	Progress func(status string)
}

func NewStackManager(profile string) *StackManager {
	return &StackManager{Profile: profile, Region: StackRegion, Poll: 5 * time.Second, Timeout: 15 * time.Minute,
		Progress: func(status string) { log.Printf("CloudFormation stack: %s", status) }}
}

func (s *StackManager) aws(args ...string) (string, error) {
	base := []string{"--profile", s.Profile, "--region", s.Region, "cloudformation"}
	return runAWS(append(base, args...)...)
}

// stackParameters is --parameters for the role template's three parameters.
func stackParameters(roleName, externalId, pennsieveAccountId string) []string {
	return []string{
		"ParameterKey=RoleName,ParameterValue=" + roleName,
		"ParameterKey=ExternalId,ParameterValue=" + externalId,
		"ParameterKey=PennsieveAccountId,ParameterValue=" + pennsieveAccountId,
	}
}

// Create starts creating the stack (named after the role) from templateURL.
func (s *StackManager) Create(stackName, templateURL, roleName, externalId, pennsieveAccountId string) error {
	args := []string{"create-stack", "--stack-name", stackName, "--template-url", templateURL, "--parameters"}
	args = append(args, stackParameters(roleName, externalId, pennsieveAccountId)...)
	args = append(args, "--capabilities", "CAPABILITY_NAMED_IAM")
	if _, err := s.aws(args...); err != nil {
		return fmt.Errorf("creating CloudFormation stack %s: %w", stackName, err)
	}
	return nil
}

// Update starts updating the stack to templateURL, keeping its parameters.
// updated is false when the stack already is that template ("No updates are
// to be performed").
func (s *StackManager) Update(stackName, templateURL string) (updated bool, err error) {
	args := []string{"update-stack", "--stack-name", stackName, "--template-url", templateURL, "--parameters",
		"ParameterKey=RoleName,UsePreviousValue=true",
		"ParameterKey=ExternalId,UsePreviousValue=true",
		"ParameterKey=PennsieveAccountId,UsePreviousValue=true",
		"--capabilities", "CAPABILITY_NAMED_IAM"}
	if _, err := s.aws(args...); err != nil {
		if strings.Contains(err.Error(), "No updates are to be performed") {
			return false, nil
		}
		return false, fmt.Errorf("updating CloudFormation stack %s: %w", stackName, err)
	}
	return true, nil
}

// Delete starts deleting the stack, and the role with it.
func (s *StackManager) Delete(stackName string) error {
	if _, err := s.aws("delete-stack", "--stack-name", stackName); err != nil {
		return fmt.Errorf("deleting CloudFormation stack %s: %w", stackName, err)
	}
	return nil
}

// Exists reports whether there's a stack by that name (not deleted).
func (s *StackManager) Exists(stackName string) bool {
	status, err := s.status(stackName)
	return err == nil && status != "" && status != "DELETE_COMPLETE"
}

func (s *StackManager) status(stackName string) (string, error) {
	return s.aws("describe-stacks", "--stack-name", stackName, "--query", "Stacks[0].StackStatus", "--output", "text")
}

// Wait waits for the stack to reach want (CREATE_COMPLETE, UPDATE_COMPLETE),
// telling Progress each status on the way. A stack that fails or rolls back
// comes back as an error with CloudFormation's reasons.
func (s *StackManager) Wait(stackName, want string) error {
	deadline := time.Now().Add(s.Timeout)
	last := ""
	for {
		status, err := s.status(stackName)
		if err != nil {
			// A deleted stack is soon gone from describe-stacks altogether.
			if want == "DELETE_COMPLETE" && strings.Contains(err.Error(), "does not exist") {
				return nil
			}
			return fmt.Errorf("checking CloudFormation stack %s: %w", stackName, err)
		}
		if status != last {
			last = status
			if s.Progress != nil {
				s.Progress(status)
			}
		}
		switch stackState(status, want) {
		case stackDone:
			return nil
		case stackFailed:
			reasons := s.failureReasons(stackName)
			if reasons == "" {
				return fmt.Errorf("CloudFormation stack %s is %s", stackName, status)
			}
			return fmt.Errorf("CloudFormation stack %s is %s: %s", stackName, status, reasons)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("CloudFormation stack %s is still %s after %s; check it in the AWS console", stackName, status, s.Timeout)
		}
		time.Sleep(s.Poll)
	}
}

type stackProgress int

const (
	stackInProgress stackProgress = iota
	stackDone
	stackFailed
)

// stackState is where a stack in status is, on its way to want.
func stackState(status, want string) stackProgress {
	switch {
	case status == want:
		return stackDone
	case strings.HasPrefix(status, "DELETE_") && !strings.HasPrefix(want, "DELETE_"):
		return stackFailed
	case strings.HasSuffix(status, "_IN_PROGRESS") && !strings.Contains(status, "ROLLBACK"):
		// Including UPDATE_COMPLETE_CLEANUP_IN_PROGRESS on the way to UPDATE_COMPLETE.
		return stackInProgress
	default:
		// FAILED, ROLLBACK_*, DELETE_*, or a complete status that isn't want.
		return stackFailed
	}
}

// failureReasons is what CloudFormation said about the resources that failed,
// one "resource: reason" per line, newest first.
func (s *StackManager) failureReasons(stackName string) string {
	out, err := s.aws("describe-stack-events", "--stack-name", stackName,
		"--query", "StackEvents[?contains(ResourceStatus, 'FAILED')].[LogicalResourceId, ResourceStatusReason]",
		"--output", "text")
	if err != nil {
		return ""
	}
	return formatFailureReasons(out)
}

// formatFailureReasons turns describe-stack-events' text rows (resource, tab,
// reason) into "resource: reason; …", skipping repeats.
func formatFailureReasons(text string) string {
	var reasons []string
	seen := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "\t", 2)
		if len(parts) != 2 || parts[1] == "" || parts[1] == "None" {
			continue
		}
		r := parts[0] + ": " + parts[1]
		if !seen[r] {
			seen[r] = true
			reasons = append(reasons, r)
		}
	}
	return strings.Join(reasons, "; ")
}
