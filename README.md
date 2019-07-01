sensu-check-log
===============

CircleCI: [![CircleCI Build Status](https://circleci.com/gh/sensu/sensu-check-log/tree/master.svg?style=svg)](https://circleci.com/gh/sensu/sensu-check-log/tree/master)

`sensu-check-log` is a log file analyzer plugin for Sensu Go. The program scans
a log file, checks it for matches, and sends a special failure event to the
agent events API when a match is detected.

The check itself will always return a 0 status, unless execution fails for
some reason.

The check must be configured with `stdin: true` so that failure events can
be formed correctly. If the check is not configured with `stdin: true`, then
it will fail to execute.

```
Usage of sensu-check-log:
  -api-url string
    	agent events API URL (default "http://localhost:3031/events")
  -event-status int
    	event status on positive match (default 1)
  -log string
    	path to log file (required)
  -match string
    	RE2 regexp matcher expression (required)
  -max-bytes int
    	max number of bytes to read (0 means unlimited)
  -procs int
    	number of parallel analyzer processes (default 4)
  -state string
    	state file for incremental log analysis (required)
```
