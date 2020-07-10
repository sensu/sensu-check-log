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
		_ = os.RemoveAll(td)
	}()
	stateFile := filepath.Join(td, "state")
	state, err := getState(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(state.Offset), ""; got != want {
		t.Errorf("bad offset: got %s, want %s", got, want)
	}
	state.Offset = json.Number(fmt.Sprintf("%d", 0xBEEF))
	if err := setState(state, stateFile); err != nil {
		t.Fatal(err)
	}

	state, err = getState(stateFile)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(state.Offset), fmt.Sprintf("%d", 0xBEEF); got != want {
		t.Errorf("bad offset: got %s, want %s", got, want)
	}
}
