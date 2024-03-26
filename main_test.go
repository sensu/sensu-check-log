package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	corev2 "github.com/sensu/core/v2"
	"github.com/stretchr/testify/assert"
)

type apiHandler struct {
	t     *testing.T
	event *corev2.Event
}

func (h apiHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !strings.HasSuffix(req.URL.Path, "/events") {
		http.Error(w, "not found", 404)
		return
	}
}

func clearPlugin() {
	plugin.LogFile = ""
	plugin.StateDir = ""
	plugin.MatchExpr = ""
	plugin.LogFileExpr = ""
	plugin.LogPath = ""
	plugin.StateDir = ""
	plugin.DryRun = false
	plugin.IgnoreInitialRun = false
	plugin.InvertThresholds = false
	plugin.WarningOnly = false
	plugin.CriticalOnly = false
	plugin.WarningThreshold = 0
	plugin.CriticalThreshold = 0
}

func TestStdin(t *testing.T) {
	clearPlugin()
	test, err := testStdin()
	assert.NoError(t, err)
	assert.Equal(t, false, test)
}

func TestSettStatusWithCurrentStatusZero(t *testing.T) {
	clearPlugin()
	s := 0
	numMatches := 10
	status := setStatus(s, numMatches)
	assert.Equal(t, 0, status)
	plugin.WarningThreshold = 1
	status = setStatus(s, numMatches)
	assert.Equal(t, 1, status)
	plugin.WarningThreshold = 11
	status = setStatus(s, numMatches)
	assert.Equal(t, 0, status)
	plugin.WarningThreshold = 1
	plugin.CriticalThreshold = 8
	status = setStatus(s, numMatches)
	assert.Equal(t, 2, status)
	plugin.WarningOnly = true
	status = setStatus(s, numMatches)
	assert.Equal(t, 1, status)
	plugin.CriticalOnly = true
	plugin.WarningOnly = false
	numMatches = 5
	status = setStatus(s, numMatches)
	assert.Equal(t, 0, status)
}
func TestSettStatusWithCurrentStatusZeroAndInvertThresholds(t *testing.T) {
	clearPlugin()
	s := 0
	numMatches := 10
	status := setStatus(s, numMatches)
	assert.Equal(t, 0, status)
	plugin.InvertThresholds = true
	plugin.WarningThreshold = 20
	status = setStatus(s, numMatches)
	assert.Equal(t, 1, status)
	plugin.WarningThreshold = 5
	status = setStatus(s, numMatches)
	assert.Equal(t, 0, status)
	plugin.WarningThreshold = 20
	plugin.CriticalThreshold = 15
	status = setStatus(s, numMatches)
	assert.Equal(t, 2, status)
	plugin.WarningOnly = true
	status = setStatus(s, numMatches)
	assert.Equal(t, 1, status)
	plugin.CriticalOnly = true
	plugin.WarningOnly = false
	status = setStatus(s, numMatches)
	assert.Equal(t, 2, status)
	numMatches = 30
	status = setStatus(s, numMatches)
	assert.Equal(t, 0, status)
}

func TestSettStatusWithCurrentStatusOne(t *testing.T) {
	clearPlugin()
	s := 1
	numMatches := 10
	status := setStatus(s, numMatches)
	assert.Equal(t, 1, status)
	plugin.WarningThreshold = 1
	status = setStatus(s, numMatches)
	assert.Equal(t, 1, status)
	plugin.WarningThreshold = 11
	status = setStatus(s, numMatches)
	assert.Equal(t, 1, status)
	plugin.WarningThreshold = 1
	plugin.CriticalThreshold = 8
	status = setStatus(s, numMatches)
	assert.Equal(t, 2, status)
	plugin.WarningOnly = true
	status = setStatus(s, numMatches)
	assert.Equal(t, 1, status)
	plugin.CriticalOnly = true
	plugin.WarningOnly = false
	numMatches = 5
	status = setStatus(s, numMatches)
	assert.Equal(t, 1, status)
}
func TestSettStatusWithCurrentStatusTwo(t *testing.T) {
	clearPlugin()
	s := 2
	numMatches := 10
	status := setStatus(s, numMatches)
	assert.Equal(t, 2, status)
	plugin.WarningThreshold = 1
	status = setStatus(s, numMatches)
	assert.Equal(t, 2, status)
	plugin.WarningThreshold = 11
	status = setStatus(s, numMatches)
	assert.Equal(t, 2, status)
	plugin.WarningThreshold = 1
	plugin.CriticalThreshold = 8
	status = setStatus(s, numMatches)
	assert.Equal(t, 2, status)
	plugin.WarningOnly = true
	status = setStatus(s, numMatches)
	assert.Equal(t, 2, status)
	plugin.CriticalOnly = true
	plugin.WarningOnly = false
	numMatches = 5
	status = setStatus(s, numMatches)
	assert.Equal(t, 2, status)
}

func TestSetStatus(t *testing.T) {
	clearPlugin()
	numMatches := 10
	status := setStatus(0, numMatches)
	assert.Equal(t, 0, status)
	plugin.WarningThreshold = 1
	status = setStatus(0, numMatches)
	assert.Equal(t, 1, status)
	plugin.WarningThreshold = 11
	status = setStatus(0, numMatches)
	assert.Equal(t, 0, status)
	plugin.WarningThreshold = 1
	plugin.CriticalThreshold = 8
	status = setStatus(0, numMatches)
	assert.Equal(t, 2, status)
	plugin.WarningOnly = true
	status = setStatus(0, numMatches)
	assert.Equal(t, 1, status)
	plugin.CriticalOnly = true
	plugin.WarningOnly = false
	numMatches = 5
	status = setStatus(0, numMatches)
	assert.Equal(t, 0, status)
}

func TestCheckArgs(t *testing.T) {
	clearPlugin()
	status, err := checkArgs(nil)
	assert.Error(t, err)
	assert.Equal(t, 2, status)
	plugin.WarningThreshold = 1
	plugin.CriticalThreshold = 2
	event := corev2.FixtureEvent("foo", "bar")
	status, err = checkArgs(event)
	assert.Error(t, err)
	assert.Equal(t, 2, status)
	plugin.MatchExpr = "test"
	status, err = checkArgs(event)
	assert.Error(t, err)
	assert.Equal(t, 2, status)
	plugin.LogFile = "test.log"
	status, err = checkArgs(event)
	assert.Error(t, err)
	assert.Equal(t, 2, status)
	plugin.LogFileExpr = "logexpr"
	status, err = checkArgs(event)
	assert.Error(t, err)
	assert.Equal(t, 2, status)
	plugin.LogPath = "/var/log"
	status, err = checkArgs(event)
	assert.Error(t, err)
	assert.Equal(t, 2, status)
	plugin.StateDir = "missing_dir"
	status, err = checkArgs(event)
	assert.NoError(t, err)
	assert.Equal(t, 0, status)
	td, err := os.MkdirTemp("", "")
	assert.NoError(t, err)
	defer os.RemoveAll(td)
	plugin.StateDir = td
	status, err = checkArgs(event)
	assert.NoError(t, err)
	assert.Equal(t, 0, status)
	defer os.RemoveAll(td)
	plugin.MatchExpr = "error"
	status, err = checkArgs(event)
	assert.NoError(t, err)
	assert.Equal(t, 0, status)
	status, err = checkArgs(event)
	assert.NoError(t, err)
	assert.Equal(t, 0, status)
	plugin.DryRun = true
	status, err = checkArgs(event)
	assert.NoError(t, err)
	assert.Equal(t, 0, status)

	clearPlugin()
}

func TestState(t *testing.T) {
	td, err := os.MkdirTemp("", "")
	defer os.RemoveAll(td)
	if err != nil {
		t.Fatal(err)
	}
	stateFile := filepath.Join(td, "state")
	state, err := getState(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := state.Offset, int64(0); got != want {
		t.Errorf("bad offset: got %v, want %v", got, want)
	}
	state.Offset = int64(0xBEEF)
	if err := setState(state, stateFile); err != nil {
		t.Fatal(err)
	}

	state, err = getState(stateFile)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := state.Offset, int64(0xBEEF); got != want {
		t.Errorf("bad offset: got %v, want %v", got, want)
	}
}

func TestBuildLogArray(t *testing.T) {
	logs, err := buildLogArray()
	assert.Equal(t, 0, len(logs))
	assert.NoError(t, err)
	err = os.Chmod("./testingdata/test.log", 0755)
	assert.NoError(t, err)

	plugin.LogFile = "./testingdata/test.log"
	plugin.LogPath = ""
	plugin.LogFileExpr = ""
	plugin.Verbose = false
	logs, err = buildLogArray()
	if err != nil {
		t.Errorf("BuildLogArray err: %s", err)
	}
	if len(logs) != 1 {
		t.Errorf("BuildLogArray len %v", len(logs))
	}
	for _, log := range logs {
		if runtime.GOOS != "windows" {
			assert.Contains(t, log, `/testingdata/test.log`)
		} else {
			assert.Contains(t, log, `\testingdata\test.log`)
		}
	}
	plugin.LogFile = ""
	plugin.LogPath = "testingdata/"
	plugin.LogFileExpr = "test.log"
	plugin.Verbose = false
	logs, err = buildLogArray()
	if err != nil {
		t.Errorf("BuildLogArray err: %s", err)
	}
	if len(logs) != 1 {
		t.Errorf("BuildLogArray len %v", len(logs))
	}
	for _, log := range logs {
		if runtime.GOOS != "windows" {
			assert.Contains(t, log, `/testingdata/test.log`)
		} else {
			assert.Contains(t, log, `\testingdata\test.log`)
		}
	}
	plugin.LogFile = ""
	plugin.LogPath = `testingdata/`
	plugin.LogFileExpr = `webserver`
	plugin.Verbose = false
	logs, err = buildLogArray()
	if err != nil {
		t.Errorf("BuildLogArray err: %s", err)
	}
	if len(logs) != 3 {
		t.Errorf("BuildLogArray len %v", len(logs))
	}
	for _, log := range logs {
		assert.Contains(t, log, "access.log")
	}

}

func TestExecuteCheckWithDisableEvent(t *testing.T) {
	plugin.Verbose = true
	plugin.Procs = 1
	plugin.DisableEvent = true
	plugin.LogFile = "./testingdata/test.log"
	plugin.MatchExpr = "test"
	plugin.WarningThreshold = 1
	plugin.WarningOnly = true
	td, err := os.MkdirTemp("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	status, err := executeCheck(nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, status)
	plugin.CriticalThreshold = 1
	plugin.CriticalOnly = true
	ctd, err := os.MkdirTemp("", "")
	defer os.RemoveAll(ctd)
	assert.NoError(t, err)
	plugin.StateDir = ctd
	status, err = executeCheck(nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, status)
}
func TestExecuteCheckWithNoEvent(t *testing.T) {
	plugin.Verbose = true
	plugin.Procs = 1
	plugin.DisableEvent = false
	plugin.LogFile = "./testingdata/test.log"
	plugin.MatchExpr = "test"
	td, err := os.MkdirTemp("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	status, err := executeCheck(nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, status)
}

func TestExecuteCheckWithNoEventAndFileError(t *testing.T) {
	plugin.Verbose = true
	plugin.Procs = 1
	plugin.DisableEvent = false
	plugin.LogFile = "./testingdata/test.log"
	plugin.MatchExpr = "test"
	td, err := os.MkdirTemp("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	if runtime.GOOS != "windows" {
		err = os.Chmod("./testingdata/test.log", 0000)
		assert.NoError(t, err)
		status, err := executeCheck(nil)
		assert.NoError(t, err)
		assert.Equal(t, 2, status)
		err = os.Chmod("./testingdata/test.log", 0755)
		assert.NoError(t, err)
	}
}

func TestExecuteWithEvent(t *testing.T) {
	event := corev2.FixtureEvent("foo", "bar")
	server := httptest.NewServer(apiHandler{t: t, event: event})
	defer server.Close()

	plugin.Verbose = true
	plugin.Procs = 1
	plugin.DisableEvent = false
	plugin.LogFile = "./testingdata/test.log"
	plugin.MatchExpr = "test"

	// no events api defined error
	td, err := os.MkdirTemp("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	status, err := executeCheck(event)
	assert.NoError(t, err)
	assert.Equal(t, 1, status)

	//no name template error
	td, err = os.MkdirTemp("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	plugin.EventsAPI = server.URL + "/events"
	plugin.CheckNameTemplate = ""
	status, err = executeCheck(event)
	assert.NoError(t, err)
	assert.Equal(t, 0, status)

	// 404 events api status error
	td, err = os.MkdirTemp("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	plugin.EventsAPI = server.URL + "/bad"
	plugin.CheckNameTemplate = "test-check"
	status, err = executeCheck(event)
	assert.NoError(t, err)
	assert.Equal(t, 1, status)
	td, err = os.MkdirTemp("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	plugin.EventsAPI = server.URL + "/events"
	plugin.CheckNameTemplate = "test-check"
	status, err = executeCheck(event)
	assert.NoError(t, err)
	assert.Equal(t, 0, status)
	plugin.DryRun = true
	td, err = os.MkdirTemp("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	plugin.EventsAPI = server.URL + "/events"
	plugin.CheckNameTemplate = "test-check"
	status, err = executeCheck(event)
	assert.NoError(t, err)
	assert.Equal(t, 0, status)
	plugin.DryRun = false

}

func TestProcessLogFile(t *testing.T) {
	plugin.Verbose = true
	plugin.MaxBytes = 4000
	plugin.Procs = 1
	plugin.DisableEvent = true
	plugin.LogFile = "./testingdata/test.log"
	plugin.MatchExpr = "test"

	td, err := os.MkdirTemp("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	err = os.Chmod("./testingdata/test.log", 0755)
	assert.NoError(t, err)
	eventBuf := new(bytes.Buffer)
	enc := json.NewEncoder(eventBuf)

	// test for good match
	td, err = os.MkdirTemp("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	plugin.MatchExpr = "test"
	plugin.WarningOnly = true
	logs, err := buildLogArray()
	assert.NoError(t, err)
	matches, err := processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 1, matches)

	// test for abs log file path err
	td, err = os.MkdirTemp("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	matches, err = processLogFile(plugin.LogFile, enc)
	assert.Error(t, err)
	assert.Equal(t, 0, matches)
	logs, err = buildLogArray()
	assert.NoError(t, err)
	matches, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 1, matches)

	// test for IgnoreFirstRun
	plugin.IgnoreInitialRun = true
	td, err = os.MkdirTemp("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	logs, err = buildLogArray()
	assert.NoError(t, err)
	matches, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 0, matches)
	plugin.IgnoreInitialRun = false
	td, err = os.MkdirTemp("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	logs, err = buildLogArray()
	assert.NoError(t, err)
	matches, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 1, matches)

	// test for state mismatch error
	plugin.MatchExpr = "hmm"
	matches, err = processLogFile(logs[0], enc)
	assert.Error(t, err)
	assert.Equal(t, 0, matches)
	plugin.EnableStateReset = true
	matches, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 0, matches)

	// Do not run error condition tests that require chmod on windows, they will fail
	if runtime.GOOS != "windows" {
		// test for read log file error
		err = os.Chmod("./testingdata/test.log", 0000)
		assert.NoError(t, err)
		matches, err = processLogFile(logs[0], enc)
		assert.Error(t, err)
		assert.Equal(t, 0, matches)
		err = os.Chmod("./testingdata/test.log", 0755)
		assert.NoError(t, err)

		// test for state file read error
		err = os.Chmod(td, 0000)
		assert.NoError(t, err)
		matches, err = processLogFile(logs[0], enc)
		assert.Error(t, err)
		assert.Equal(t, 0, matches)
		err = os.Chmod(td, 0755)
		assert.NoError(t, err)

		// test for state file write error
		td, err = os.MkdirTemp("", "")
		defer os.RemoveAll(td)
		assert.NoError(t, err)
		plugin.StateDir = td
		err = os.Chmod(td, 0500)
		assert.NoError(t, err)
		matches, err = processLogFile(logs[0], enc)
		assert.Error(t, err)
		assert.Equal(t, 0, matches)
		err = os.Chmod(td, 0755)
		assert.NoError(t, err)
	}
}

func TestProcessLogFileRotatedFileVerboseTrue(t *testing.T) {
	clearPlugin()
	plugin.Verbose = true
	plugin.Procs = 1
	plugin.DisableEvent = true
	plugin.MatchExpr = "brown"

	td, err := os.MkdirTemp("", "")
	assert.NoError(t, err)
	defer os.RemoveAll(td)
	plugin.StateDir = td

	logdir, err := os.MkdirTemp("", "")
	assert.NoError(t, err)
	defer os.RemoveAll(logdir)

	plugin.LogFile = logdir + "/test.log"
	f, err := os.OpenFile(plugin.LogFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	assert.NoError(t, err)
	_, err = f.WriteString("what now brown cow\n")
	assert.NoError(t, err)
	f.Close()
	_, err = os.ReadFile(plugin.LogFile)
	assert.NoError(t, err)

	eventBuf := new(bytes.Buffer)
	enc := json.NewEncoder(eventBuf)

	logs, err := buildLogArray()
	assert.NoError(t, err)
	matches, err := processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 1, matches)

	//re-run should have no new matches
	matches, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 0, matches)

	//rotate the file
	os.Remove(plugin.LogFile)
	f, err = os.OpenFile(plugin.LogFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	assert.NoError(t, err)
	_, err = f.WriteString("the brown cow\n")
	assert.NoError(t, err)
	f.Close()
	_, err = os.ReadFile(plugin.LogFile)
	assert.NoError(t, err)
	logs, err = buildLogArray()
	assert.NoError(t, err)
	matches, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 1, matches)

	//append file and test offset seeking
	f, err = os.OpenFile(plugin.LogFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	assert.NoError(t, err)
	_, err = f.WriteString("brown cows yeah!\n")
	assert.NoError(t, err)
	f.Close()
	_, err = os.ReadFile(plugin.LogFile)
	assert.NoError(t, err)
	matches, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 1, matches)

}

func TestProcessLogFileRotatedFileVerboseFalse(t *testing.T) {
	clearPlugin()
	plugin.Verbose = false
	plugin.Procs = 1
	plugin.DisableEvent = true
	plugin.MatchExpr = "brown"

	td, err := os.MkdirTemp("", "")
	assert.NoError(t, err)
	defer os.RemoveAll(td)
	plugin.StateDir = td

	logdir, err := os.MkdirTemp("", "")
	assert.NoError(t, err)
	defer os.RemoveAll(logdir)

	plugin.LogFile = path.Join(logdir, "test.log")
	f, err := os.OpenFile(plugin.LogFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	assert.NoError(t, err)
	_, err = f.WriteString("what now brown cow\n")
	assert.NoError(t, err)
	f.Close()
	_, err = os.ReadFile(plugin.LogFile)
	assert.NoError(t, err)

	eventBuf := new(bytes.Buffer)
	enc := json.NewEncoder(eventBuf)

	logs, err := buildLogArray()
	assert.NoError(t, err)
	matches, err := processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 1, matches)

	//re-run should have no new matches
	matches, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 0, matches)

	//rotate the file
	os.Remove(plugin.LogFile)
	f, err = os.OpenFile(plugin.LogFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	assert.NoError(t, err)
	_, err = f.WriteString("the brown cow\n")
	assert.NoError(t, err)
	f.Close()
	_, err = os.ReadFile(plugin.LogFile)
	assert.NoError(t, err)
	logs, err = buildLogArray()
	assert.NoError(t, err)
	matches, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 1, matches)

	//append file and test offset seeking
	f, err = os.OpenFile(plugin.LogFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	assert.NoError(t, err)
	_, err = f.WriteString("brown cows yeah!\n")
	assert.NoError(t, err)
	f.Close()
	_, err = os.ReadFile(plugin.LogFile)
	assert.NoError(t, err)
	matches, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 1, matches)

}

func TestProcessLogFileWithNegativeCachedOffset(t *testing.T) {

	if runtime.GOOS != "windows" {
		clearPlugin()
		plugin.Verbose = false
		plugin.Procs = 1
		plugin.DisableEvent = true
		plugin.MatchExpr = "brown"

		td, err := os.MkdirTemp("", "")
		assert.NoError(t, err)
		defer os.RemoveAll(td)
		plugin.StateDir = td

		logdir, err := os.MkdirTemp("", "")
		assert.NoError(t, err)
		defer os.RemoveAll(logdir)

		plugin.LogFile = path.Join(logdir, "test.log")
		f, err := os.OpenFile(plugin.LogFile,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		assert.NoError(t, err)
		_, err = f.WriteString("what now brown cow\n")
		assert.NoError(t, err)
		f.Close()

		stateFile := filepath.Join(plugin.StateDir, strings.ReplaceAll(plugin.LogFile, string(os.PathSeparator), string("_")))
		state, err := getState(stateFile)
		assert.NoError(t, err)
		state.Offset = -10
		err = setState(state, stateFile)
		assert.NoError(t, err)

		eventBuf := new(bytes.Buffer)
		enc := json.NewEncoder(eventBuf)

		logs, err := buildLogArray()
		assert.NoError(t, err)
		matches, err := processLogFile(logs[0], enc)
		assert.Error(t, err)
		assert.Equal(t, 0, matches)
	}
}
