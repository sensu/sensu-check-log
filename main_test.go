package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestState(t *testing.T) {
	td, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(td)
	}()
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
	plugin.Verbose = true
	err = buildLogArray()
	if err != nil {
		t.Errorf("BuildLogArray err: %s", err)
	}
	if len(logs) != 1 {
		t.Errorf("BuildLogArray len %v", len(logs))
	}

}

func TestProcessLogFile(t *testing.T) {
	logs = []string{}
	plugin.LogFile = "./testingdata/test.log"
	plugin.MatchExpr = "test"
	td, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	plugin.StateDir = td
	err = os.Chmod("./testingdata/test.log", 0755)
	assert.NoError(t, err)
	eventBuf := new(bytes.Buffer)
	enc := json.NewEncoder(eventBuf)

	// test for abs log file path err
	status, err := processLogFile(plugin.LogFile, enc)
	assert.Error(t, err)
	assert.Equal(t, status, 2)
	err = buildLogArray()
	assert.NoError(t, err)
	status, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, status, 0)

	// test for read log file error
	err = os.Chmod("./testingdata/test.log", 0000)
	assert.NoError(t, err)
	status, err = processLogFile(logs[0], enc)
	assert.Error(t, err)
	assert.Equal(t, status, 2)
	err = os.Chmod("./testingdata/test.log", 0755)
	assert.NoError(t, err)

	// test for state file read error
	err = os.Chmod(td, 0000)
	assert.NoError(t, err)
	status, err = processLogFile(logs[0], enc)
	assert.Error(t, err)
	assert.Equal(t, status, 2)
	err = os.Chmod(td, 0755)
	assert.NoError(t, err)

	// test for state mismatch error
	plugin.MatchExpr = "hmm"
	status, err = processLogFile(logs[0], enc)
	assert.Error(t, err)
	assert.Equal(t, status, 2)
	plugin.EnableStateReset = true
	status, err = processLogFile(logs[0], enc)
	assert.NoError(t, err)
	assert.Equal(t, status, 0)

	// test for state file write error
	td, err = ioutil.TempDir("", "")
	assert.NoError(t, err)
	plugin.StateDir = td
	err = os.Chmod(td, 0500)
	assert.NoError(t, err)
	status, err = processLogFile(logs[0], enc)
	assert.Error(t, err)
	assert.Equal(t, status, 2)
	err = os.Chmod(td, 0755)
	assert.NoError(t, err)

}
