package sandbox_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client/sandbox"
)

func eventIDs(evs []sandbox.Event) string {
	var ids []string
	for _, ev := range evs {
		ids = append(ids, ev.ID+":"+ev.Type)
	}
	return strings.Join(ids, " ")
}

func TestEventsLongPoll(t *testing.T) {
	f, sbx := newFake(t)
	a, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine", Labels: map[string]string{"task": "t-81"},
		Retain: &sandbox.Retain{Paths: []string{"/workspace/output.json"}}})
	b, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine", Labels: map[string]string{"task": "t-82"}})
	_ = a.WriteFile(ctx, "/workspace/output.json", []byte("{}"))
	ex, _ := a.ExecDetach(ctx, sandbox.ExecRequest{Command: []string{"seq", "2"}, IdempotencyKey: "t-81:1"})
	if _, err := ex.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	f.exitMain(a.ID, 0)
	_ = b.Terminate(ctx, sandbox.TerminateOptions{})

	all, next, err := sbx.Events(ctx, sandbox.EventQuery{After: sandbox.FromOldest})
	if err != nil || eventIDs(all) != "1:sandbox.running 2:sandbox.running 3:exec.exited 4:sandbox.exited 5:sandbox.exited" || next != "5" {
		t.Fatalf("%s next %s %v", eventIDs(all), next, err)
	}
	running, exec, exited := all[0], all[2], all[3]
	if running.Image == nil || running.Image.Ref != "alpine" || running.ReadyAt == nil || running.Labels["task"] != "t-81" || running.Bucket != 4116 {
		t.Errorf("sandbox.running %+v", running)
	}
	if exec.ExecID != ex.ExecID || exec.ExitCode == nil || *exec.ExitCode != 0 || exec.IdempotencyKey != "t-81:1" || exec.StdoutBytes != 4 {
		t.Errorf("exec.exited %+v", exec)
	}
	if exited.SandboxID != a.ID || exited.ExitKind != sandbox.ExitKindMainExited || exited.Status != sandbox.StatusTerminated ||
		len(exited.Retained) != 1 || exited.Retained[0].Path != "/workspace/output.json" || exited.Usage == nil || exited.Usage.CPUAvg != 0.04 {
		t.Errorf("sandbox.exited %+v", exited)
	}

	// A label and a type: only that sandbox's end.
	evs, next, err := sbx.Events(ctx, sandbox.EventQuery{After: sandbox.FromOldest, Types: []string{sandbox.EventSandboxExited},
		Labels: map[string]string{"task": "t-81"}, Limit: 10, WaitS: 25})
	if err != nil || eventIDs(evs) != "4:sandbox.exited" || next != "4" {
		t.Fatalf("%s next %s %v", eventIDs(evs), next, err)
	}
	if got := f.polls[len(f.polls)-1]; got != "after=0&label=task%3Dt-81&limit=10&types=sandbox.exited&wait_s=25" {
		t.Errorf("query %s", got)
	}
	// Nothing new: the cursor stays.
	evs, next, err = sbx.Events(ctx, sandbox.EventQuery{After: "5"})
	if err != nil || len(evs) != 0 || next != "5" {
		t.Fatalf("%s next %s %v", eventIDs(evs), next, err)
	}
	// No cursor: from now on, and the cursor of now.
	evs, next, err = sbx.Events(ctx, sandbox.EventQuery{SandboxID: b.ID})
	if err != nil || len(evs) != 0 || next != "5" {
		t.Fatalf("%s next %s %v", eventIDs(evs), next, err)
	}
	// A cursor older than the events kept.
	f.pruned = 4
	if _, _, err := sbx.Events(ctx, sandbox.EventQuery{After: "3"}); !client.IsCode(err, sandbox.CodeCursorExpired) || client.StatusOf(err) != 410 {
		t.Errorf("an expired cursor: %v", err)
	}
	for _, q := range []sandbox.EventQuery{{WaitS: 61}, {WaitS: -1}, {Limit: 501}} {
		if _, _, err := sbx.Events(ctx, q); err == nil {
			t.Errorf("%+v: no error", q)
		}
	}
}

func TestStreamEventsReconnectsWithLastEventID(t *testing.T) {
	f, sbx := newFake(t)
	for range 3 {
		_, _ = sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine"})
	}
	first, _ := sbx.List(ctx, sandbox.ListOptions{Limit: -1})
	_ = first[2].Terminate(ctx, sandbox.TerminateOptions{})

	// The server ends each stream after one new event, and resends the last
	// one a reconnect names (at least once).
	f.sseBatch, f.sseRepeat = 1, true
	var got []sandbox.Event
	err := sbx.StreamEvents(ctx, sandbox.EventQuery{After: sandbox.FromOldest}, func(ev sandbox.Event) error {
		got = append(got, ev)
		if ev.Type == sandbox.EventSandboxExited {
			return io.EOF
		}
		return nil
	})
	if err != nil || eventIDs(got) != "1:sandbox.running 2:sandbox.running 3:sandbox.running 4:sandbox.exited" {
		t.Fatalf("%v: %s", err, eventIDs(got))
	}
	want := "after=0 last=|after=1 last=1|after=2 last=2|after=3 last=3"
	if strings.Join(f.sseConns, "|") != want {
		t.Errorf("connections\n got %s\nwant %s", strings.Join(f.sseConns, "|"), want)
	}
	// fn's error is returned as it is.
	boom := errors.New("boom")
	if err := sbx.StreamEvents(ctx, sandbox.EventQuery{After: sandbox.FromOldest}, func(sandbox.Event) error { return boom }); !errors.Is(err, boom) {
		t.Errorf("fn's error: %v", err)
	}
}

func TestStreamEventsFromNowMissesNothing(t *testing.T) {
	f, sbx := newFake(t)
	old, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine", Labels: map[string]string{"task": "t-81"}})
	// The first connection has nothing to send and ends; the sandbox ends
	// before the second.
	f.onSSE = func(n int) {
		if n == 2 {
			f.end(f.sandboxes[old.ID], "terminated", "terminated by API", nil, nil)
		}
	}
	var got []sandbox.Event
	err := sbx.StreamEvents(ctx, sandbox.EventQuery{Labels: map[string]string{"task": "t-81"}}, func(ev sandbox.Event) error {
		got = append(got, ev)
		return io.EOF
	})
	if err != nil || eventIDs(got) != "2:sandbox.exited" {
		t.Fatalf("%v: %s", err, eventIDs(got))
	}
	// It took the feed's position first, and streamed from there both times.
	if f.polls[0] != "label=task%3Dt-81" || strings.Join(f.sseConns, "|") != "after=1 last=|after=1 last=" {
		t.Errorf("polls %q, connections %q", f.polls, f.sseConns)
	}
}

func TestStreamEventsCursorExpired(t *testing.T) {
	f, sbx := newFake(t)
	f.pruned = 10
	err := sbx.StreamEvents(ctx, sandbox.EventQuery{After: "3"}, func(sandbox.Event) error { return nil })
	if !client.IsCode(err, sandbox.CodeCursorExpired) || len(f.sseConns) != 0 {
		t.Fatalf("%v after %d connections", err, len(f.sseConns))
	}
}
