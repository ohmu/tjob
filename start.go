package main

/*
Package tjob - Test job management utility

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/

import (
	"fmt"
	"github.com/ohmu/tjob/config"
	"github.com/ohmu/tjob/pipeline"
)

type startResultPrinter struct {
	pipeline.Node
	conf   *config.Config
	Input  chan *config.Job
	Output chan *config.Job
}

func (node *startResultPrinter) Run() error {
	defer close(node.Output)
	for res := range node.Input {
		fmt.Printf("started job: %s %s %s\n", res.Runner, res.JobName,
			res.BuildNumber)
		node.conf.Jobs = append(node.conf.Jobs, res)

		// save after each successful launch, next one may fail
		if err := node.conf.Save(); err != nil {
			// exit immediately, don't wait for the goroutines
			return node.AbortWithError(
				fmt.Errorf("failed to save state: %s",
					err.Error()))
		}
	}
	return nil
}

type jobStarter struct {
	pipeline.Node
	conf   *config.Config
	Input  chan *config.Job
	Output chan *config.Job
}

func (node *jobStarter) Run() error {
	defer close(node.Output)
	limiter := make(chan bool, 1) // launch one at a time
	for job := range node.Input {
		runner, exists := node.conf.Runners[job.Runner]
		if !exists {
			return node.AbortWithError(fmt.Errorf("runner '%s' does not exists, use the 'runner create' command\n", job.Runner))
		}
		limiter <- true
		job, err := startJob(runner, job)
		if err != nil {
			return node.AbortWithError(err)
		}
		select {
		case <-node.AbortChannel():
			return nil
		case node.Output <- job:
		}
		<-limiter
	}
	return nil
}
