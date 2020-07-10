package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	sensu "github.com/sensu/sensu-go/api/core/v2"
)

func sendEvent(path string, inputEvent *sensu.Event, status int, results string) error {
	if status < 0 {
		return errors.New("negative status")
	}
	outputEvent := sensu.Event{Entity: inputEvent.Entity}
	outputEvent.Namespace = inputEvent.Namespace
	check := inputEvent.Check
	outputEvent.Check = check
	check.Executed = time.Now().Unix()
	check.Issued = inputEvent.Check.Issued
	check.Command = inputEvent.Check.Command
	check.Name = fmt.Sprintf("%s-failure", check.Name)
	check.Output = results
	check.Status = uint32(status)

	b, err := json.Marshal(outputEvent)
	if err != nil {
		return fmt.Errorf("error writing event: %s", err)
	}

	resp, err := http.Post(path, "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("error writing event: %s", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if status := resp.StatusCode; status >= 400 {
		b, _ := ioutil.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("error writing event: status %d: %s", status, string(b))
	}

	return nil
}
