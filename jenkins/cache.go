/*
Package jenkins - Job Result Cache

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/
package jenkins

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"runtime"
)

var globalCacheLimiter chan struct{}

func init() {
	// limit the number of concurrent cache io ops
	// TODO: better value for the limit
	globalCacheLimiter = make(chan struct{}, runtime.NumCPU()*2)
}

type JobCache struct {
	CacheDir string
}

func (c *JobCache) Retrieve(runner, job, build string) (*JobStatus, error) {
	if c == nil || c.CacheDir == "" {
		return nil, nil
	}
	fullPath := path.Join(c.CacheDir, runner, job, build+".json")

	globalCacheLimiter <- struct{}{}
	defer func() {
		<-globalCacheLimiter
	}()

	data, err := ioutil.ReadFile(fullPath)
	var status JobStatus
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, err
	default:
		err = json.Unmarshal(data, &status)
		if err != nil {
			return nil, err
		}
		return &status, nil
	}
}

func (c *JobCache) Store(runner, job, build string, status *JobStatus) error {
	if c == nil || c.CacheDir == "" || status.Building {
		return nil
	}
	globalCacheLimiter <- struct{}{}
	defer func() {
		<-globalCacheLimiter
	}()

	fullDir := path.Join(c.CacheDir, runner, job)
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(status, "", "\t")
	if err != nil {
		return err
	}
	fullPath := path.Join(fullDir, build+".json")
	tmpPath := fullPath + ".tmp"
	if err = ioutil.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	if err = os.Rename(tmpPath, fullPath); err != nil {
		return err
	}

	return nil
}
