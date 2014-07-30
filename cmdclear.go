package main

/*
Package tjob - Test job management utility

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/

import (
	"github.com/ohmu/tjob/config"
)

type clearJobsCmd struct {
	// filterFlags // TODO: implementation
	// TODO: --all flag, default: nothing -> error
}

func (r *clearJobsCmd) Execute(args []string) error {
	conf, err := config.Load(globalFlags.ConfigFile)
	if err != nil {
		return err
	}

	for i := len(conf.Jobs) - 1; i >= 0; i-- {
		// TODO: proper impl
		conf.Jobs = append(conf.Jobs[:i], conf.Jobs[i+1:]...)
	}
	return conf.Save()
}
