package aws

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeAWS answers the aws CLI from a script: each call gets the next answer
// for its subcommand.
type fakeAWS struct {
	calls   [][]string
	answers map[string][]fakeAnswer
}

type fakeAnswer struct {
	out string
	err error
}

func (f *fakeAWS) run(args ...string) (string, error) {
	f.calls = append(f.calls, args)
	sub := ""
	for i, a := range args {
		if a == "cloudformation" && i+1 < len(args) {
			sub = args[i+1]
		}
	}
	q := f.answers[sub]
	if len(q) == 0 {
		return "", errors.New("unexpected " + sub)
	}
	a := q[0]
	if len(q) > 1 {
		f.answers[sub] = q[1:]
	}
	return a.out, a.err
}

func useFakeAWS(t *testing.T, f *fakeAWS) {
	prev := runAWS
	runAWS = f.run
	t.Cleanup(func() { runAWS = prev })
}

func testStacks() (*StackManager, *[]string) {
	var seen []string
	s := NewStackManager("lab")
	s.Poll = time.Millisecond
	s.Timeout = time.Second
	s.Progress = func(status string) { seen = append(seen, status) }
	return s, &seen
}

func TestStackCreate_Arguments(t *testing.T) {
	f := &fakeAWS{answers: map[string][]fakeAnswer{"create-stack": {{out: `{"StackId":"x"}`}}}}
	useFakeAWS(t, f)
	s, _ := testStacks()
	if err := s.Create("Pennsieve-Compute-dev-1a2b3c4d", "https://bucket/cfn/pennsieve-compute-role/v1.yaml",
		"Pennsieve-Compute-dev-1a2b3c4d", "pn-0123456789abcdef0123456789abcdef", "123456789012"); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(f.calls[0], " ")
	want := "--profile lab --region us-east-1 cloudformation create-stack --stack-name Pennsieve-Compute-dev-1a2b3c4d" +
		" --template-url https://bucket/cfn/pennsieve-compute-role/v1.yaml --parameters" +
		" ParameterKey=RoleName,ParameterValue=Pennsieve-Compute-dev-1a2b3c4d" +
		" ParameterKey=ExternalId,ParameterValue=pn-0123456789abcdef0123456789abcdef" +
		" ParameterKey=PennsieveAccountId,ParameterValue=123456789012 --capabilities CAPABILITY_NAMED_IAM"
	if got != want {
		t.Fatalf("aws %s\nwant aws %s", got, want)
	}
}

func TestStackWait_Completes(t *testing.T) {
	f := &fakeAWS{answers: map[string][]fakeAnswer{"describe-stacks": {
		{out: "CREATE_IN_PROGRESS"}, {out: "CREATE_IN_PROGRESS"}, {out: "CREATE_COMPLETE"},
	}}}
	useFakeAWS(t, f)
	s, seen := testStacks()
	if err := s.Wait("r", "CREATE_COMPLETE"); err != nil {
		t.Fatal(err)
	}
	if strings.Join(*seen, ",") != "CREATE_IN_PROGRESS,CREATE_COMPLETE" {
		t.Fatalf("progress %v", *seen)
	}
}

// A stack that rolls back comes back with CloudFormation's reasons.
func TestStackWait_FailsWithReasons(t *testing.T) {
	f := &fakeAWS{answers: map[string][]fakeAnswer{
		"describe-stacks": {{out: "CREATE_IN_PROGRESS"}, {out: "ROLLBACK_IN_PROGRESS"}},
		"describe-stack-events": {{out: "PennsieveComputeRole\tUser: arn:aws:iam::1:user/x is not authorized to perform: iam:CreateRole\n" +
			"Pennsieve-Compute-dev-1a2b3c4d\tThe following resource(s) failed to create: [PennsieveComputeRole]."}},
	}}
	useFakeAWS(t, f)
	s, _ := testStacks()
	err := s.Wait("Pennsieve-Compute-dev-1a2b3c4d", "CREATE_COMPLETE")
	if err == nil || !strings.Contains(err.Error(), "ROLLBACK_IN_PROGRESS") || !strings.Contains(err.Error(), "not authorized to perform: iam:CreateRole") {
		t.Fatalf("got %v", err)
	}
}

func TestStackUpdate_NothingToDo(t *testing.T) {
	f := &fakeAWS{answers: map[string][]fakeAnswer{"update-stack": {{err: errors.New("An error occurred (ValidationError) when calling the UpdateStack operation: No updates are to be performed.")}}}}
	useFakeAWS(t, f)
	s, _ := testStacks()
	updated, err := s.Update("r", "https://bucket/latest.yaml")
	if err != nil || updated {
		t.Fatalf("updated %v, err %v", updated, err)
	}
	if !strings.Contains(strings.Join(f.calls[0], " "), "ParameterKey=ExternalId,UsePreviousValue=true") {
		t.Fatalf("update must keep the stack's parameters: %v", f.calls[0])
	}
}

func TestStackState(t *testing.T) {
	cases := []struct {
		status, want string
		state        stackProgress
	}{
		{"CREATE_COMPLETE", "CREATE_COMPLETE", stackDone},
		{"CREATE_IN_PROGRESS", "CREATE_COMPLETE", stackInProgress},
		{"ROLLBACK_IN_PROGRESS", "CREATE_COMPLETE", stackFailed},
		{"ROLLBACK_COMPLETE", "CREATE_COMPLETE", stackFailed},
		{"CREATE_FAILED", "CREATE_COMPLETE", stackFailed},
		{"UPDATE_COMPLETE_CLEANUP_IN_PROGRESS", "UPDATE_COMPLETE", stackInProgress},
		{"UPDATE_ROLLBACK_IN_PROGRESS", "UPDATE_COMPLETE", stackFailed},
		{"DELETE_IN_PROGRESS", "CREATE_COMPLETE", stackFailed},
		{"UPDATE_COMPLETE", "CREATE_COMPLETE", stackFailed},
	}
	for _, c := range cases {
		if got := stackState(c.status, c.want); got != c.state {
			t.Errorf("stackState(%s, %s) = %v, want %v", c.status, c.want, got, c.state)
		}
	}
}

func TestFormatFailureReasons(t *testing.T) {
	got := formatFailureReasons("A\tno permission\nA\tno permission\nB\tNone\n\n")
	if got != "A: no permission" {
		t.Fatalf("got %q", got)
	}
}
