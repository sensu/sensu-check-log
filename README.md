[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/sensu/sensu-check-log)
![Go Test](https://github.com/sensu/sensu-check-log/workflows/Go%20Test/badge.svg)
![goreleaser](https://github.com/sensu/sensu-check-log/workflows/goreleaser/badge.svg)

# sensu-check-log

## Table of Contents
- [Overview](#overview)
- [Files](#files)
- [Usage examples](#usage-examples)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Check definition](#check-definition)
- [Installation from source](#installation-from-source)
- [Additional notes](#additional-notes)
- [Contributing](#contributing)

## Overview
`sensu-check-log` is a [Sensu Check][2] and log file analyzer plugin for
Sensu Go. The program scans a log file, checks it for matches, and sends
a special failure event to the agent events API when a match is detected.

The check itself will always return a 0 status, unless execution fails for
some reason.

The check must be configured with `stdin: true` so that failure events can
be formed correctly. If the check is not configured with `stdin: true`, then
it will fail to execute.

## Files
`sensu-check-log` requires a log file to be analyzed `-log` and a state file
to track the offset for incremental log analysis `-state`. If the state file
provided by `-state` does not exist, `sensu-check-log` will create one for you.

## Usage examples
```
Usage of sensu-check-log:
  -api-url string
        agent events API URL (default "http://localhost:3031/events")
  -event-status int
        event status on positive match (default 1)
  -ignore-initial-run
        suppresses alerts for any matches found on the first run of the plugin
  -log string
        path to log file (required)
  -match string
        RE2 regexp matcher expression (required)
  -max-bytes int
        max number of bytes to read (0 means unlimited)
  -procs int
        number of parallel analyzer processes (see "Additional Notes" for default)
  -state string
        state file for incremental log analysis (required)
```

## Configuration

### Asset registration

[Sensu Assets][3] are the best way to make use of this plugin. If you're not using an asset, please
consider doing so! If you're using sensuctl 5.13 with Sensu Backend 5.13 or later, you can use the
following command to add the asset:

```
sensuctl asset add sensu/sensu-check-log
```

If you're using an earlier version of sensuctl, you can find the asset on the [Bonsai Asset Index][4].

### Check definition

```yml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: sensu-check-log
  namespace: default
spec:
  command: sensu-check-log -log log.json -state state.json -match critical
  stdin: true
  subscriptions:
  - system
  runtime_assets:
  - sensu/sensu-check-log
```

## Installation from source

The preferred way of installing and deploying this plugin is to use it as an Asset. If you would
like to compile and install the plugin from source or contribute to it, download the latest version
or create an executable script from this source.

From the local path of the sensu-check-log repository:

```
go build
```

## Additional notes

The default for `-procs` is determined by [runtime.NumCPU()][5].
> NumCPU returns the number of logical CPUs usable by the current process.
The set of available CPUs is checked by querying the operating system at process startup.
Changes to operating system CPU allocation after process startup are not reflected.

## Contributing

For more information about contributing to this plugin, see [Contributing][1].

[1]: https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md
[2]: https://docs.sensu.io/sensu-go/latest/reference/checks/
[3]: https://docs.sensu.io/sensu-go/latest/reference/assets/
[4]: https://bonsai.sensu.io/assets/sensu/sensu-check-log
[5]: https://golang.org/pkg/runtime/#NumCPU
