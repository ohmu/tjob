package main

/*
Package tjob - Job Status

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/

import (
	"fmt"
	"github.com/ohmu/tjob/config"
	"github.com/ohmu/tjob/jenkins"
)

type JobStatus struct {
	*config.Job
	Status *jenkins.JobStatus
	err    error
}

type jobStatusByBuildNumber []*JobStatus

func (a jobStatusByBuildNumber) Len() int      { return len(a) }
func (a jobStatusByBuildNumber) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a jobStatusByBuildNumber) Less(i, j int) bool {
	// TODO: better implementation
	switch {
	case a[i].Runner < a[j].Runner:
		return true
	case a[i].Runner > a[j].Runner:
		return false
	case a[i].JobName < a[j].JobName:
		return true
	case a[i].JobName > a[j].JobName:
		return false
	default:
		// oneliner approximation of descending numeric value order...
		return (fmt.Sprintf("%10s", a[i].BuildNumber) <
			fmt.Sprintf("%10s", a[j].BuildNumber))
	}
}
