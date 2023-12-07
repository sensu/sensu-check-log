package main

import (
	"errors"
	"time"

	corev2 "github.com/sensu/core/v2"
	"github.com/sensu/sensu-plugin-sdk/templates"
)

func createEvent(inputEvent *corev2.Event, status int, checkNameTemplate string, results string) (*corev2.Event, error) {
	if status < 0 {
		return nil, errors.New("negative status")
	}
	// Let's construct the check name from template
	checkName, err := templates.EvalTemplate("check-name", checkNameTemplate, inputEvent)
	if err != nil {
		return nil, err
	}
	outputEvent := corev2.Event{Entity: inputEvent.Entity}
	outputEvent.Namespace = inputEvent.Namespace
	check := inputEvent.Check
	outputEvent.Check = check
	check.Executed = time.Now().Unix()
	check.Issued = inputEvent.Check.Issued
	check.Command = inputEvent.Check.Command
	check.Name = checkName
	check.Output = results
	check.Status = uint32(status)
	return &outputEvent, nil
}
