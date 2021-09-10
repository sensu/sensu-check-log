package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	corev2 "github.com/sensu/sensu-go/api/core/v2"
	sensu "github.com/sensu/sensu-go/api/core/v2"
	"github.com/stretchr/testify/assert"
)

type apiHandler struct {
	t     *testing.T
	event *sensu.Event
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
	plugin.MatchStatus = 0
	plugin.IgnoreInitialRun = false
	plugin.InverseMatch = false
}

func TestStdin(t *testing.T) {
	clearPlugin()
	test, err := testStdin()
	assert.NoError(t, err)
	assert.Equal(t, false, test)
}

func TestCheckArgs(t *testing.T) {
	clearPlugin()
	status, err := checkArgs(nil)
	assert.Error(t, err)
	assert.Equal(t, 2, status)
	event := corev2.FixtureEvent("foo", "bar")
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
	td, err := ioutil.TempDir("", "")
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
	td, err := ioutil.TempDir("", "")
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
	err := buildLogArray()
	assert.NoError(t, err)
	plugin.LogFile = "./testingdata/test.log"
	err = os.Chmod("./testingdata/test.log", 0755)
	assert.NoError(t, err)
	plugin.LogPath = "testingdata/"
	plugin.LogFileExpr = "test.log"
	plugin.Verbose = false
	err = buildLogArray()
	if err != nil {
		t.Errorf("BuildLogArray err: %s", err)
	}
	if len(logs) != 1 {
		t.Errorf("BuildLogArray len %v", len(logs))
	}

}

func TestExecuteCheckWithDisableEvent(t *testing.T) {
	plugin.Verbose = true
	plugin.Procs = 1
	plugin.DisableEvent = true
	plugin.LogFile = "./testingdata/test.log"
	plugin.MatchExpr = "test"
	plugin.MatchStatus = 40
	td, err := ioutil.TempDir("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	status, err := executeCheck(nil)
	assert.NoError(t, err)
	assert.Equal(t, 40, status)
}
func TestExecuteCheckWithNoEvent(t *testing.T) {
	plugin.Verbose = true
	plugin.Procs = 1
	plugin.DisableEvent = false
	plugin.LogFile = "./testingdata/test.log"
	plugin.MatchExpr = "test"
	plugin.MatchStatus = 40
	td, err := ioutil.TempDir("", "")
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
	plugin.MatchStatus = 40
	td, err := ioutil.TempDir("", "")
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
	plugin.MatchStatus = 40

	// no events api defined error
	td, err := ioutil.TempDir("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	status, err := executeCheck(event)
	assert.NoError(t, err)
	assert.Equal(t, 1, status)

	// no name template error
	td, err = ioutil.TempDir("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	plugin.EventsAPI = server.URL + "/events"
	plugin.CheckNameTemplate = ""
	status, err = executeCheck(event)
	assert.NoError(t, err)
	assert.Equal(t, 1, status)

	// 404 events api status error
	td, err = ioutil.TempDir("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	plugin.EventsAPI = server.URL + "/bad"
	plugin.CheckNameTemplate = "test-check"
	status, err = executeCheck(event)
	assert.NoError(t, err)
	assert.Equal(t, 1, status)
	td, err = ioutil.TempDir("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	plugin.EventsAPI = server.URL + "/events"
	plugin.CheckNameTemplate = "test-check"
	status, err = executeCheck(event)
	assert.NoError(t, err)
	assert.Equal(t, 0, status)
	plugin.DryRun = true
	td, err = ioutil.TempDir("", "")
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
	plugin.MatchStatus = 40
	logs = []string{}
	plugin.LogFile = "./testingdata/test.log"
	plugin.MatchExpr = "test"

	td, err := ioutil.TempDir("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	err = os.Chmod("./testingdata/test.log", 0755)
	assert.NoError(t, err)
	eventBuf := new(bytes.Buffer)
	enc := json.NewEncoder(eventBuf)

	// test for good match
	logs = []string{}
	td, err = ioutil.TempDir("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	plugin.MatchExpr = "test"
	err = buildLogArray()
	assert.NoError(t, err)
	status, err := processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 40, status)

	// test for abs log file path err
	logs = []string{}
	td, err = ioutil.TempDir("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	status, err = processLogFile(plugin.LogFile, enc)
	assert.Error(t, err)
	assert.Equal(t, 2, status)
	err = buildLogArray()
	assert.NoError(t, err)
	status, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 40, status)

	// test for IgnoreFirstRun
	plugin.IgnoreInitialRun = true
	logs = []string{}
	td, err = ioutil.TempDir("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	err = buildLogArray()
	assert.NoError(t, err)
	status, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 0, status)
	plugin.IgnoreInitialRun = false
	td, err = ioutil.TempDir("", "")
	defer os.RemoveAll(td)
	assert.NoError(t, err)
	plugin.StateDir = td
	err = buildLogArray()
	assert.NoError(t, err)
	status, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 40, status)

	// test for state mismatch error
	plugin.MatchExpr = "hmm"
	status, err = processLogFile(logs[0], enc)
	assert.Error(t, err)
	assert.Equal(t, 2, status)
	plugin.EnableStateReset = true
	status, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 0, status)
	plugin.InverseMatch = true
	plugin.EnableStateReset = false
	status, err = processLogFile(logs[0], enc)
	assert.Error(t, err)
	assert.Equal(t, 2, status)
	plugin.EnableStateReset = true
	status, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 40, status)

	// Do not run error condition tests that require chmod on windows, they will fail
	if runtime.GOOS != "windows" {
		// test for read log file error
		err = os.Chmod("./testingdata/test.log", 0000)
		assert.NoError(t, err)
		status, err = processLogFile(logs[0], enc)
		assert.Error(t, err)
		assert.Equal(t, 2, status)
		err = os.Chmod("./testingdata/test.log", 0755)
		assert.NoError(t, err)

		// test for state file read error
		err = os.Chmod(td, 0000)
		assert.NoError(t, err)
		status, err = processLogFile(logs[0], enc)
		assert.Error(t, err)
		assert.Equal(t, 2, status)
		err = os.Chmod(td, 0755)
		assert.NoError(t, err)

		// test for state file write error
		td, err = ioutil.TempDir("", "")
		defer os.RemoveAll(td)
		assert.NoError(t, err)
		plugin.StateDir = td
		err = os.Chmod(td, 0500)
		assert.NoError(t, err)
		status, err = processLogFile(logs[0], enc)
		assert.Error(t, err)
		assert.Equal(t, 2, status)
		err = os.Chmod(td, 0755)
		assert.NoError(t, err)
	}
}

func TestProcessLogFileRotatedFile(t *testing.T) {
	clearPlugin()
	plugin.Verbose = true
	plugin.Procs = 1
	plugin.DisableEvent = true
	plugin.MatchStatus = 40
	plugin.MatchExpr = "brown"
	logs = []string{}

	td, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	defer os.RemoveAll(td)
	plugin.StateDir = td

	logdir, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	defer os.RemoveAll(logdir)

	plugin.LogFile = logdir + "/test.log"
	f, err := os.OpenFile(plugin.LogFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	assert.NoError(t, err)
	_, err = f.WriteString("what now brown cow\n")
	assert.NoError(t, err)
	f.Close()
	_, err = ioutil.ReadFile(plugin.LogFile)
	assert.NoError(t, err)

	eventBuf := new(bytes.Buffer)
	enc := json.NewEncoder(eventBuf)

	err = buildLogArray()
	assert.NoError(t, err)
	status, err := processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 40, status)

	//rotate the file
	os.Remove(plugin.LogFile)
	f, err = os.OpenFile(plugin.LogFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	assert.NoError(t, err)
	_, err = f.WriteString("the brown cow\n")
	assert.NoError(t, err)
	f.Close()
	_, err = ioutil.ReadFile(plugin.LogFile)
	assert.NoError(t, err)
	err = buildLogArray()
	assert.NoError(t, err)
	status, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 40, status)

	//append file and test offset seeking
	f, err = os.OpenFile(plugin.LogFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	assert.NoError(t, err)
	_, err = f.WriteString("brown cows yeah!\n")
	assert.NoError(t, err)
	f.Close()
	_, err = ioutil.ReadFile(plugin.LogFile)
	assert.NoError(t, err)
	status, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, 40, status)

}
