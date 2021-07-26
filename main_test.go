package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
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
	if err != nil {
		t.Errorf("BuildLogArray err: %s", err)
	}
	plugin.LogFile = "./testingdata/test.log"
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
