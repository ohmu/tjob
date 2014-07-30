/*
Package jenkins - JSON Data Structs

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/
package jenkins

import (
	"fmt"
	"time"
)

type Timestamp int64

func (t *Timestamp) String() string {
	if *t == 0 {
		return ""
	}
	ut := time.Unix(int64(*t)/1000, int64(*t)%1000)
	return ut.Format(time.Stamp)
}

type Duration int64 // Jenkins duration is total ms

func (d *Duration) String() string {
	return (time.Duration(*d) * time.Millisecond).String()
}

type Cause struct {
	UserID string
}

type Action struct {
	Causes []Cause
}

type JobStatus struct {
	Building bool
	Result   string
	Timestamp
	Duration
	BuiltOn string
	URL     string
	*TestReport
	Actions []Action
	XUserID string // moved from "causes"
	*GitStatus
}

func (j *JobStatus) HasTestReport() bool {
	return j.TestReport != nil
}

func (j *JobStatus) Prune() {
ALL:
	for _, action := range j.Actions {
		for _, cause := range action.Causes {
			if cause.UserID != "" {
				j.XUserID = cause.UserID
				break ALL
			}
		}
	}
	j.Actions = nil
}

type GitStatus struct {
	LastBuiltRevision struct {
		Branch []struct {
			SHA1 string
			Name string
		}
	}
	RemoteURLs []string
}

func (g *GitStatus) CommitID() string {
	if g == nil {
		return ""
	}
	for _, branch := range g.LastBuiltRevision.Branch {
		if branch.SHA1 != "" {
			return branch.SHA1
		}
	}
	return ""
}

func (g *GitStatus) Branch() string {
	if g == nil {
		return ""
	}
	for _, branch := range g.LastBuiltRevision.Branch {
		if branch.Name != "" {
			return branch.Name
		}
	}
	return ""
}

type Count int64

func (c *Count) String() string {
	return fmt.Sprintf("%5d", *c)
}

type Case struct {
	Status          string
	Name            string
	ClassName       string
	Duration        float64
	ErrorStackTrace string
	Stderr          string
	Stdout          string
}

type Suite struct {
	Cases []*Case
}

type TestReport struct {
	FailCount Count
	PassCount Count
	SkipCount Count
	Suites    []*Suite
}

func (s *TestReport) Prune() {
	if s == nil {
		return
	}
	for si := len(s.Suites) - 1; si >= 0; si-- {
		su := s.Suites[si]
		for i := len(su.Cases) - 1; i >= 0; i-- {
			if su.Cases[i].Status == "PASSED" || su.Cases[i].Status == "FIXED" || su.Cases[i].Status == "SKIPPED" {
				su.Cases = append(su.Cases[:i], su.Cases[i+1:]...)
			}
		}
		if len(su.Cases) == 0 {
			s.Suites = append(s.Suites[:si], s.Suites[si+1:]...)
		}
	}
	if len(s.Suites) == 0 {
		s.Suites = []*Suite{}
	}
}

type Build struct {
	Number int
}

type JobBuilds struct {
	Builds *[]Build
}

type Job struct {
	Name string
}

type Jobs struct {
	Jobs *[]Job
}

func (j *JobStatus) String() string {
	return fmt.Sprintf(
		"building=%t result=%s duration=%d P/S/F=%d/%d/%d",
		j.Building, j.Result, j.Duration, j.TestReport.PassCount,
		j.TestReport.SkipCount, j.TestReport.FailCount)
}

func (j *JobStatus) IsFailed() bool {
	return j.Result != "SUCCESS" && !j.Building
}
