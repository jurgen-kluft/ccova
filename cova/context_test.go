package cova

import (
	"bytes"
	"errors"
	"testing"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestIssueString(t *testing.T) {
	tests := []struct {
		issue Issue
		want  string
	}{
		{Issue{IssueType: IssueTypeError, Description: "bad input"}, "Error: bad input"},
		{Issue{IssueType: IssueTypeWarning, Description: "unused"}, "Warning: unused"},
		{Issue{IssueType: IssueTypeInformation, Description: "size"}, "Information: size"},
		{Issue{IssueType: IssueType(100), Description: "other"}, "Unknown: other"},
	}
	for _, test := range tests {
		if got := test.issue.String(); got != test.want {
			t.Fatalf("Issue.String() = %q, want %q", got, test.want)
		}
	}
}

func TestContextAccumulatesIssues(t *testing.T) {
	ctx := NewContext()
	ctx.AddInformation("hidden")
	ctx.AddWarning("unused %s", "function")
	ctx.AddError("invalid value %d", 3)
	ctx.SetVerbose(true)
	ctx.AddInformation("text size %d", 12)

	issues := ctx.Issues()
	if len(issues) != 3 {
		t.Fatalf("len(Issues()) = %d, want 3", len(issues))
	}
	if !ctx.HasErrors() || !ctx.HasWarnings() {
		t.Fatalf("issue type detection failed: errors=%v warnings=%v", ctx.HasErrors(), ctx.HasWarnings())
	}
	issues[0].Description = "changed"
	if ctx.Issues()[0].Description == "changed" {
		t.Fatal("Issues() returned mutable context storage")
	}
}

func TestContextReportRoutesIssues(t *testing.T) {
	ctx := NewContext()
	ctx.SetVerbose(true)
	ctx.AddWarning("first")
	ctx.AddError("second")
	ctx.AddInformation("third")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if !ctx.Report(&stdout, &stderr) {
		t.Fatal("Report() failed")
	}
	if got, want := stdout.String(), "Warning: first\nInformation: third\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got, want := stderr.String(), "Error: second\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestContextReportFailure(t *testing.T) {
	ctx := NewContext()
	ctx.AddWarning("warning")
	if ctx.Report(nil, &bytes.Buffer{}) {
		t.Fatal("Report() succeeded with nil stdout")
	}
	if ctx.Report(&bytes.Buffer{}, nil) {
		t.Fatal("Report() succeeded with nil stderr")
	}
	if ctx.Report(failingWriter{}, &bytes.Buffer{}) {
		t.Fatal("Report() succeeded after a write failure")
	}
}
