package main

/*
Package tjob - Remove Command

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/

import (
	"fmt"
	"github.com/ohmu/tjob/config"
	"github.com/ohmu/tjob/pipeline"
)

type removeJobsCmd struct {
	filterFlags
}

type jobRemover struct {
	pipeline.Node
	Input chan *JobStatus
	conf  *config.Config
}

func (node *jobRemover) Run() error {
	// TODO: inefficient, consider better storage structure
	for res := range node.Input {
		for i := len(node.conf.Jobs) - 1; i >= 0; i-- {
			job := node.conf.Jobs[i]
			if job.Runner == res.Runner &&
				job.JobName == res.JobName &&
				job.BuildNumber == res.BuildNumber {
				fmt.Printf("removed %s %s %s\n",
					job.Runner, job.JobName, job.BuildNumber)
				node.conf.Jobs = append(node.conf.Jobs[:i], node.conf.Jobs[i+1:]...)
			}
		}
	}
	return nil
}

func (r *removeJobsCmd) Execute(args []string) error {
	conf, err := config.Load(globalFlags.ConfigFile)
	if err != nil {
		return err
	}

	var jobs chan *config.Job
	jobber := configJobSender{Output: make(chan *config.Job, 10), conf: conf}
	jobs = jobber.Output
	preFiltered := jobFilterer{Input: jobs,
		Output: make(chan *config.Job, 10), flags: &r.filterFlags}
	collected := jobStatusQuery{Input: preFiltered.Output,
		Output: make(chan *JobStatus, 10), conf: conf,
		display: &displayOptions{}}

	// post-query filtering that requires build results in a slice
	postFiltered := resultFilterer{Input: collected.Output,
		Output: make(chan *JobStatus, 10), flags: &r.filterFlags,
		display: &displayOptions{}}

	removed := jobRemover{Input: postFiltered.Output, conf: conf}
	errors := pipeline.Wait(&jobber, &preFiltered, &collected, &postFiltered,
		&removed)
	if err := handleErrors(errors); err != nil {
		return err
	}

	return conf.Save()
}
