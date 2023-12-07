package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	corev2 "github.com/sensu/core/v2"
)

func sendEvent(path string, outputEvent *corev2.Event) error {

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
