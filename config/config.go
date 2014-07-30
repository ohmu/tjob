package config

/*
Package config - tjob configuration types

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/

import (
	"encoding/json"
	"github.com/ohmu/tjob/sshcmd"
	"io/ioutil"
	"os"
	"path"
)

type Runner struct {
	URL string
	sshcmd.SSHPort
	SSHKey   string
	User     string
	Insecure bool
}

type Project struct {
}

type Job struct {
	Runner      string
	JobName     string
	BuildNumber string
	Options     map[string]string
	Tags        []string
}

func (j *Job) Copy(options map[string]string, tags []string) *Job {
	newOpts := make(map[string]string)
	for k, v := range j.Options {
		newOpts[k] = v // old options are spared
	}
	for k, v := range options {
		newOpts[k] = v // new options override old ones
	}
	var tagSource []string
	if len(tags) == 0 {
		// when no new tags are given, transfer all the old ones
		// e.g. rebuild a failed job in the same context
		tagSource = j.Tags
	} else {
		// old tags are not transferred to new job, keep none of
		// the old context, don't mix old and new tags
		tagSource = tags
	}

	newTags := make([]string, len(tagSource))
	for i, v := range tagSource {
		newTags[i] = v // only new tags are set for the new job
	}
	return &Job{j.Runner, j.JobName, j.BuildNumber, newOpts, newTags}
}

type Config struct {
	path     string
	Runners  map[string]*Runner
	Projects map[string]*Project
	Jobs     []*Job
}

func (c *Config) Dir() string {
	return path.Dir(c.path)
}

func Load(filePath string) (*Config, error) {
	data, err := ioutil.ReadFile(filePath)
	var config Config
	switch {
	case os.IsNotExist(err):
		// pass
	case err != nil:
		return nil, err
	default:
		err = json.Unmarshal(data, &config)
		if err != nil {
			return nil, err
		}
	}
	config.path = filePath

	if config.Runners == nil {
		config.Runners = make(map[string]*Runner)
	}
	if config.Projects == nil {
		config.Projects = make(map[string]*Project)
	}
	if config.Jobs == nil {
		config.Jobs = make([]*Job, 0)
	}
	return &config, nil
}

func (c *Config) Save() error {
	dir := path.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}
	tmpPath := c.path + ".tmp"
	if err = ioutil.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	if err = os.Rename(tmpPath, c.path); err != nil {
		return err
	}
	return nil
}
