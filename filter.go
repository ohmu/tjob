package main

/*
Package tjob - Job Filtering

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/

import (
	"github.com/ohmu/tjob/config"
	"github.com/ohmu/tjob/pipeline"
	"path/filepath"
	"strings"
)

type filterFlags struct {
	FilterTags    []string          `short:"t" long:"tag" description:"Select only builds with tag"`
	FilterOptions map[string]string `short:"o" long:"option" description:"Select only builds with option key:value"`
	FilterRunner  []string          `short:"r" long:"runner" description:"Select only builds for the given runner"`
	FilterJob     []string          `short:"j" long:"job" description:"Select only builds for the given job"`
	FilterBuild   []string          `short:"b" long:"build" description:"Select only builds with buildnumber"`
	OnlyAllFailed bool              `long:"all-failed" description:"Select only failed builds"`
	OnlyFailing   bool              `long:"failing" description:"Select only currently failing builds"`
}

type filterFunc func(r *filterFlags, job *config.Job) (bool, error)

// filterByTags: "-t A -t B" means "A or B", "-t A,B" means "A and B"
func filterByTags(r *filterFlags, job *config.Job) (bool, error) {
	if len(r.FilterTags) == 0 {
		return true, nil
	}
	for _, wantTag := range r.FilterTags {
		if wantTag == "-" && len(job.Tags) == 0 {
			return true, nil
		}
		wantTags := strings.Split(wantTag, ",")
		for _, tag := range wantTags {
			found := false
			for _, haveTag := range job.Tags {
				matched, err := filepath.Match(tag, haveTag)
				if err != nil {
					return false, err
				}
				if matched {
					found = true
					break
				}
			}
			if !found {
				return false, nil
			}
		}
	}
	return true, nil
}

func filterByOptions(r *filterFlags, job *config.Job) (bool, error) {
	if len(r.FilterOptions) == 0 {
		return true, nil
	}
	for wantKey, wantValue := range r.FilterOptions {
		// TODO: should matching be any or all?
		matched, err := filepath.Match(wantValue, job.Options[wantKey])
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func listFilter(req []string, name string) (bool, error) {
	if len(req) == 0 {
		return true, nil
	}
	for _, value := range req {
		matched, err := filepath.Match(value, name)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func filterByJobName(r *filterFlags, job *config.Job) (bool, error) {
	return listFilter(r.FilterJob, job.JobName)
}

func filterByBuildNumber(r *filterFlags, job *config.Job) (bool, error) {
	return listFilter(r.FilterBuild, job.BuildNumber)
}

func filterByRunnerName(r *filterFlags, job *config.Job) (bool, error) {
	return listFilter(r.FilterRunner, job.Runner)
}

func multiFilter(r *filterFlags, job *config.Job, funcs ...filterFunc) (bool, error) {
	for _, f := range funcs {
		matched, err := f(r, job)
		if err != nil || !matched {
			return false, err
		}
	}
	return true, nil
}

type jobFilterer struct {
	pipeline.Node
	flags  *filterFlags
	Input  chan *config.Job
	Output chan *config.Job
}

func (node *jobFilterer) Run() error {
	defer close(node.Output)
	for job := range node.Input {
		matched, err := multiFilter(node.flags, job, filterByTags,
			filterByOptions, filterByJobName, filterByBuildNumber,
			filterByRunnerName)
		if err != nil {
			return node.AbortWithError(err)
		}
		if matched {
			select {
			case <-node.AbortChannel():
				return nil
			case node.Output <- job:
			}
		}
	}
	return nil
}
