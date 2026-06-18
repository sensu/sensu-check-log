package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	sensu "github.com/sensu/core/v2"
)

type testHandler struct {
	t     *testing.T
	event *sensu.Event
}

func TestCreateEvent_TemplateEvaluation(t *testing.T) {
	testCases := []struct {
		name              string
		event             *sensu.Event // Setup sensu event data here
		checkNameTemplate string
		expectedCheckName string
		expectedError     bool
	}{
		{
			name:              "Simple Template",
			event:             sensu.FixtureEvent("entity1", "check1"),
			checkNameTemplate: "new-{{ .Check.Name }}",
			expectedCheckName: "new-check1",
			expectedError:     false,
		},
		{
			name:              "Invalid Template",
			event:             sensu.FixtureEvent("entity1", "check1"),
			checkNameTemplate: "{{ .Foo }}", // Assuming 'Foo' isn't a valid field
			expectedCheckName: "",           // Template evaluation should fail
			expectedError:     true,
		},
		// ... add more test cases with different scenarios
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			outputEvent, err := createEvent(tc.event, 1, tc.checkNameTemplate, "") // Status 0 here as it doesn't impact this test

			if tc.expectedError {
				if err == nil {
					t.Error("Expected an error during template evaluation, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error during template evaluation: %v", err)
				}
				if outputEvent.Check.Name != tc.expectedCheckName {
					t.Errorf("Incorrect check name. Expected: %s, Got: %s", tc.expectedCheckName, outputEvent.Check.Name)
				}
			}
		})
	}
}

func (h testHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	if !strings.HasSuffix(req.URL.Path, "/events") {
		http.Error(w, "not found", 404)
		return
	}
	var event sensu.Event
	if err := json.NewDecoder(req.Body).Decode(&event); err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	if got, want := event.Check.Status, uint32(1); got != want {
		h.t.Errorf("bad check status: got %d, want %d", got, want)
	}
	if got, want := event.Check.Output, "output"; got != want {
		h.t.Errorf("bad check output: got %q, want %q", got, want)
	}
	if got, want := event.Check.Name, "bar-failure"; got != want {
		h.t.Errorf("bad check name: got %q, want %q", got, want)
	}
	if got, want := event.Check.Issued, h.event.Check.Issued; got != want {
		h.t.Errorf("bad check issued: got %d, want %d", got, want)
	}
	if got, want := event.Check.Command, h.event.Check.Command; got != want {
		h.t.Errorf("bad check command: got %q, want %q", got, want)
	}
	if got, want := event.Entity, h.event.Entity; !reflect.DeepEqual(got, want) {
		h.t.Errorf("bad entity: got %v, want %v", got, want)
	}
}

func TestSendEvent(t *testing.T) {

	event := sensu.FixtureEvent("foo", "bar")
	server := httptest.NewServer(testHandler{t: t, event: event})
	defer server.Close()
	outputEvent, err := createEvent(event, 1, "{{ .Check.Name }}-failure", "output")
	if err != nil {
		t.Fatal(err)
	}
	if err := sendEvent(server.URL+"/events", outputEvent); err != nil {
		t.Fatal(err)
	}
}
