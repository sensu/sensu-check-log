# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.2] - 2021-10-07

### Fixed
* Incorrect handling of file processing when cached file offset corresponds to end-of-file.


## [0.4.1] - 2021-09-20

### Fixed
* --log-file-expr  will match directory elements in fully qualified file path instead of just filename 

## [0.4.0] - 2021-09-10

### Breaking Changes
* Removed --match-event-status argument and replaced with match number based status controlled by --critical-threshold and --warning-threshold 
* Will now attempt to create state directory if it doesnt not exist

### Added
* Added --missing-ok to suppress errors if requested log file not found 
* Added --critical-threshold to set matching number needed to raise critical event status
* Added --warning-threshold to set matching number needed to raise warning event status
* Added --warning-only to make sure only warning event status is sent (even if critical threshold reached)
* Added --critical-only to make sure only critical event status is sent

### Fixed
* Will now correctly traverse subdirectories of --log-path when looking for file names that match the regexp provided by --log-file-expr   
* Silenced annotation override information messages when check annotations are used.

## [0.3.0] - 2021-08-06

### Breaking Changes
#### Refactored to use sensu plugin sdk, notable breaking changes:
* cmdline arguments now support double dash  and short options
* Now uses state directory

### Added
* New support for optional regex log file matching

* Support for alert on inverse of matching regex

* new support for error reporting if requested log file(s) are missing

## [0.2.0] - 2020-08-21

### Added
* Add `-ignore-initial-run` flag to suppress alerts on the first run.

### Changed
* Update README.md with more context and examples.

## [0.1.2] - 2019-07-01

### Added
* Add CHANGELOG.md
* Add .bonsai.yml
* Add .travis.yml

### Removed
* Remove .circleci

## [0.1.1] - 2019-07-01

### Added
* Additional build targets

## [0.1.0] - 2017-07-01

### Added
* Initial release
