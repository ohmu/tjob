package main

/*
Package tjob - Output Rendering

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/

import (
	"code.google.com/p/go.crypto/ssh/terminal"
	"fmt"
	"github.com/ohmu/tjob/config"
	"github.com/ohmu/tjob/jenkins"
	"github.com/ohmu/tjob/pipeline"
	"os"
	"sort"
	"sync"
	"time"
)

var globalProgramStart time.Time

func init() {
	globalProgramStart = time.Now()
}

func filterByStartedSince(display *displayOptions, jobStatus *jenkins.JobStatus) bool {
	if display.SinceDuration == 0 {
		return true
	} else if jobStatus == nil || jobStatus.Timestamp == 0 {
		return false
	}
	minDate := globalProgramStart.Add(-display.SinceDuration)
	return time.Unix(int64(jobStatus.Timestamp)/1000, 0).After(minDate)
}

type resultFilterer struct {
	pipeline.Node
	flags   *filterFlags
	display *displayOptions
	Input   chan *JobStatus
	Output  chan *JobStatus
}

// filterResults input is sorted in ascending BuildNumber order
func (node *resultFilterer) Run() error {
	defer close(node.Output)
	var prev *JobStatus
	for cur := range node.Input {
		var sendVal *JobStatus
		switch {
		case node.display.SinceDuration != 0 &&
			!filterByStartedSince(node.display, cur.Status):
		case node.flags.OnlyAllFailed && (cur.Status == nil ||
			cur.Status.IsFailed()):
			sendVal = cur
		case node.flags.OnlyAllFailed:
		case (node.flags.OnlyFailing && prev == nil):
		case (node.flags.OnlyFailing && prev.Status != nil &&
			!prev.Status.IsFailed()):
		case (node.flags.OnlyFailing && (cur.Runner != prev.Runner ||
			cur.JobName != prev.JobName)):
			sendVal = prev
		case node.flags.OnlyFailing:
		default:
			sendVal = cur

		}
		if sendVal != nil {
			select {
			case <-node.AbortChannel():
				return nil
			case node.Output <- sendVal:
			}
		}
		prev = cur
	}
	if node.flags.OnlyFailing && prev != nil && prev.Status.IsFailed() {
		select {
		case <-node.AbortChannel():
			return nil
		case node.Output <- prev:
		}
	}
	return nil
}

type jobStatusQuery struct {
	pipeline.Node
	conf    *config.Config
	display *displayOptions
	Input   chan *config.Job
	Output  chan *JobStatus
}

func (node *jobStatusQuery) Run() error {
	defer close(node.Output)
	limiter := make(chan struct{}, 10) // max concurrent ops
	wgJenk := sync.WaitGroup{}
	i := 0
	reportProgress := (terminal.IsTerminal(int(os.Stdout.Fd())) &&
		!node.display.NoSorting)
	for job := range node.Input {
		jenk, err := getJenkins(node.conf, job.Runner)
		if err != nil {
			return node.AbortWithError(err)
		}

		// report query progress
		if reportProgress {
			locStr := fmt.Sprintf("%s %s %s", job.Runner,
				job.JobName, job.BuildNumber)
			if len(locStr) > 40 {
				locStr = locStr[:40]
			}
			fmt.Printf("%6d %-60s\r", i, locStr)
		}

		wgJenk.Add(1)
		limiter <- struct{}{} // don't want too many goroutines
		go func(job *config.Job) {
			defer wgJenk.Done()
			status, err := jenk.QueryJobStatus(job.JobName,
				job.BuildNumber, node.display.ShowTestDetails)
			select {
			case <-node.AbortChannel():
				return
			case node.Output <- &JobStatus{job, status, err}:
			}
			<-limiter
		}(job)
		i++
	}
	if reportProgress {
		fmt.Printf("%6s %-60s\r", "", "") // clean up output
	}
	// wait for tasks to complete before closing output channel
	wgJenk.Wait()
	return nil
}

type jobStatusSorter struct {
	pipeline.Node
	Input  chan *JobStatus
	Output chan *JobStatus
}

func (node *jobStatusSorter) Run() error {
	defer close(node.Output)
	var results []*JobStatus
	i := 0
	for job := range node.Input {
		results = append(results, job)
		i++
	}
	sort.Sort(jobStatusByBuildNumber(results))

	for _, res := range results {
		select {
		case <-node.AbortChannel():
			return nil
		case node.Output <- res:
		}
	}
	return nil
}
