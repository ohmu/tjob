package main

/*
Package tjob - Restart Command

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/

import (
	"github.com/ohmu/tjob/config"
	"github.com/ohmu/tjob/pipeline"
)

type restartJobCmd struct {
	runJobFlags
	filterFlags
	UserConfirmed bool `long:"--confirm" description:"Confirm restarting many jobs"`
}

func (r *restartJobCmd) Execute(args []string) error {
	conf, err := config.Load(globalFlags.ConfigFile)
	if err != nil {
		return err
	}
	jobs := configJobSender{Output: make(chan *config.Job, 10), conf: conf}
	preFiltered := jobFilterer{Input: jobs.Output,
		Output: make(chan *config.Job, 10), flags: &r.filterFlags}
	// TODO: channel-capable --confirm limit enforcement
	/*
		confirmLimit := 10
		if count > confirmLimit && !r.UserConfirmed {
			return fmt.Errorf("restarting more than %d jobs (%d jobs selected) requires --confirm",
				confirmLimit, count)
		}
	*/
	collected := jobStatusQuery{Input: preFiltered.Output,
		Output: make(chan *JobStatus, 10), conf: conf,
		display: &displayOptions{NoSorting: true}}
	sorter := jobStatusSorter{Input: collected.Output,
		Output: make(chan *JobStatus, 10)}
	postFiltered := resultFilterer{Input: sorter.Output,
		Output: make(chan *JobStatus, 10), flags: &r.filterFlags,
		display: &displayOptions{NoSorting: true}}
	jobCopies := jobCopier{Input: postFiltered.Output,
		Output: make(chan *config.Job, 10), Options: r.Option,
		Tags: r.Tags}
	started := jobStarter{Input: jobCopies.Output,
		Output: make(chan *config.Job, 10), conf: conf}
	results := startResultPrinter{Input: started.Output,
		Output: make(chan *config.Job, 10), conf: conf}
	errors := pipeline.Wait(&jobs, &preFiltered, &collected, &sorter,
		&postFiltered, &jobCopies, &started, &results)
	return handleErrors(errors)
}
