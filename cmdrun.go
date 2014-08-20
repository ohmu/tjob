package main

/*
Package tjob - Run Command

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/

import (
	"fmt"
	"github.com/ohmu/tjob/config"
	"github.com/ohmu/tjob/pipeline"
	"github.com/ohmu/tjob/sshcmd"
	"net/url"
	"strings"
)

type runJobPosArgs struct {
	RunnerID string `description:"Runner ID"`
	JobName  string `description:"Name of the job to run"`
}

type runJobFlags struct {
	Option    map[string]string `short:"O" long:"set-option" description:"Set option for the build: 'key:value'"`
	Tags      []string          `short:"T" long:"set-tag" description:"Set tags for the build"`
	NumBuilds int               `short:"n" default:"1" description:"Number of builds to start"`
}

type runJobCmd struct {
	runJobFlags
	runJobPosArgs `positional-args:"yes" required:"yes"`
}

func startJob(runner *config.Runner, job *config.Job) (*config.Job, error) {
	url, err := url.Parse(runner.URL)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to start job %s: %s", job.JobName, err)
	}
	// url.Host may funnily be "host:port"
	host := strings.Split(url.Host, ":")[0]
	ssh := sshcmd.SSHNode{Host: host, Port: runner.SSHPort,
		User: runner.User, Key: runner.SSHKey}
	optStr := ""
	for key, value := range job.Options {
		optStr += fmt.Sprintf(" -p %s=%s", key, value)
	}
	cmd := fmt.Sprintf("build %s -w%s", job.JobName, optStr)
	// TODO: jenkins does not handle simultaneous parallel requests properly,
	// it may return the same build number for two different requests,
	// do something about it...
	resp, err := ssh.Execute(cmd)
	if err != nil {
		return nil, fmt.Errorf(
			"job %s start failed: %s: %s", job.JobName, err,
			resp)
	}
	// output: "Started testjobname #12345"
	var parsedJobName, buildNumber string
	n, err := fmt.Sscanf(resp, "Started %s #%s\n",
		&parsedJobName, &buildNumber)
	if err != nil || n != 2 {
		return nil, fmt.Errorf(
			"job %s start failed: %s: %s", job.JobName, err,
			resp)
	}
	return &config.Job{job.Runner, job.JobName, buildNumber,
		job.Options, job.Tags}, nil
}

func expandTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, multitag := range tags {
		for _, tag := range strings.Split(multitag, ",") {
			out = append(out, tag)
		}
	}
	return out
}

func (r *runJobCmd) Execute(args []string) error {
	conf, err := config.Load(globalFlags.ConfigFile)
	if err != nil {
		return err
	}
	start := make(chan *config.Job, r.NumBuilds)
	for nBuild := 0; nBuild < r.NumBuilds; nBuild++ {
		start <- &config.Job{r.RunnerID, r.JobName, "", r.Option, expandTags(r.Tags)}
	}
	close(start)
	started := jobStarter{Input: start, Output: make(chan *config.Job, 10),
		conf: conf}
	results := startResultPrinter{Input: started.Output,
		Output: make(chan *config.Job, 10), conf: conf}
	errors := pipeline.Wait(&started, &results)
	return handleErrors(errors)
}
