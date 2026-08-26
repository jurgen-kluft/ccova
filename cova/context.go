package cova

import (
	"fmt"
	"io"
)

type IssueType int8

const (
	IssueTypeError IssueType = iota
	IssueTypeWarning
	IssueTypeInformation
)

type Issue struct {
	IssueType   IssueType
	Description string
}

func (issue Issue) String() string {
	label := "Unknown"
	switch issue.IssueType {
	case IssueTypeError:
		label = "Error"
	case IssueTypeWarning:
		label = "Warning"
	case IssueTypeInformation:
		label = "Information"
	}
	return fmt.Sprintf("%s: %s", label, issue.Description)
}

type Context struct {
	issues  []Issue
	verbose bool
}

func NewContext() *Context {
	return &Context{}
}

func (ctx *Context) String() string {
	errors := ""
	for _, issue := range ctx.issues {
		errors += issue.String() + "\n"
	}
	return errors
}

func (ctx *Context) SetVerbose(verbose bool) {
	ctx.verbose = verbose
}

func (ctx *Context) Issues() []Issue {
	return append([]Issue(nil), ctx.issues...)
}

func (ctx *Context) HasErrors() bool {
	return ctx.hasIssueType(IssueTypeError)
}

func (ctx *Context) HasWarnings() bool {
	return ctx.hasIssueType(IssueTypeWarning)
}

func (ctx *Context) Report(stdout, stderr io.Writer) bool {
	if stdout == nil || stderr == nil {
		return false
	}
	for _, issue := range ctx.issues {
		output := stdout
		if issue.IssueType == IssueTypeError {
			output = stderr
		}
		if _, err := fmt.Fprintln(output, issue.String()); err != nil {
			return false
		}
	}
	return true
}

func (ctx *Context) AddError(format string, args ...any) {
	ctx.add(IssueTypeError, fmt.Sprintf(format, args...))
}

func (ctx *Context) AddWarning(format string, args ...any) {
	ctx.add(IssueTypeWarning, fmt.Sprintf(format, args...))
}

func (ctx *Context) AddInformation(format string, args ...any) {
	if ctx.verbose {
		ctx.add(IssueTypeInformation, fmt.Sprintf(format, args...))
	}
}

func (ctx *Context) hasIssueType(issueType IssueType) bool {
	for _, issue := range ctx.issues {
		if issue.IssueType == issueType {
			return true
		}
	}
	return false
}

func (ctx *Context) add(issueType IssueType, description string) {
	ctx.issues = append(ctx.issues, Issue{IssueType: issueType, Description: description})
}
