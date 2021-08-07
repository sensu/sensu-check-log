package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/sensu/sensu-plugin-sdk/sensu"
)

// Config represents the check plugin config.
type Config struct {
	sensu.PluginConfig
	LogFile           string
	LogFileExpr       string
	LogPath           string
	StateDir          string
	Procs             int
	MatchExpr         string
	MatchStatus       int
	InverseMatch      bool
	MaxBytes          int64
	EventsAPI         string
	IgnoreInitialRun  bool
	DisableEvent      bool
	DryRun            bool
	Verbose           bool
	EnableStateReset  bool
	CheckNameTemplate string
}

var (
	defaultNameTemplate = "{{ .Check.Name }}-alert"
	logs                = []string{}
	plugin              = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-check-log",
			Short:    "Check Log",
			Keyspace: "sensu.io/plugins/sensu-check-log/config",
		},
	}

	options = []*sensu.PluginConfigOption{
		&sensu.PluginConfigOption{
			Path:      "log-file",
			Env:       "CHECK_LOG_FILE",
			Argument:  "log-file",
			Shorthand: "f",
			Default:   "",
			Usage:     "Log file to check. (Required if --log-file-expr not used)",
			Value:     &plugin.LogFile,
		},
		&sensu.PluginConfigOption{
			Path:      "log-file-expr",
			Env:       "CHECK_LOG_FILE_EXPR",
			Argument:  "log-file-expr",
			Shorthand: "e",
			Default:   "",
			Usage:     "Log file regexp to check. (Required if --log-file not used)",
			Value:     &plugin.LogFileExpr,
		},
		&sensu.PluginConfigOption{
			Path:      "log-path",
			Env:       "CHECK_LOG_PATH",
			Argument:  "log-path",
			Shorthand: "p",
			Default:   "/var/log/",
			Usage:     "Log path for basis of log file regexp. Only finds files under this path. (Required if --log-file-expr used)",
			Value:     &plugin.LogPath,
		},
		&sensu.PluginConfigOption{
			Path:      "state-directory",
			Env:       "CHECK_LOG_STATE_DIRECTORY",
			Argument:  "state-directory",
			Shorthand: "d",
			Default:   "",
			Usage:     "Directory where check will hold state for each processed log file. Note: checks using different match expressions should use different state directories to avoid conflict. (Required)",
			Value:     &plugin.StateDir,
		},
		&sensu.PluginConfigOption{
			Path:      "analyzer-procs",
			Env:       "CHECK_LOG_ANALYZER_PROCS",
			Argument:  "analyzer-procs",
			Shorthand: "a",
			Default:   runtime.NumCPU(),
			Usage:     "Number of parallel analyzer processes per file.",
			Value:     &plugin.Procs,
		},
		&sensu.PluginConfigOption{
			Path:      "match-expr",
			Env:       "CHECK_LOG_MATCH_EXPR",
			Argument:  "match-expr",
			Shorthand: "m",
			Default:   "",
			Usage:     "RE2 regexp matcher expression. (required)",
			Value:     &plugin.MatchExpr,
		},
		&sensu.PluginConfigOption{
			Path:      "match-event-status",
			Env:       "CHECK_LOG_MATCH_EVENT_STATUS",
			Argument:  "match-event-status",
			Shorthand: "s",
			Default:   1,
			Usage:     "Event status to return on match in generated event.",
			Value:     &plugin.MatchStatus,
		},
		&sensu.PluginConfigOption{
			Path:      "max-bytes",
			Env:       "CHECK_LOG_MAX_BYTES",
			Argument:  "max-bytes",
			Shorthand: "b",
			Default:   int64(0),
			Usage:     "Max number of bytes to read (0 means unlimited).",
			Value:     &plugin.MaxBytes,
		},
		&sensu.PluginConfigOption{
			Path:      "events-api-url",
			Env:       "CHECK_LOG_EVENTS_API_URL",
			Argument:  "events-api-url",
			Shorthand: "u",
			Default:   "http://localhost:3031/events",
			Usage:     "Agent Events API URL.",
			Value:     &plugin.EventsAPI,
		},
		&sensu.PluginConfigOption{
			Path:      "ignore-initial-run",
			Env:       "CHECK_LOG_IGNORE_INITIAL_RUN",
			Argument:  "ignore-initial-run",
			Shorthand: "I",
			Default:   false,
			Usage:     "Suppresses alerts for any matches found on the first run of the plugin.",
			Value:     &plugin.IgnoreInitialRun,
		},
		&sensu.PluginConfigOption{
			Path:      "check-name-tamplate",
			Env:       "CHECK_LOG_CHECK_NAME_TEMPLATE",
			Argument:  "check-name-template",
			Shorthand: "t",
			Default:   defaultNameTemplate,
			Usage:     "Check name to use in generated events",
			Value:     &plugin.CheckNameTemplate,
		},
		&sensu.PluginConfigOption{
			Path:      "disable-event-generation",
			Env:       "CHECK_LOG_CHECK_DISABLE_EVENT_GENERATION",
			Argument:  "disable-event-generation",
			Shorthand: "D",
			Default:   false,
			Usage:     "Disable event generation, send results to stdout instead.",
			Value:     &plugin.DisableEvent,
		},
		&sensu.PluginConfigOption{
			Path:      "reset-state",
			Env:       "CHECK_LOG_RESET_STATE",
			Argument:  "reset-state",
			Shorthand: "r",
			Default:   false,
			Usage:     "Allow automatic state reset if match expression changes, instead of failing.",
			Value:     &plugin.EnableStateReset,
		},
		&sensu.PluginConfigOption{
			Path:      "inverse-match",
			Env:       "CHECK_LOG_INVERSE_MATCH",
			Argument:  "inverse-match",
			Shorthand: "i",
			Default:   false,
			Usage:     "Inverse match, only generate alert event if no lines match.",
			Value:     &plugin.InverseMatch,
		},
		&sensu.PluginConfigOption{
			Path:      "verbose",
			Argument:  "verbose",
			Shorthand: "v",
			Default:   false,
			Usage:     "Verbose output, useful for testing.",
			Value:     &plugin.Verbose,
		},
		&sensu.PluginConfigOption{
			Path:      "dry-run",
			Argument:  "dry-run",
			Shorthand: "n",
			Default:   false,
			Usage:     "Suppress generation of events and report intended actions instead. (implies verbose)",
			Value:     &plugin.DryRun,
		},
	}
)

// State represents the state file offset
type State struct {
	Offset       int64
	MatchExpr    string
	InverseMatch bool
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
	if event == nil && !plugin.DisableEvent {
		return sensu.CheckStateCritical, fmt.Errorf("--disable-event-generation not selected but event missing from stdin")
	}
	if plugin.LogFileExpr == "" && plugin.LogFile == "" {
		return sensu.CheckStateCritical, fmt.Errorf("At least one of --log-file or --log-file-expr must be specified")
	}
	if plugin.LogFileExpr != "" && plugin.LogPath == "" {
		return sensu.CheckStateCritical, fmt.Errorf("--log-path must be specified if --log-file-expr is used")
	}
	if plugin.StateDir == "" {
		return sensu.CheckStateCritical, fmt.Errorf("--state-directory not specified")
	}
	_, err := os.Stat(plugin.StateDir)
	if errors.Is(err, os.ErrNotExist) {
		return sensu.CheckStateCritical, fmt.Errorf("selected --state-directory %s does not exist", plugin.StateDir)
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
	check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, useStdin)
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

func buildLogArray() error {
	var e error
	if plugin.LogFile != "" {
		absPath, e := filepath.Abs(plugin.LogFile)
		if e != nil {
			return e
		}
		if filepath.IsAbs(absPath) {
			logs = append(logs, absPath)
		} else {
			return fmt.Errorf("Path %s not absolute", absPath)
		}

	}
	if len(plugin.LogPath) > 0 && len(plugin.LogFileExpr) > 0 {
		logRegExp, e := regexp.Compile(plugin.LogFileExpr)
		if e != nil {
			return e
		}
		absLogPath, _ := filepath.Abs(plugin.LogPath)

		if filepath.IsAbs(absLogPath) {
			e = filepath.Walk(absLogPath, func(path string, info os.FileInfo, err error) error {
				if err == nil && logRegExp.MatchString(info.Name()) {
					if filepath.IsAbs(path) {
						logs = append(logs, path)
					} else {
						return fmt.Errorf("Path %s not absolute", path)
					}
				}
				return nil
			})
			if e != nil {
				return e
			}
		}
	}
	logs = removeDuplicates(logs)
	if plugin.Verbose {
		fmt.Printf("Log file array to process: %v", logs)
	}
	return e
}

func processLogFile(file string, enc *json.Encoder) (int, error) {
	if !filepath.IsAbs(file) {
		return sensu.CheckStateCritical, fmt.Errorf("error file %s: is not absolute path", file)
	}
	if plugin.Verbose {
		fmt.Printf("Processing: %v", file)
	}
	f, err := os.Open(file)
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("error couldn't open log file %s: %s", file, err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("error couldn't close log file %s: %s", file, err)
		}
	}()

	stateFile := filepath.Join(plugin.StateDir, strings.ReplaceAll(file, string(os.PathSeparator), string("_")))
	if plugin.Verbose {
		fmt.Println("stateFile", stateFile)
	}
	state, err := getState(stateFile)
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("error couldn't get state for log file %s: %s", file, err)

	}
	// Do we need to reset the state because the requested MatchExpr or InverseMatch is different?
	resetState := false
	if state.MatchExpr != "" && state.MatchExpr != plugin.MatchExpr {
		resetState = true
	}
	if state.MatchExpr != "" && state.InverseMatch != plugin.InverseMatch {
		resetState = true
	}
	if resetState {
		if plugin.EnableStateReset {
			state = State{}
			if plugin.Verbose {
				fmt.Printf("Info: resetting state file %s because unexpected cached matching condition detected and --reset-state in use", file)
			}
		} else {
			return sensu.CheckStateCritical, fmt.Errorf("Error: state file for %s has unexpected cached matching condition:: Expr: %s Inverse: %v\nEither use --reset-state option, or manually delete state file %s", file, state.MatchExpr, state.InverseMatch, stateFile)
		}
	}
	info, err := f.Stat()
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("error couldn't get info for file %s: %s", file, err)
	}
	// supress alerts on first run (when state file is empty) only when configured (with -ignore-initial-run)
	if state == (State{}) && plugin.IgnoreInitialRun {
		state.Offset = int64(info.Size())
		state.MatchExpr = plugin.MatchExpr
		if err := setState(state, stateFile); err != nil {
			return sensu.CheckStateCritical, fmt.Errorf("error couldn't set state for log file %s: %s", file, err)
		}
		return sensu.CheckStateOK, nil
	}

	offset := state.Offset
	// Are we looking at freshly rotated file since last time we run?
	// If so let's reset the offset back to 0 and read the file again
	if offset >= info.Size() {
		offset = 0
		if plugin.Verbose {
			fmt.Printf("Resetting offset to zero, because cached offset is beyond end of file and modtime is newer than last time processed")
		}
	}

	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return sensu.CheckStateCritical, fmt.Errorf("error couldn't seek file %s to offset %d: %s", file, offset, err)

		}
	}

	var reader io.Reader = f
	if plugin.MaxBytes > 0 {
		reader = io.LimitReader(f, plugin.MaxBytes)
	}

	analyzer := Analyzer{
		Path:  file,
		Procs: plugin.Procs,
		Log:   reader,
		Func:  AnalyzeRegexp(plugin.MatchExpr),
	}

	status := sensu.CheckStateOK
	results := analyzer.Go(context.Background())

	for result := range results {
		if result.Err != nil {
			status = sensu.CheckStateCritical
		}
		if err := enc.Encode(result); err != nil {
			return sensu.CheckStateCritical, fmt.Errorf("error couldn't encode result %+v for file %s: %s", result, result.Path, err)
		}
		if status < plugin.MatchStatus {
			status = plugin.MatchStatus
		}
	}
	if plugin.Verbose {
		fmt.Printf("File %s Match Status %v", file, status)
	}
	bytesRead := analyzer.BytesRead()
	state.Offset = int64(offset + bytesRead)
	state.MatchExpr = plugin.MatchExpr
	if plugin.Verbose {
		fmt.Printf("File %s Match Status %v BytesRead: %v New Offset: %v", file, status, bytesRead, state.Offset)
	}

	if err := setState(state, stateFile); err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("Error setting state: %s", err)
	}
	return status, nil
}

func executeCheck(event *corev2.Event) (int, error) {
	var status int
	e := buildLogArray()
	if e != nil {
		return sensu.CheckStateCritical, e
	}
	fileErrors := []error{}

	eventBuf := new(bytes.Buffer)
	enc := json.NewEncoder(eventBuf)

	status = sensu.CheckStateOK

	for _, file := range logs {
		fileStatus, err := processLogFile(file, enc)
		if err != nil {
			fileErrors = append(fileErrors, err)
			status = fileStatus
			continue
		}
		if status < fileStatus {
			status = fileStatus
		}
	} // end of loop over log files
	if len(fileErrors) > 0 {
		for _, e := range fileErrors {
			fmt.Printf("%v", e)
		}
		return status, nil
	}

	// sendEvent or report to stdout
	if status != sensu.CheckStateOK {
		//if event generation disabled just output the results as this check's output
		if plugin.DisableEvent {
			fmt.Printf("%s\n", eventBuf.String())
			return status, nil
		}

		// proceed with event generation
		if event == nil {
			fmt.Printf("Error: Input event not defined. Event generation aborted")
			return sensu.CheckStateWarning, nil
		}
		if len(plugin.EventsAPI) == 0 {
			fmt.Printf("Error: Event API url not defined. Event generation aborted")
			return sensu.CheckStateWarning, nil
		}
		outputEvent, err := createEvent(event, status, plugin.CheckNameTemplate, eventBuf.String())
		if err != nil {
			fmt.Printf("Error creating event: %s", err)
			return sensu.CheckStateWarning, nil
		}

		// if --dry-run selected lets report what we would have sent instead of sending.
		if plugin.DryRun {
			fmt.Printf("Dry-run enabled, event to send:\n%+v", outputEvent)
		} else {
			if err := sendEvent(plugin.EventsAPI, outputEvent); err != nil {
				fmt.Printf("Error sending event: %s", err)
				return sensu.CheckStateWarning, nil
			}
		}
	}

	return sensu.CheckStateOK, nil
}
