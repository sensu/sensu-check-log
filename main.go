package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"

	sensu "github.com/sensu/sensu-go/api/core/v2"
)

var (
	logFile     = flag.String("log", "", "path to log file")
	procs       = flag.Int("procs", runtime.NumCPU(), "number of parallel analyzer processes")
	match       = flag.String("match", "", "RE2 regexp matcher expression")
	stateFile   = flag.String("state", "", "state file for incremental log analysis (required)")
	eventStatus = flag.Int("event-status", 1, "event status on positive match")
	eventsAPI   = flag.String("api-url", "http://localhost:3031", "agent events API URL")
)

const (
	StatusOK   = 0
	StatusWarn = 1
	StatusCrit = 2
)

type State struct {
	Offset json.Number `json:"offset"`
	Status int         `json:"status"`
}

func getState(path string) (state State, err error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, fmt.Errorf("couldn't read state file: %s", err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&state); err != nil {
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
	if err := json.NewEncoder(f).Encode(cur); err != nil {
		return fmt.Errorf("couldn't write state file: %s", err)
	}
	return nil
}

func fatal(formatter string, args ...interface{}) {
	log.Printf(formatter, args...)
	os.Exit(2)
}

func testFlags() {
	flag.Parse()
	if *logFile == "" {
		fatal("-log not specified")
	}
	if *stateFile == "" {
		fatal("-state not specified")
	}
	if *match == "" {
		fatal("-match not specified")
	}
}

func main() {
	testFlags()

	timer := time.NewTimer(5 * time.Second)
	go func() {
		if _, ok := <-timer.C; ok {
			fatal("failed to read from stdin. (did you set 'stdin: true' on check config?)")
		}
	}()

	var inputEvent sensu.Event
	if err := json.NewDecoder(os.Stdin).Decode(&inputEvent); err != nil {
		fatal("error decoding input event: %s", err)
	}

	timer.Stop()

	f, err := os.Open(*logFile)
	if err != nil {
		fatal("couldn't open log file: %s", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			fatal("error closing log file: %s", err)
		}
	}()

	state, err := getState(*stateFile)
	if err != nil {
		fatal("%s", err)
	}
	offset, _ := state.Offset.Int64()
	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			fatal("couldn't seek to offset %d: %s", offset, err)
		}
	}

	analyzer := Analyzer{
		Procs: *procs,
		Log:   f,
		Func:  AnalyzeRegexp(*match),
	}

	results := analyzer.Go(context.Background())
	eventBuf := new(bytes.Buffer)
	enc := json.NewEncoder(eventBuf)

	status := state.Status

	for result := range results {
		if result.Err != nil {
			status = StatusCrit
		}
		if err := enc.Encode(result); err != nil {
			fatal("%s", err)
		}
		status = *eventStatus
	}

	if status != state.Status {
		if err := sendEvent(*eventsAPI, &inputEvent, status, eventBuf.String()); err != nil {
			fatal("error sending event: %s", err)
		}
	}

	bytesRead := analyzer.BytesRead()
	state.Offset = json.Number(fmt.Sprintf("%d", offset+bytesRead))

	if err := setState(state, *stateFile); err != nil {
		fatal("%s", err)
	}
}
