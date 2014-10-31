package main

/*
Package tjob - Test job management utility

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/

import (
	"fmt"
	"github.com/ohmu/tjob/config"
	"github.com/ohmu/tjob/jenkins"
	"github.com/ohmu/tjob/pipeline"
	"time"
)

func init() {
	// TODO: long descs
	globalParser().AddCommand("run", "Start a job", "Start a job",
		&runJobCmd{})
	globalParser().AddCommand("restart", "Restart jobs", "Restart jobs",
		&restartJobCmd{})
	globalParser().AddCommand("list", "List jobs", "List jobs",
		&listJobsCmd{})
	globalParser().AddCommand("remove", "Remove jobs", "Remove jobs",
		&removeJobsCmd{})
}

func handleErrors(errors <-chan error) error {
	count := 0
	for err := range errors {
		fmt.Println("error:", err)
	}
	if count > 0 {
		return fmt.Errorf("there were %d errors", count)
	}
	return nil
}

type displayOptions struct {
	ShowTestDetails   bool          `short:"d" long:"error-detail" description:"Show detailed test results"`
	ShowTestTraceback bool          `short:"s" long:"stack-trace" description:"Show failed test stack traces"`
	ShowTestOutput    bool          `short:"p" long:"output" description:"Show failed test stdout/stderr"`
	ShowBuildURL      bool          `short:"l" long:"url" description:"Show build URL link"`
	ShowBuilder       bool          `long:"builder" description:"Show builder"`
	ShowTags          bool          `long:"tags" description:"Show tags"`
	ShowUser          bool          `short:"u" long:"username" description:"Show username"`
	ShowCommitID      bool          `short:"c" long:"commit-id" description:"Show version-control commit-id"`
	ShowBranch        bool          `short:"B" long:"branch" description:"Show version-control branch"`
	NoSorting         bool          `long:"no-sort" description:"CSV format, no output sorting, saves memory in large queries"`
	FailedTestSummary bool          `long:"failed-summary" description:"Show a summary of failed test counts"`
	SinceDuration     time.Duration `long:"since" description:"Show only builds started since X duration ago"` // TODO: better desc
}

type remoteJobQuery struct {
	pipeline.Node
	conf   *config.Config
	flags  *filterFlags
	Input  chan *config.Job
	Output chan *config.Job
}

func (node *remoteJobQuery) Run() error {
	defer close(node.Output)
	for runnerName, runner := range node.conf.Runners {
		jenk := jenkins.MakeJenkins(runnerName, runner.URL,
			runner.Insecure, nil) // JobCache not needed here
		jobs, err := jenk.QueryJobs()
		if err != nil {
			return node.AbortWithError(err)
		}
		for _, jobName := range jobs {
			// TODO: filterJob is only created for filtering, something better?
			filterJob := config.Job{
				Runner: runnerName, JobName: jobName,
				BuildNumber: "", Options: map[string]string{},
				Tags: []string{}}
			matched, err := filterByJobName(node.flags, &filterJob)
			if err != nil {
				return node.AbortWithError(err)
			} else if !matched {
				continue
			}

			// TODO: concurrent requests, channels
			builds, err := jenk.QueryJobBuilds(jobName)
			if err != nil {
				return node.AbortWithError(err)
			}
			for _, buildNumber := range builds {
				select {
				case <-node.AbortChannel():
					return nil
				case node.Output <- &config.Job{
					Runner: runnerName, JobName: jobName,
					BuildNumber: buildNumber, Options: map[string]string{},
					Tags: []string{}}:
				}
			}
		}
	}
	return nil
}

type configJobSender struct {
	pipeline.Node
	conf   *config.Config
	Input  chan *config.Job
	Output chan *config.Job
}

func (node *configJobSender) Run() error {
	defer close(node.Output)
	for _, job := range node.conf.Jobs {
		select {
		case <-node.AbortChannel():
			return nil
		case node.Output <- job:
		}
	}
	return nil
}

type jobCopier struct {
	pipeline.Node
	Options map[string]string
	Tags    []string
	Input   chan *JobStatus
	Output  chan *config.Job
}

func (node *jobCopier) Run() error {
	defer close(node.Output)
	for oldJob := range node.Input {
		select {
		case <-node.AbortChannel():
			return nil
		case node.Output <- oldJob.Copy(node.Options, node.Tags):
		}
	}
	return nil
}
