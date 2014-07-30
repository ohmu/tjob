| JOB | BUILD | STATUS | URL |
|-----|-------|--------|-----|
{{range .Tasks}}| {{.JobName}} | {{.BuildNumber}} | {{.Status.Result}} | {{.Status.URL}} |
{{end}}

| JOB | BUILD | RESULT | CLASS | TEST CASE |
|-----|-------|--------|-------|-----------|
{{range $task := .Tasks}}{{if .Status.HasTestReport}}{{range .Status.TestReport.Suites}}{{range .Cases}}| {{$task.JobName}} | {{$task.BuildNumber}} | {{.Status}} | {{.ClassName}} | {{.Name}} |
{{end}}{{end}}{{end}}{{end}}
