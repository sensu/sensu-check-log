package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	corev2 "github.com/sensu/core/v2"
	"github.com/sensu/sensu-plugin-sdk/sensu"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Config represents the check plugin config.
type Config struct {
	sensu.PluginConfig
	LogFile            string
	LogFileExpr        string
	LogPath            string
	StateDir           string
	Procs              int
	MatchExpr          string
	InvertThresholds   bool
	MaxBytes           int64
	EventsAPI          string
	IgnoreInitialRun   bool
	DisableEvent       bool
	DryRun             bool
	Verbose            bool
	EnableStateReset   bool
	MissingOK          bool
	ForceReadFromStart bool
	WarningThreshold   int
	WarningOnly        bool
	CriticalThreshold  int
	CriticalOnly       bool
	CheckNameTemplate  string
	VerboseResults     bool
}

var (
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-check-log",
			Short:    "Check Log",
			Keyspace: "sensu.io/plugins/sensu-check-log/config",
		},
	}

	options = []sensu.ConfigOption{
		&sensu.PluginConfigOption[string]{
			Path:      "log-file",
			Env:       "CHECK_LOG_FILE",
			Argument:  "log-file",
			Shorthand: "f",
			Usage:     "Log file to check. (Required if --log-file-expr not used)",
			Value:     &plugin.LogFile,
		},
		&sensu.PluginConfigOption[string]{
			Path:      "log-file-expr",
			Env:       "CHECK_LOG_FILE_EXPR",
			Argument:  "log-file-expr",
			Shorthand: "e",
			Usage:     "Log file regexp to check. (Required if --log-file not used)",
			Value:     &plugin.LogFileExpr,
		},
		&sensu.PluginConfigOption[string]{
			Path:      "log-path",
			Env:       "CHECK_LOG_PATH",
			Argument:  "log-path",
			Shorthand: "p",
			Default:   "/var/log/",
			Usage:     "Log path for basis of log file regexp. Only finds files under this path. (Required if --log-file-expr used)",
			Value:     &plugin.LogPath,
		},
		&sensu.PluginConfigOption[string]{
			Path:      "state-directory",
			Env:       "CHECK_LOG_STATE_DIRECTORY",
			Argument:  "state-directory",
			Shorthand: "d",
			Usage:     "Directory where check will hold state for each processed log file. Note: checks using different match expressions should use different state directories to avoid conflict. (Required)",
			Value:     &plugin.StateDir,
		},
		&sensu.PluginConfigOption[int]{
			Path:      "analyzer-procs",
			Env:       "CHECK_LOG_ANALYZER_PROCS",
			Argument:  "analyzer-procs",
			Shorthand: "a",
			Default:   runtime.NumCPU(),
			Usage:     "Number of parallel analyzer processes per file.",
			Value:     &plugin.Procs,
		},
		&sensu.PluginConfigOption[string]{
			Path:      "match-expr",
			Env:       "CHECK_LOG_MATCH_EXPR",
			Argument:  "match-expr",
			Shorthand: "m",
			Usage:     "RE2 regexp matcher expression. (required)",
			Value:     &plugin.MatchExpr,
		},
		&sensu.PluginConfigOption[int]{
			Path:      "warning-threshold",
			Env:       "CHECK_LOG_WARNING_THRESHOLD",
			Argument:  "warning-threshold",
			Shorthand: "w",
			Default:   1,
			Usage:     "Minimum match count that results in an warning",
			Value:     &plugin.WarningThreshold,
		},
		&sensu.PluginConfigOption[bool]{
			Path:      "warning-only",
			Env:       "CHECK_LOG_WARNING_ONLY",
			Argument:  "warning-only",
			Shorthand: "W",
			Usage:     "Only issue warning status if matches are found",
			Value:     &plugin.WarningOnly,
		},
		&sensu.PluginConfigOption[int]{
			Path:      "critical-threshold",
			Env:       "CHECK_LOG_CRITICAL_THRESHOLD",
			Argument:  "critical-threshold",
			Shorthand: "c",
			Default:   5,
			Usage:     "Minimum match count that results in an warning",
			Value:     &plugin.CriticalThreshold,
		},
		&sensu.PluginConfigOption[bool]{
			Path:      "critical-only",
			Env:       "CHECK_LOG_CRITICAL_ONLY",
			Argument:  "critical-only",
			Shorthand: "C",
			Default:   false,
			Usage:     "Only issue critical status if matches are found",
			Value:     &plugin.CriticalOnly,
		},
		&sensu.PluginConfigOption[int64]{
			Path:      "max-bytes",
			Env:       "CHECK_LOG_MAX_BYTES",
			Argument:  "max-bytes",
			Shorthand: "b",
			Usage:     "Max number of bytes to read (0 means unlimited).",
			Value:     &plugin.MaxBytes,
		},
		&sensu.PluginConfigOption[string]{
			Path:      "events-api-url",
			Env:       "CHECK_LOG_EVENTS_API_URL",
			Argument:  "events-api-url",
			Shorthand: "u",
			Default:   "http://localhost:3031/events",
			Usage:     "Agent Events API URL.",
			Value:     &plugin.EventsAPI,
		},
		&sensu.PluginConfigOption[bool]{
			Path:      "ignore-initial-run",
			Env:       "CHECK_LOG_IGNORE_INITIAL_RUN",
			Argument:  "ignore-initial-run",
			Shorthand: "I",
			Usage:     "Suppresses alerts for any matches found on the first run of the plugin.",
			Value:     &plugin.IgnoreInitialRun,
		},
		&sensu.PluginConfigOption[bool]{
			Path:      "missing-ok",
			Env:       "CHECK_LOG_MISSING_OK",
			Argument:  "missing-ok",
			Shorthand: "M",
			Usage:     "Suppresses error if selected log files are missing",
			Value:     &plugin.MissingOK,
		},
		&sensu.PluginConfigOption[string]{
			Path:      "check-name-tamplate",
			Env:       "CHECK_LOG_CHECK_NAME_TEMPLATE",
			Argument:  "check-name-template",
			Shorthand: "t",
			Usage:     "Check name to use in generated events",
			Value:     &plugin.CheckNameTemplate,
		},
		&sensu.PluginConfigOption[bool]{
			Path:      "disable-event-generation",
			Env:       "CHECK_LOG_CHECK_DISABLE_EVENT_GENERATION",
			Argument:  "disable-event-generation",
			Shorthand: "D",
			Usage:     "Disable event generation, send results to stdout instead.",
			Value:     &plugin.DisableEvent,
		},
		&sensu.PluginConfigOption[bool]{
			Path:      "reset-state",
			Env:       "CHECK_LOG_RESET_STATE",
			Argument:  "reset-state",
			Shorthand: "r",
			Usage:     "Allow automatic state reset if match expression changes, instead of failing.",
			Value:     &plugin.EnableStateReset,
		},
		&sensu.PluginConfigOption[bool]{
			Path:      "invert-thresholds",
			Env:       "CHECK_LOG_INVERT_THRESHOLDS",
			Argument:  "invert-thresholds",
			Shorthand: "i",
			Usage:     "Invert warning and critical threshold values, making them minimum values to alert on",
			Value:     &plugin.InvertThresholds,
		},
		&sensu.PluginConfigOption[bool]{
			Path:      "verbose",
			Argument:  "verbose",
			Shorthand: "v",
			Usage:     "Verbose output, useful for testing.",
			Value:     &plugin.Verbose,
		},
		&sensu.PluginConfigOption[bool]{
			Path:      "dry-run",
			Argument:  "dry-run",
			Shorthand: "n",
			Usage:     "Suppress generation of events and report intended actions instead. (implies verbose)",
			Value:     &plugin.DryRun,
		},
		&sensu.PluginConfigOption[bool]{
			Path:      "output-matching-string",
			Argument:  "output-matching-string",
			Shorthand: "",
			Usage:     "Include detailed information about each matching line in output.",
			Value:     &plugin.VerboseResults,
		},
		&sensu.PluginConfigOption[bool]{
			Path:     "force-read-from-start",
			Argument: "force-read-from-start",
			Usage:    "Ignore cached file offset in state directory and read file(s) from beginning.",
			Value:    &plugin.ForceReadFromStart,
		},
	}
)

// State represents the state file offset
type State struct {
	Offset    int64
	MatchExpr string
}

func getState(path string) (state State, err error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, fmt.Errorf("couldn't read state file: %s", err)
	}
	defer func() {
		err = f.Close()
	}()

	if err := gob.NewDecoder(f).Decode(&state); err != nil {
		return state, fmt.Errorf("couldn't read state file: %s", err)
	}

	return state, nil
}

func setState(cur State, path string) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("couldn't write state file: %s", err)
	}
	defer func() {
		e := f.Close()
		if err == nil && e != nil {
			err = fmt.Errorf("couldn't close state file: %s", err)
		}
	}()
	if err := gob.NewEncoder(f).Encode(cur); err != nil {
		return fmt.Errorf("couldn't write state file: %s", err)
	}
	return nil
}

func fatal(formatter string, args ...interface{}) {
	fmt.Printf(formatter, args...)
	os.Exit(2)
}

func checkArgs(event *corev2.Event) (int, error) {
	if plugin.WarningOnly && plugin.CriticalOnly {
		return sensu.CheckStateCritical, fmt.Errorf("--warning-only and --critical-only options conflict, cannot use both")
	}
	if plugin.InvertThresholds {
		if plugin.WarningThreshold <= plugin.CriticalThreshold {
			return sensu.CheckStateCritical, fmt.Errorf("--warning-threshold must be greater than or equal to --critical-threshold when --invert-thresholds is in use")
		}
	} else {
		if plugin.WarningThreshold >= plugin.CriticalThreshold {
			return sensu.CheckStateCritical, fmt.Errorf("--warning-threshold must be less than or equal to --critical-threshold")
		}
	}
	if event == nil && !plugin.DisableEvent {
		return sensu.CheckStateCritical, fmt.Errorf("--disable-event-generation not selected but event missing from stdin")
	}
	if plugin.LogFileExpr == "" && plugin.LogFile == "" {
		return sensu.CheckStateCritical, fmt.Errorf("at least one of --log-file or --log-file-expr must be specified")
	}
	if plugin.LogFileExpr != "" && plugin.LogPath == "" {
		return sensu.CheckStateCritical, fmt.Errorf("--log-path must be specified if --log-file-expr is used")
	}
	if plugin.StateDir == "" {
		return sensu.CheckStateCritical, fmt.Errorf("--state-directory not specified")
	}
	if plugin.MatchExpr == "" {
		return sensu.CheckStateCritical, fmt.Errorf("--match-expr not specified")
	}
	if plugin.DryRun {
		plugin.Verbose = true
		fmt.Printf("LogFileExpr: %s StateDir: %s\n", plugin.LogFileExpr, plugin.StateDir)
	}
	return sensu.CheckStateOK, nil
}

func testStdin() (bool, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		fmt.Printf("Error accessing stdin: %v\n", err)
		return false, err
	}
	//Check the Mode bitmask for Named Pipe to indicate stdin is connected
	if fi.Mode()&os.ModeNamedPipe != 0 {
		return true, nil
	}
	return false, nil
}

func main() {
	useStdin, err := testStdin()
	if err != nil {
		panic(err)
	}
	//check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, useStdin)
	check := sensu.NewCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, useStdin)
	//fmt.Println("Check==", check.)
	check.Execute()
}

func removeDuplicates(elements []string) []string { // change string to int here if required
	// Use map to record duplicates as we find them.
	encountered := map[string]bool{} // change string to int here if required
	result := []string{}             // change string to int here if required

	for v := range elements {
		if encountered[elements[v]] {
			// Do not add duplicate.
		} else {
			// Record this element as an encountered element.
			encountered[elements[v]] = true
			// Append to result slice.
			result = append(result, elements[v])
		}
	}
	// Return the new slice.
	return result
}

func buildLogArray() ([]string, error) {
	logs := []string{}
	var e error
	if plugin.LogFile != "" {
		absPath, e := filepath.Abs(plugin.LogFile)
		if e != nil {
			return nil, e
		}
		if filepath.IsAbs(absPath) {
			logs = append(logs, absPath)
		} else {
			return nil, fmt.Errorf("path %s not absolute", absPath)
		}

	}
	if len(plugin.LogPath) > 0 && len(plugin.LogFileExpr) > 0 {
		logRegExp, e := regexp.Compile(plugin.LogFileExpr)
		if e != nil {
			return nil, e
		}
		absLogPath, _ := filepath.Abs(plugin.LogPath)
		if plugin.Verbose {
			fmt.Printf("Searching for matching file names in: %v\n", absLogPath)
		}

		if filepath.IsAbs(absLogPath) {
			e = filepath.Walk(absLogPath, func(path string, info os.FileInfo, err error) error {
				if err == nil && logRegExp.MatchString(path) {
					if filepath.IsAbs(path) {
						if !info.IsDir() {
							logs = append(logs, path)
						}
					} else {
						return fmt.Errorf("path %s not absolute", path)
					}
				}
				return nil
			})
			if e != nil {
				return nil, e
			}
		}
	}
	logs = removeDuplicates(logs)
	if plugin.Verbose {
		fmt.Printf("Log file array to process: %v\n", logs)
	}
	return logs, e
}

func processLogFile(file string, enc *json.Encoder) (int, error) {
	if !filepath.IsAbs(file) {
		return 0, fmt.Errorf("error file %s: is not absolute path", file)
	}
	if plugin.Verbose {
		fmt.Printf("Now Processing: %v\n", file)
	}
	f, err := os.Open(file)
	if err != nil {
		if plugin.MissingOK {
			return 0, nil
		} else {
			return 0, fmt.Errorf("error couldn't open log file %s: %s", file, err)
		}

	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("error couldn't close log file %s: %s\n", file, err)
		}
	}()

	stateFile := filepath.Join(plugin.StateDir, strings.ReplaceAll(file, string(os.PathSeparator), string("_")))
	if plugin.Verbose {
		fmt.Printf("stateFile: %s\n", stateFile)
	}
	state, err := getState(stateFile)
	if err != nil {
		return 0, fmt.Errorf("error couldn't get state for log file %s: %s", file, err)

	}
	resetState := false
	if state.MatchExpr != "" && state.MatchExpr != plugin.MatchExpr {
		resetState = true
	}
	if resetState {
		if plugin.EnableStateReset {
			state = State{}
			if plugin.Verbose {
				fmt.Printf("info: resetting state file %s because unexpected cached matching condition detected and --reset-state in use\n", file)
			}
		} else {
			return 0, fmt.Errorf("error: state file for %s has unexpected cached matching condition:: Expr: '%s'. Either use --reset-state option, or manually delete state file '%s'", file, state.MatchExpr, stateFile)
		}
	}

	info, err := f.Stat()
	if err != nil {
		return 0, fmt.Errorf("error couldn't get info for file %s: %s", file, err)
	}
	// supress alerts on first run (when state file is empty) only when configured (with -ignore-initial-run)
	if state == (State{}) && plugin.IgnoreInitialRun {
		state.Offset = int64(info.Size())
		state.MatchExpr = plugin.MatchExpr
		if err := setState(state, stateFile); err != nil {
			return 0, fmt.Errorf("error couldn't set state for log file %s: %s", file, err)
		}
		return 0, nil
	}

	offset := state.Offset
	if plugin.ForceReadFromStart {
		offset = 0
	}
	// Are we looking at freshly rotated file since last time we run?
	// If so let's reset the offset back to 0 and read the file again
	if offset < 0 {
		return 0, fmt.Errorf("error file %s: cached offset is less than 0, possibly corrupt state file: %s", file, stateFile)
	}
	if offset > info.Size() {
		if plugin.Verbose {
			fmt.Printf("Resetting offset to zero, because cached offset (%v bytes) is beyond end of file (%v bytes), indicating file has been rotated, truncated or replaced\n", offset, info.Size())
		}
		offset = 0
	}

	if offset > 0 {
		if offset == info.Size() {
			if plugin.Verbose {
				fmt.Printf("Cached offset in state directory for %s indicates file not updated since last read\n", file)
			}
			return 0, nil
		} else {
			if _, err := f.Seek(offset, io.SeekStart); err != nil {
				return 0, fmt.Errorf("error couldn't seek file %s to offset %d: %s", file, offset, err)

			}
		}
	}

	var reader io.Reader = f
	if plugin.MaxBytes > 0 {
		reader = io.LimitReader(f, plugin.MaxBytes)
	}

	analyzer := Analyzer{
		Path:           file,
		Procs:          plugin.Procs,
		Log:            reader,
		Offset:         offset,
		Func:           AnalyzeRegexp(plugin.MatchExpr),
		VerboseResults: plugin.VerboseResults,
	}

	status := sensu.CheckStateOK
	results := analyzer.Go(context.Background())
	numResults := 0
	for result := range results {
		if result.Err != nil {
			status = sensu.CheckStateCritical
		}
		if err := enc.Encode(result); err != nil {
			return 0, fmt.Errorf("error couldn't encode result %+v for file %s: %s", result, result.Path, err)
		}
		numResults++
	}
	if plugin.Verbose {
		fmt.Printf("File %s Match Count: %v\n", file, numResults)
	}
	bytesRead := analyzer.BytesRead()
	state.Offset = int64(offset + bytesRead)
	state.MatchExpr = plugin.MatchExpr
	if plugin.Verbose {
		fmt.Printf("File %s Match Status %v BytesRead: %v"+
			" New Offset: %v\n", file, status, bytesRead, state.Offset)
	}

	if err := setState(state, stateFile); err != nil {
		return 0, fmt.Errorf("error setting state: %s", err)
	}
	return numResults, nil
}

func setStatus(currentStatus int, numMatches int) int {
	status := sensu.CheckStateOK
	warn := false
	critical := false
	if plugin.InvertThresholds {
		if plugin.WarningThreshold > 0 && numMatches <= plugin.WarningThreshold {
			warn = true
		}
		if plugin.CriticalThreshold > 0 && numMatches <= plugin.CriticalThreshold {
			critical = true
		}
	} else {
		if plugin.WarningThreshold > 0 && numMatches >= plugin.WarningThreshold {
			warn = true
		}
		if plugin.CriticalThreshold > 0 && numMatches >= plugin.CriticalThreshold {
			critical = true
		}
	}

	if plugin.WarningOnly || plugin.CriticalOnly {
		if plugin.WarningOnly && warn {
			status = sensu.CheckStateWarning
		}
		if plugin.CriticalOnly && critical {
			status = sensu.CheckStateCritical
		}
	} else {
		if warn {
			status = sensu.CheckStateWarning
		}
		if critical {
			status = sensu.CheckStateCritical
		}
	}
	if status > currentStatus {
		return status
	} else {
		return currentStatus
	}
}

func executeCheck(event *corev2.Event) (int, error) {
	var status int
	status = 0

	//create state directory if not existing already
	if _, err := os.Stat(plugin.StateDir); errors.Is(err, os.ErrNotExist) {
		//creating recursive directories incase
		err := os.MkdirAll(plugin.StateDir, os.ModePerm)
		if err != nil {
			return sensu.CheckStateCritical, fmt.Errorf("selected --state-directory %s does not exist and cannot be created.Expected a correct Path to create/reach the directory", plugin.StateDir)
		}
	}
	if _, err := os.Stat(plugin.StateDir); err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("unexpected error accessing --state-directory %s: %s", plugin.StateDir, err)
	}

	logs, e := buildLogArray()
	if e != nil {
		return sensu.CheckStateCritical, e
	}
	fileErrors := []error{}
	matchingFiles := make(map[string]int)
	eventBuf := new(bytes.Buffer)
	enc := json.NewEncoder(eventBuf)

	for _, file := range logs {
		numMatches, err := processLogFile(file, enc)
		if numMatches >= 0 {
			matchingFiles[file] = numMatches
		}
		if err != nil {
			fileErrors = append(fileErrors, err)
			status = sensu.CheckStateOK
			continue
		}
		status = setStatus(status, numMatches)

	} // end of loop over log files
	if len(fileErrors) > 0 {
		for _, e := range fileErrors {
			fmt.Printf("%v\n", e)
		}
		return sensu.CheckStateCritical, nil
	}
	// sendEvent or report to stdout
	if status != sensu.CheckStateOK {
		//use summary output unless VerboseResults is true
		output := ""
		if plugin.VerboseResults {
			output = fmt.Sprintf("%s\n", eventBuf.String())
		} else {
			for f, n := range matchingFiles {
				output = output + fmt.Sprintf("File %s has %d matching lines\n", f, n)
			}
		}
		//if event generation disabled just output the results as this check's output
		if plugin.DisableEvent {
			fmt.Printf("%s", output)
			return status, nil
		}

		// proceed with event generation
		if event == nil {
			fmt.Printf("Error: Input event not defined. Event generation aborted\n")
			return sensu.CheckStateWarning, nil
		}
		if len(plugin.EventsAPI) == 0 {
			fmt.Printf("Error: Event API url not defined. Event generation aborted\n")
			return sensu.CheckStateWarning, nil
		}
		outputEvent, err := createEvent(event, status, plugin.CheckNameTemplate, output)
		if err != nil {
			fmt.Printf("Error creating event: %s\n", err)
			return sensu.CheckStateWarning, nil
		}

		// if --dry-run selected lets report what we would have sent instead of sending.
		if plugin.DryRun {
			fmt.Printf("Dry-run enabled, event to send:\n%+v\n", outputEvent)
		} else {
			if err := sendEvent(plugin.EventsAPI, outputEvent); err != nil {
				fmt.Printf("Error sending event: %s\n", err)
				return sensu.CheckStateWarning, nil
			}
		}
	}

	return sensu.CheckStateOK, nil
}
