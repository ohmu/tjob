package main

/*
Package tjob - List Command

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/

import (
	"github.com/ohmu/tjob/config"
	"github.com/ohmu/tjob/pipeline"
	"path"
)

type listJobsCmd struct {
	TemplateFile string `long:"template-file" description:"Use a Go template at given path to render output"`
	TemplateName string `long:"template" description:"Use a named Go template from the tjob configuration directory to render output"`
	displayOptions
	filterFlags
	RemoteMode bool `long:"remote" description:"Show jobs from remote server"`
}

func (r *listJobsCmd) Execute(args []string) error {
	conf, err := config.Load(globalFlags.ConfigFile)
	if err != nil {
		return err
	}

	var jobs chan *config.Job
	var jobsUp pipeline.Upstreamer
	if r.RemoteMode {
		jobber := remoteJobQuery{Output: make(chan *config.Job, 10),
			conf: conf, flags: &r.filterFlags}
		jobsUp = &jobber
		jobs = jobber.Output
	} else {
		jobber := configJobSender{Output: make(chan *config.Job, 10),
			conf: conf}
		jobsUp = &jobber
		jobs = jobber.Output
	}
	preFiltered := jobFilterer{Input: jobs,
		Output: make(chan *config.Job, 10), flags: &r.filterFlags}
	templateFile := r.TemplateFile
	if r.TemplateName != "" {
		templateFile = path.Join(path.Dir(globalFlags.ConfigFile),
			r.TemplateName)
	}
	collected := jobStatusQuery{Input: preFiltered.Output,
		Output: make(chan *JobStatus, 10), conf: conf,
		display: &r.displayOptions}

	var sorted chan *JobStatus
	var sortedUp pipeline.Upstreamer
	if r.displayOptions.NoSorting || r.displayOptions.FailedTestSummary {
		sorted = collected.Output
	} else {
		sorter := jobStatusSorter{Input: collected.Output,
			Output: make(chan *JobStatus, 10)}
		sortedUp = &sorter
		sorted = sorter.Output
	}

	// post-query filtering that requires build results in a slice
	postFiltered := resultFilterer{Input: sorted,
		Output: make(chan *JobStatus, 10), flags: &r.filterFlags,
		display: &r.displayOptions}

	var displayedUp pipeline.Upstreamer
	if templateFile != "" {
		displayed := templateRenderer{Input: postFiltered.Output,
			templateFile: templateFile, display: &r.displayOptions}
		displayedUp = &displayed
	} else if r.FailedTestSummary {
		displayed := failedTestSummaryRenderer{Input: postFiltered.Output,
			display: &r.displayOptions}
		displayedUp = &displayed
	} else {
		displayed := tabOutputRenderer{Input: postFiltered.Output,
			display: &r.displayOptions}
		displayedUp = &displayed
	}
	errors := pipeline.Wait(jobsUp, &preFiltered, &collected, sortedUp,
		&postFiltered, displayedUp)
	return handleErrors(errors)
}
