package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	sensu "github.com/sensu/sensu-go/api/core/v2"
)

type testHandler struct {
	t     *testing.T
	event *sensu.Event
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

	if err := sendEvent(server.URL+"/events", event, 1, "test_check", "output"); err != nil {
		t.Fatal(err)
	}
}
