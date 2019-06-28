package main

import (
	"encoding/json"
	"fmt"
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
		os.RemoveAll(td)
	}()
	stateFile := filepath.Join(td, "state")
	state, err := getState(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := state.Status, 0; got != want {
		t.Errorf("bad status: got %d, want %d", got, want)
	}
	if got, want := string(state.Offset), ""; got != want {
		t.Errorf("bad offset: got %s, want %s", got, want)
	}
	state.Status = 1
	state.Offset = json.Number(fmt.Sprintf("%d", 0xBEEF))
	if err := setState(state, stateFile); err != nil {
		t.Fatal(err)
	}

	state, err = getState(stateFile)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := state.Status, 1; got != want {
		t.Errorf("bad status: got %d, want %d", got, want)
	}
	if got, want := string(state.Offset), fmt.Sprintf("%d", 0xBEEF); got != want {
		t.Errorf("bad offset: got %s, want %s", got, want)
	}
}
