| JOB | BUILD | STATUS | PASS | SKIP | FAIL | URL |
|-----|-------|--------|------|------|------|-----|
{{range .Tasks}}| {{.JobName}} | {{.BuildNumber}} | {{.Status.Result}} | {{.Status.PassCount}} | {{.Status.SkipCount}} | {{.Status.FailCount}} | {{.Status.URL}} |
{{end}}

| JOB | BUILD | RESULT | CLASS | TEST CASE |
|-----|-------|--------|-------|-----------|
{{range $task := .Tasks}}{{if .Status.HasTestReport}}{{range .Status.TestReport.Suites}}{{range .Cases}}| {{$task.JobName}} | {{$task.BuildNumber}} | {{.Status}} | {{.ClassName}} | {{.Name}} |
{{end}}{{end}}{{end}}{{end}}
