package main

/*
Package tjob - Test job management utility

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/

import (
	"fmt"
	"github.com/ohmu/tjob/config"
	"github.com/ohmu/tjob/jenkins"
	"github.com/ohmu/tjob/sshcmd"
	"github.com/ohmu/tjob/tabout"
	"path"
	"strconv"
)

// TODO: go-flags support for forcing a lower-case struct to be processed (anonymous members)
// TODO: go-flags support for automatic lowercasing and dashifying (JobStatus -> job-status) as default long names
func init() {
	globalParser().AddCommand("runner", "Runner commands", "Runner commands", &struct { // TODO: long desc
		Add    runnerAddCmd    `command:"add" description:"Add a runner"`
		List   runnerListCmd   `command:"list" description:"List runners"`
		Update runnerUpdateCmd `command:"update" description:"Update a runner"`
		Remove runnerRemoveCmd `command:"rm" description:"Remove a runner"`
	}{})
}

type runnerPosArgs struct {
	RunnerID string `description:"Runner ID"`
}

// TODO: go-flags support for positional args without the extra struct
type runnerIDCmd struct {
	URL           string `long:"url" description:"Jenkins URL"`
	User          string `long:"user" description:"Jenkins/SSH username"`
	SSHPort       int    `long:"ssh-port" description:"Jenkins SSH port"`
	SSHKey        string `long:"ssh-key" description:"Jenkins SSH private key"`
	Insecure      string `long:"insecure" description:"Skip TLS server cert validation"`
	runnerPosArgs `positional-args:"yes" required:"yes"`
}

type runnerAddCmd runnerIDCmd

func (r *runnerAddCmd) Execute(args []string) error {
	conf, err := config.Load(globalFlags.ConfigFile)
	if err != nil {
		return err
	}
	if _, exists := conf.Runners[r.RunnerID]; exists {
		return fmt.Errorf(
			"runner '%s' already exists, use the 'update' command",
			r.RunnerID)
	}
	sshPortInt := 54410
	if r.SSHPort != 0 {
		sshPortInt = r.SSHPort
	}
	sshPort := sshcmd.SSHPort(sshPortInt)
	insecureStr := "false"
	if r.Insecure != "" {
		insecureStr = r.Insecure
	}
	insecure, err := strconv.ParseBool(insecureStr)
	if err != nil {
		return err
	}
	sshKey := "id_rsa"
	if r.SSHKey != "" {
		sshKey = r.SSHKey
	}
	conf.Runners[r.RunnerID] = &config.Runner{
		URL: r.URL, SSHPort: sshPort, SSHKey: sshKey, User: r.User,
		Insecure: insecure,
	}

	return conf.Save()
}

type runnerUpdateCmd runnerIDCmd

func (r *runnerUpdateCmd) Execute(args []string) error {
	conf, err := config.Load(globalFlags.ConfigFile)
	if err != nil {
		return err
	}
	if _, exists := conf.Runners[r.RunnerID]; !exists {
		return fmt.Errorf(
			"runner '%s' does not exist, use the 'add' command",
			r.RunnerID)
	}
	if value := r.URL; value != "" {
		conf.Runners[r.RunnerID].URL = value
	}
	if value := sshcmd.SSHPort(r.SSHPort); value != 0 {
		conf.Runners[r.RunnerID].SSHPort = value
	}
	if value := r.SSHKey; value != "" {
		conf.Runners[r.RunnerID].SSHKey = value
	}
	if value := r.User; value != "" {
		conf.Runners[r.RunnerID].User = value
	}
	if value := r.Insecure; value != "" {
		flag, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		conf.Runners[r.RunnerID].Insecure = flag
	}

	return conf.Save()
}

type runnerRemoveCmd runnerIDCmd

func (r *runnerRemoveCmd) Execute(args []string) error {
	conf, err := config.Load(globalFlags.ConfigFile)
	if err != nil {
		return err
	}
	if _, exists := conf.Runners[r.RunnerID]; !exists {
		return fmt.Errorf(
			"runner '%s' does not exist", r.RunnerID)
	}
	delete(conf.Runners, r.RunnerID)
	return conf.Save()
}

type runnerListCmd struct{}

func (r *runnerListCmd) Execute(args []string) error {
	conf, err := config.Load(globalFlags.ConfigFile)
	if err != nil {
		return err
	}
	output := tabout.New([]string{"NAME", "URL", "USER", "SSH-PORT",
		"SSH-KEY", "INSECURE"}, nil)
	for name, runner := range conf.Runners {
		output.Write(map[string]string{
			"NAME": name, "URL": runner.URL, "USER": runner.User,
			"SSH-PORT": runner.SSHPort.String(),
			"SSH-KEY":  runner.SSHKey,
			"INSECURE": strconv.FormatBool(runner.Insecure),
		})
	}
	output.Flush()
	return nil
}

func getJenkins(conf *config.Config, runnerID string) (*jenkins.Jenkins, error) {
	runner, exists := conf.Runners[runnerID]
	if !exists {
		return nil, fmt.Errorf(
			"runner '%s' does not exist", runnerID)
	}
	jobCache := jenkins.JobCache{path.Join(conf.Dir(), "cache")}
	jenk := jenkins.MakeJenkins(runnerID, runner.URL, runner.Insecure,
		&jobCache)
	return jenk, nil
}
