package main

/*
Package tjob - Failed Test Summary Rendering

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/

import (
	"github.com/ohmu/tjob/pipeline"
	"github.com/ohmu/tjob/tabout"
	"sort"
	"strconv"
)

type summaryKey struct {
	class string
	test  string
	count int // not used when used as a key
}
type summarySlice []summaryKey

func (a summarySlice) Len() int           { return len(a) }
func (a summarySlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a summarySlice) Less(i, j int) bool { return a[i].count > a[j].count }

type failedTestSummaryRenderer struct {
	pipeline.Node
	display *displayOptions
	Input   chan *JobStatus
}

func (node *failedTestSummaryRenderer) Run() error {
	// print test results
	// TODO: make func out of skipStatuses
	skipStatuses := map[string]bool{
		"PASSED": true, "FIXED": true, "SKIPPED": true}
	sum := make(map[summaryKey]int)
	for res := range node.Input {
		status := res.Status
		if status == nil || status.TestReport == nil {
			continue
		}
		for _, suite := range status.TestReport.Suites {
			for _, testCase := range suite.Cases {
				if skipStatuses[testCase.Status] {
					continue
				}
				key := summaryKey{testCase.ClassName,
					testCase.Name, 0}
				sum[key]++
			}
		}
	}

	arr := make(summarySlice, len(sum))
	i := 0
	for key, hits := range sum {
		arr[i] = key
		arr[i].count = hits
		i++
	}
	sort.Sort(arr)

	output := tabout.New([]string{"COUNT", "CLASS", "TEST"}, nil)
	for _, key := range arr {
		if err := output.Write(map[string]string{
			"COUNT": strconv.Itoa(key.count),
			"CLASS": key.class,
			"TEST":  key.test,
		}); err != nil {
			return node.AbortWithError(err)
		}
	}
	output.Flush()
	return nil
}
