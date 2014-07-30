/*
Package tjob - Test job management utility

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/
package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
)

type topArgs struct {
	ConfigFile string `short:"c" long:"config" description:"Config file path"`
}

var gParser *flags.Parser
var globalFlags topArgs

func globalParser() *flags.Parser {
	if gParser == nil {
		globalFlags.ConfigFile = path.Join(
			os.Getenv("HOME"), ".tjob", "default.json")
		gParser = flags.NewParser(&globalFlags, flags.Default)
	}
	return gParser
}

func writeManPage() {
	if troffFile := os.Getenv("WRITE_MAN_PAGE"); troffFile != "" {
		out, err := os.Create(troffFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		globalParser().WriteManPage(out)
		os.Exit(0)
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	writeManPage()
	profileFile := os.Getenv("TJOB_CPUPROFILE")
	if profileFile != "" {
		f, perr := os.Create(profileFile)
		if perr != nil {
			log.Fatal(perr)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}
		defer func() {
			pprof.StopCPUProfile()
			fmt.Println("CPU profiling results written to",
				profileFile)
		}()
	}
	_, err := globalParser().ParseArgs(os.Args[1:])
	if err != nil {
		os.Exit(1)
	}
}
