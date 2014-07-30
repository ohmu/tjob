/*
Package jenkins - Jenkins CI HTTP/SSH Client

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/
package jenkins

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/ohmu/tjob/sshcmd"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

var globalNetworkLimiter chan struct{}

func init() {
	// limit the number of concurrent network ops
	// TODO: better value for the limit
	globalNetworkLimiter = make(chan struct{}, 10)
}

type Jenkins struct {
	name               string
	URL                string
	InsecureSkipVerify bool
	SSH                *sshcmd.SSHNode
	*JobCache
}

func MakeJenkins(runnerID, url string, insecure bool, jobCache *JobCache) *Jenkins {
	return &Jenkins{
		name:               runnerID,
		URL:                url,
		InsecureSkipVerify: insecure,
		JobCache:           jobCache,
	}
}

func (j *Jenkins) jsonRequest(
	jobName string, sub string, params string) ([]byte, error) {
	globalNetworkLimiter <- struct{}{}
	defer func() {
		<-globalNetworkLimiter
	}()
	// TODO: better HTTP transport management, reuse of keep-alive connections
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: bool(j.InsecureSkipVerify),
		},
	}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr}
	url := fmt.Sprintf("%s/%s/%s/api/json%s",
		j.URL, jobName, sub, params)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (j *Jenkins) QueryJobs() ([]string, error) {
	// TODO: send to channel instead of returning slice
	jobsJSON, err := j.jsonRequest("view", "All", "?tree=jobs[name]")
	if err != nil {
		return nil, err
	}

	var jobs Jobs
	err = json.Unmarshal(jobsJSON, &jobs)
	if err != nil {
		return nil, err
	}
	jobsStr := make([]string, len(*jobs.Jobs))
	for i, job := range *jobs.Jobs {
		jobsStr[i] = job.Name
	}
	return jobsStr, nil
}

func (j *Jenkins) QueryJobBuilds(jobName string) ([]string, error) {
	// TODO: send to channel instead of returning slice
	buildsJSON, err := j.jsonRequest("job/"+jobName, "", "?tree=builds[number]")
	if err != nil {
		return nil, err
	}

	var builds JobBuilds
	err = json.Unmarshal(buildsJSON, &builds)
	if err != nil {
		return nil, err
	}
	buildStr := make([]string, len(*builds.Builds))
	for i, build := range *builds.Builds {
		buildStr[i] = strconv.Itoa(build.Number)
	}
	return buildStr, nil
}

func (j *Jenkins) QueryJobStatus(jobName string, jobNumber string, testDetails bool) (*JobStatus, error) {
	cached, err := j.JobCache.Retrieve(j.name, jobName, jobNumber)
	if err != nil || cached != nil {
		return cached, err
	}
	testDetails = true // always ask for details when using cache
	mainJSON, err := j.jsonRequest(
		"job/"+jobName, fmt.Sprintf("/%s", jobNumber),
		"?tree=building,duration,builtOn,result,timestamp,url,actions[causes[userId]]")
	if err != nil {
		return nil, err
	}

	var status JobStatus
	err = json.Unmarshal(mainJSON, &status)
	if err != nil {
		return nil, err
	}

	extra := ""
	if testDetails {
		/*
		   "age" : 0,
		   "className" : "foo.tests.TestClass",
		   "duration" : 1.2,
		   "errorDetails" : null,
		   "errorStackTrace" : null,
		   "failedSince" : 0,
		   "name" : "testSomething",
		   "skipped" : false,
		   "skippedMessage" : null,
		   "status" : "PASSED",
		   "stderr" : null,
		   "stdout" : null
		*/
		extra = ",suites[cases[status,name,className,duration,errorStackTrace,stderr,stdout]]"
	}
	testReportJSON, err := j.jsonRequest("job/"+jobName,
		fmt.Sprintf("/%s/testReport", jobNumber),
		"?tree=duration,failCount,passCount,skipCount"+extra)
	if err != nil {
		return nil, err
	}
	var testReport TestReport
	err = json.Unmarshal(testReportJSON, &testReport)
	if err != nil {
		// Jenkins may generate invalid JSON:
		// '{"duration":NaN,"failCount":5,"passCount":4422}'
		// attempt naive hack and replace NaN with null
		// NOTE: may screw up other values in the 'JSON' data
		testReportJSON = []byte(strings.Replace(string(testReportJSON),
			"\":NaN,", "\":null,", -1))
		err = json.Unmarshal(testReportJSON, &testReport)
	}
	if err == nil {
		status.TestReport = &testReport
	} else {
		// TODO: no sense to show warning here?
		//fmt.Println("WARNING: failed to parse testReport:", err)
	}

	// Remove uninteresting test results to lose weight
	status.TestReport.Prune()

	gitStatusJSON, err := j.jsonRequest("job/"+jobName,
		fmt.Sprintf("/%s/git", jobNumber),
		"?tree=lastBuiltRevision[branch[SHA1,name]],remoteUrls")
	if err != nil {
		// TODO: accept error, Git plugin may not be installed
		return nil, err
	}
	var gitStatus GitStatus
	err = json.Unmarshal(gitStatusJSON, &gitStatus)
	if err == nil {
		status.GitStatus = &gitStatus
	} else {
		// TODO: no sense to show warning here?
		fmt.Println("WARNING: failed to parse gitStatus:", err)
	}

	status.Prune() // trim off extra weight
	// cache the result of a finished build
	if err := j.JobCache.Store(j.name, jobName, jobNumber, &status); err != nil {
		return nil, err
	}

	return &status, nil // may return nil status.TestReport
}
