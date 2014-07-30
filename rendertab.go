package main

/*
Package tjob - Tab Output Rendering

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/

import (
	"fmt"
	"github.com/ohmu/tjob/csvout"
	"github.com/ohmu/tjob/jenkins"
	"github.com/ohmu/tjob/pipeline"
	"github.com/ohmu/tjob/tabout"
	"strings"
	"time"
)

type OutputWriter interface {
	Write(map[string]string) error
	Flush() error
}

type tabOutputRenderer struct {
	pipeline.Node
	display *displayOptions
	Input   chan *JobStatus
}

func (node *tabOutputRenderer) Run() error {
	// TODO: non-buffering implementation
	var output OutputWriter
	fields := []string{"RUNNER", "JOB", "BUILD", "BUILDER", "USER",
		"BRANCH", "COMMIT-ID", "TAGS", "STATUS", "TIMESTAMP", "DURATION",
		"PASS", "SKIP", "FAIL", "URL", "ERROR"}
	if node.display.NoSorting {
		output = csvout.New(fields)
	} else {
		output = tabout.New(fields, map[string]bool{
			"BRANCH":    node.display.ShowBranch,
			"BUILDER":   node.display.ShowBuilder,
			"COMMIT-ID": node.display.ShowCommitID,
			"TAGS":      node.display.ShowTags,
			"URL":       node.display.ShowBuildURL,
			"USER":      node.display.ShowUser,
		})
	}
	var results []*JobStatus
	for res := range node.Input {
		if node.display.ShowTestDetails || node.display.ShowTestOutput ||
			node.display.ShowTestTraceback {
			// only collect to a slice when it is required
			results = append(results, res)
		}
		// TODO: select()
		status := res.Status
		var errStr string
		if res.err != nil {
			status = &jenkins.JobStatus{
				TestReport: &jenkins.TestReport{}}
			errStr = res.err.Error()
		}
		state := status.Result
		if status.Building {
			state = "RUNNING"
		}

		var pass, skip, fail string
		if status.TestReport != nil {
			pass = status.PassCount.String()
			skip = status.SkipCount.String()
			fail = status.FailCount.String()
		}
		dur := status.Duration.String()
		if status.Duration == 0 {
			startTime := time.Unix(int64(
				status.Timestamp)/1000, 0).Round(time.Second)
			elapsed := time.Now().Round(time.Second).Sub(startTime)

			dur = elapsed.String() + "+"
		}
		if err := output.Write(map[string]string{
			"RUNNER": res.Runner,
			"JOB":    res.JobName, "BUILD": res.BuildNumber,
			"BUILDER":   status.BuiltOn,
			"USER":      status.XUserID,
			"BRANCH":    status.GitStatus.Branch(),
			"COMMIT-ID": status.GitStatus.CommitID(),
			"TAGS":      strings.Join(res.Tags, ","),
			"STATUS":    state,
			"TIMESTAMP": status.Timestamp.String(),
			"DURATION":  dur, "PASS": pass, "SKIP": skip,
			"FAIL": fail, "URL": status.URL, "ERROR": errStr,
		}); err != nil {
			return node.AbortWithError(err)
		}
	}
	output.Flush()

	// print test results
	skipStatuses := map[string]bool{
		"PASSED": true, "FIXED": true, "SKIPPED": true}
	output = tabout.New([]string{"RUNNER", "JOB", "BUILD", "RESULT",
		"ELAPSED", "CLASS", "TEST"}, nil)
	first := true
	for _, res := range results {
		status := res.Status
		if !node.display.ShowTestDetails || status == nil ||
			status.TestReport == nil {
			continue
		}
		if first {
			fmt.Println()
			first = false
		}
		for _, suite := range status.TestReport.Suites {
			for _, testCase := range suite.Cases {
				if skipStatuses[testCase.Status] {
					continue
				}
				if err := output.Write(map[string]string{
					"RUNNER":  res.Runner,
					"JOB":     res.JobName,
					"BUILD":   res.BuildNumber,
					"RESULT":  testCase.Status,
					"ELAPSED": (time.Duration(testCase.Duration) * time.Second).String(),
					"CLASS":   testCase.ClassName,
					"TEST":    testCase.Name,
				}); err != nil {
					return node.AbortWithError(err)
				}
				if node.display.ShowTestTraceback {
					fmt.Print("STACK TRACE:\n" + testCase.ErrorStackTrace + "\n")
				}
				if node.display.ShowTestOutput && testCase.Stdout != "" {
					fmt.Print("STDOUT:\n" + testCase.Stdout + "\n")
				}
				if node.display.ShowTestOutput && testCase.Stderr != "" {
					fmt.Print("STDERR:\n" + testCase.Stderr + "\n")
				}
			}
		}
	}
	output.Flush()
	return nil
}
