package main

/*
Package tjob - Output Rendering

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/

import (
	"github.com/ohmu/tjob/pipeline"
	"os"
	"text/template"
)

type templateRenderer struct {
	pipeline.Node
	templateFile string
	display      *displayOptions
	Input        chan *JobStatus
}

func (node *templateRenderer) Run() error {
	tmpl, err := template.ParseFiles(node.templateFile)
	if err != nil {
		return node.AbortWithError(err)
	}
	var taskArray []*JobStatus
	for jobStatus := range node.Input {
		taskArray = append(taskArray, jobStatus)
	}
	env := struct {
		Tasks []*JobStatus
	}{taskArray}
	if err := tmpl.Execute(os.Stdout, &env); err != nil {
		return node.AbortWithError(err)
	}
	return nil
}
