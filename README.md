sensu-check-log
===============

CircleCI: [![CircleCI Build Status](https://circleci.com/gh/sensu/sensu-check-log/tree/master.svg?style=svg)](https://circleci.com/gh/sensu/sensu-check-log/tree/master)

High performance log file analyzer

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
