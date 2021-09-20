[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/sensu/sensu-check-log)
![Go Test](https://github.com/sensu/sensu-check-log/workflows/Go%20Test/badge.svg)
![goreleaser](https://github.com/sensu/sensu-check-log/workflows/goreleaser/badge.svg)


# sensu-check-log

## Table of Contents
- [Overview](#overview)
- [Usage examples](#usage-examples)
  - [Help output](#help-output)
  - [Environment variables](#environment-variables)
  - [Event generation](#event-generation)
  - [Annotations](#annotations)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Check definition](#check-definition)
- [Installation from source](#installation-from-source)
- [Additional notes](#additional-notes)
- [Contributing](#contributing)

## Overview

`sensu-check-log` is a [Sensu Check][6] and log file analyzer plugin for
Sensu Go. The program scans a set of log files, checks for matches, and sends
a special alert event to the agent events API when a match is detected.
 
The check itself will return a 0 status, unless execution fails for
some reason (ex: if one of the files can not be read)


## Usage examples

### sensu-check-log
```
Check Log

Usage:
  sensu-check-log [flags]
  sensu-check-log [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -d, --state-directory string       Directory where check will hold state for each processed log file. Note: checks using different match expressions should use different state directories to avoid conflict. (Required)
  -f, --log-file string              Log file to check. (Required if --log-file-expr not used)
  -e, --log-file-expr string         Log file regexp to check. (Required if --log-file not used)
  -m, --match-expr string            RE2 regexp matcher expression. (required)
  -p, --log-path string              Log path for basis of log file regexp. Only finds files under this path. (Required if --log-file-expr used) (default "/var/log/")
  -W, --warning-only                 Only issue warning status if matches are found
  -w, --warning-threshold int        Minimum match count that results in an warning (default 1)
  -C, --critical-only                Only issue critical status if matches are found
  -c, --critical-threshold int       Minimum match count that results in an warning (default 5)
  -b, --max-bytes int                Max number of bytes to read (0 means unlimited).
  -a, --analyzer-procs int           Number of parallel analyzer processes per file. 
  -t, --check-name-template string   Check name to use in generated events (default "{{ .Check.Name }}-alert")
  -u, --events-api-url string        Agent Events API URL. (default "http://localhost:3031/events")
  -D, --disable-event-generation     Disable event generation, send results to stdout instead.
  -I, --ignore-initial-run           Suppresses alerts for any matches found on the first run of the plugin.
  -M, --missing-ok                   Suppresses error if selected log files are missing 
  -i, --inverse-match                Inverse match, only generate alert event if no lines match.
  -r, --reset-state                  Allow automatic state reset if match expression changes, instead of failing.
  -n, --dry-run                      Suppress generation of events and report intended actions instead. (implies verbose)
  -v, --verbose                      Verbose output, useful for testing.
  -h, --help                         help for sensu-check-log
```

### Environment variables

|Argument                   |Environment Variable               |
|---------------------------|-----------------------------------|
|--state-directory          |CHECK_LOG_STATE_DIRECTORY          |
|--log-file                 |CHECK_LOG_FILE                     |
|--log-file-expr            |CHECK_LOG_FILE_EXPR                |
|--log-path                 |CHECK_LOG_PATH                     |
|--match-expr               |CHECK_LOG_MATCH_EXPR               |
|--warning-only             |CHECK_LOG_WARNING_ONLY             |
|--warning-threshold        |CHECK_LOG_WARNING_THRESHOLD        |
|--critical-only            |CHECK_LOG_CRITICAL_ONLY            |
|--critical-threshold       |CHECK_LOG_CRITICAL_THRESHOLD       |
|--max-bytes                |CHECK_LOG_MAX_BYTES                |
|--analyzer-procs           |CHECK_LOG_ANALYZER_PROCS           |
|--check-name-template      |CHECK_LOG_CHECK_NAME_TEMPLATE      |
|--events-api-url           |CHECK_LOG_EVENTS_API_URL           |
|--disable-event-generation |CHECK_LOG_DISABLE_EVENT_GENERATION |
|--ignore-initial-run       |CHECK_LOG_IGNORE_INITIAL_RUN       |
|--missing-ok               |CHECK_LOG_MISSING_OK               |
|--inverse-match            |CHECK_LOG_INVERSE_MATCH            |
|--reset-state              |CHECK_LOG_RESET_STATE              |

### Event generation

By default, sensu-check-log will attempt to create a new alert event if a log match 
is found for any of the files selected to be checked. This makes it possible for the check
to run repeatedly without automatically resolving alerts generated from previously found 
log matches.  The primary event associated with the sensu-check-log can still be used to
detect operational faults such as a missing log file, or errors writing into the state directory.

The generated alert event is created using the local Sensu agent's event api url.
You can disable event generation by using `--disable-event-generation` or `--dry-run` arguments

**Note**: Event generation requires Sensu Go check configuration `stdin:true`

#### Check Name Template

This check provides options for using a golang template aware string to populate the check name in the generated event. 
By default the check name is populated using a template that modifies the calling check name from the event passed into the command from stdin. 
More information on template syntax and format can be found in [the documentation][9]

### Annotations

All arguments for these checks are tunable on a per entity or check basis based
on annotations. The annotations keyspace for this collection of checks is
`sensu.io/plugins/sensu-check-log/config`.  You can make use of annotation overrides
when the check is configured with stdin: true.

**NOTE**: Due to [check token substituion][14], supplying a template value such
as for `check-name-template` as a check annotation requires that you place the
desired template as a [golang string literal][13] (enlcosed in backticks)
within another template definition.  This does not apply to entity annotations.

#### Examples

To customize the event api url as an entity annotation, you could use a
sensu-agent configuration snippet similar to this:

```yml
# /etc/sensu/agent.yml example
annotations:
  sensu.io/plugins/sensu-check-log/config/events-api-url: 'http://127.0.0.1:7342'
```


## Configuration

### Asset registration

[Sensu Assets][10] are the best way to make use of this plugin. If you're not using an asset, please
consider doing so! If you're using sensuctl 5.13 with Sensu Backend 5.13 or later, you can use the
following command to add the asset:

```
sensuctl asset add sensu/sensu-check-log
```

If you're using an earlier version of sensuctl, you can find the asset on the [Bonsai Asset Index][https://bonsai.sensu.io/assets/sensu/sensu-check-log].

### Check definition

#### sensu-check-log

Example of configuring a check configuration to match the word 'error' in a case-insensitive manner using [RE compatible regexp syntax][11]

```yml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: sensu-check-log
spec:
  command: sensu-check-log -f /var/log/messages.log -m "(?i)error" -d /tmp/sensu-check-log-error/
  stdin: true
  runtime_assets:
  - sensu/sensu-check-log

```

Example of configuring a check configuration to match lines without the word 'success' in a case-insensitive manner using [RE compatible regexp syntax][11]

```yml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: sensu-check-log
spec:
  command: sensu-check-log -f /var/log/messages.log -m "(?i)success" -i -d /tmp/sensu-check-log-not-success/
  stdin: true
  runtime_assets:
  - sensu/sensu-check-log

```

Example of configuring a check configuration to match lines with the word 'error' in a case-insensitive manner for all log filepaths under `/var/log` ending with `webserver-.*/access.log` files using [RE compatible regexp syntax][11]

```yml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: sensu-check-log
spec:
  command: sensu-check-log -p /var/log/ -e "webserver-.*/access.log$" -m "(?i)error" -d /tmp/sensu-check-access-log-error/
  stdin: true
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

## Contributing

For more information about contributing to this plugin, see [Contributing][1].

[1]: https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md
[2]: https://github.com/sensu-community/sensu-plugin-sdk
[3]: https://github.com/sensu-plugins/community/blob/master/PLUGIN_STYLEGUIDE.md
[4]: https://github.com/sensu-community/check-plugin-template/blob/master/.github/workflows/release.yml
[5]: https://github.com/sensu-community/check-plugin-template/actions
[6]: https://docs.sensu.io/sensu-go/latest/reference/checks/
[7]: https://github.com/sensu-community/check-plugin-template/blob/master/main.go
[8]: https://bonsai.sensu.io/
[9]: https://github.com/sensu-community/sensu-plugin-tool
[10]: https://docs.sensu.io/sensu-go/latest/reference/assets/
[11]: https://github.com/google/re2
[12]: https://docs.sensu.io/sensu-go/latest/observability-pipeline/observe-process/handler-templates/
[13]: https://golang.org/ref/spec#String_literals
[14]: https://docs.sensu.io/sensu-go/latest/observability-pipeline/observe-schedule/checks/#check-token-substitution
