package sandbox_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client/sandbox"
)

func TestExecDetach(t *testing.T) {
	f, sbx := newFake(t)
	sb, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine"})
	req := sandbox.ExecRequest{Command: []string{"seq", "5"}, Timeout: 10 * time.Minute, IdempotencyKey: "task:t-81:1"}
	ex, err := sb.ExecDetach(ctx, req)
	if err != nil || !ex.Running() || ex.Replayed() || ex.ExecID == "" || ex.SandboxID != sb.ID {
		t.Fatalf("%+v %v", ex, err)
	}
	if got := f.execs[0]; got != `{"command":["seq","5"],"stdin":null,"timeout_s":600,"detach":true}` {
		t.Errorf("exec body %s", got)
	}
	// The same key again: the same exec, nothing started.
	again, err := sb.ExecDetach(ctx, req)
	if err != nil || !again.Replayed() || again.ExecID != ex.ExecID || len(f.detached[sb.ID]) != 1 {
		t.Fatalf("replay: %+v %v", again, err)
	}
	req.Command = []string{"seq", "6"}
	if _, err := sb.ExecDetach(ctx, req); !client.IsCode(err, sandbox.CodeIdempotencyConflict) {
		t.Errorf("conflict: %v", err)
	}
	// A key is for detached execs only, and is checked before sending.
	n := len(f.execs)
	if _, err := sb.Exec(ctx, req); err == nil {
		t.Error("Exec with a key: no error")
	}
	if _, err := sb.ExecStream(ctx, req, nil, nil); err == nil {
		t.Error("ExecStream with a key: no error")
	}
	if _, err := sb.ExecDetach(ctx, sandbox.ExecRequest{Command: []string{"x"}, IdempotencyKey: "a\nb"}); err == nil {
		t.Error("a key with a newline: no error")
	}
	if len(f.execs) != n {
		t.Errorf("%d refused execs reached the API", len(f.execs)-n)
	}

	list, err := sb.Execs(ctx)
	if err != nil || len(list) != 1 || list[0].ExecID != ex.ExecID || list[0].SandboxID != sb.ID ||
		list[0].IdempotencyKey != "task:t-81:1" || strings.Join(list[0].Command, " ") != "seq 5" {
		t.Fatalf("list: %+v %v", list, err)
	}
	got, err := sb.GetExec(ctx, ex.ExecID)
	if err != nil || got.ExecID != ex.ExecID || got.StdoutBytes != 10 {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, err := sb.GetExec(ctx, "ex-nope"); !client.IsCode(err, sandbox.CodeExecNotFound) {
		t.Errorf("an unknown exec: %v", err)
	}
	// The fake ends an exec on its second read.
	if code, err := ex.Wait(ctx); err != nil || code != 0 || ex.Running() || ex.EndedAt == nil || ex.DurationMS == 0 {
		t.Fatalf("wait: %d %v %+v", code, err, ex)
	}

	// A detached exec still running when the sandbox ends has no exit code.
	ex2, err := sb.ExecDetach(ctx, sandbox.ExecRequest{Command: []string{"sleep", "60"}})
	if err != nil {
		t.Fatal(err)
	}
	_ = sb.Terminate(ctx, sandbox.TerminateOptions{})
	if code, err := ex2.Wait(ctx); err != nil || code != -1 || ex2.ExitCode != nil || ex2.Error == "" {
		t.Fatalf("after the sandbox ended: %d %v %+v", code, err, ex2)
	}
	if _, err := sb.ExecDetach(ctx, sandbox.ExecRequest{Command: []string{"true"}}); !client.IsCode(err, sandbox.CodeNotRunning) {
		t.Errorf("detach on an ended sandbox: %v", err)
	}
}

func TestExecOutputFromOffsets(t *testing.T) {
	f, sbx := newFake(t)
	f.maxRead = 4
	sb, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine"})
	ex, _ := sb.ExecDetach(ctx, sandbox.ExecRequest{Command: []string{"seq", "5"}})
	var stdout, stderr string
	from := sandbox.OutputQuery{}
	for range 5 {
		o, err := ex.Output(ctx, from)
		if err != nil {
			t.Fatal(err)
		}
		stdout, stderr = stdout+o.Stdout, stderr+o.Stderr
		if o.Next() == from {
			break
		}
		from = o.Next()
	}
	if stdout != "1\n2\n3\n4\n5\n" || stderr != "warn\n" || from != (sandbox.OutputQuery{StdoutOffset: 10, StderrOffset: 5}) {
		t.Fatalf("%q %q %+v", stdout, stderr, from)
	}
	if f.outputs[0] != "stderr_offset=0&stdout_offset=0" || f.outputs[1] != "stderr_offset=4&stdout_offset=4" {
		t.Errorf("queries %q", f.outputs)
	}
	// From the middle of stdout.
	o, err := ex.Output(ctx, sandbox.OutputQuery{StdoutOffset: 6, StderrOffset: 5})
	if err != nil || o.Stdout != "4\n5\n" || o.Stderr != "" || o.StdoutOffset != 10 {
		t.Fatalf("%+v %v", o, err)
	}
}

func TestExecStreamResumesFromOffsets(t *testing.T) {
	f, sbx := newFake(t)
	sb, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine"})
	ex, _ := sb.ExecDetach(ctx, sandbox.ExecRequest{Command: []string{"seq", "5"}})

	// The first connection drops after two chunks, the second after one:
	// each is opened again from the last offsets received.
	f.dropOutput = []int{2, 1}
	var out, errs bytes.Buffer
	exit, err := ex.Stream(ctx, sandbox.OutputQuery{}, &out, &errs)
	if err != nil || exit.ExitCode == nil || *exit.ExitCode != 0 || out.String() != "1\n2\n3\n4\n5\n" || errs.String() != "warn\n" {
		t.Fatalf("%+v %v %q %q", exit, err, out.String(), errs.String())
	}
	want := []string{"stderr_offset=0&stdout_offset=0&stream=1", "stderr_offset=0&stdout_offset=4&stream=1",
		"stderr_offset=0&stdout_offset=6&stream=1"}
	if strings.Join(f.outputs, " ") != strings.Join(want, " ") {
		t.Errorf("connections\n got %q\nwant %q", f.outputs, want)
	}

	// Follow from saved offsets: each event says where to go on from.
	var seen []string
	exit, err = ex.Follow(ctx, sandbox.OutputQuery{StdoutOffset: 6, StderrOffset: 5}, func(ev sandbox.OutputEvent) error {
		seen = append(seen, fmt.Sprintf("%s %q %d/%d", ev.Name, ev.Data, ev.Next.StdoutOffset, ev.Next.StderrOffset))
		return nil
	})
	if err != nil || exit == nil || strings.Join(seen, "|") != `stdout "4\n" 8/5|stdout "5\n" 10/5|exit "" 10/5` {
		t.Fatalf("%v %q", err, seen)
	}
	// io.EOF from fn stops early, without an error.
	exit, err = ex.Follow(ctx, sandbox.OutputQuery{}, func(sandbox.OutputEvent) error { return io.EOF })
	if err != nil || exit != nil {
		t.Fatalf("stop early: %+v %v", exit, err)
	}
	// A writer's error ends the call.
	if _, err := ex.Stream(ctx, sandbox.OutputQuery{}, failingWriter{}, nil); err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("writer error: %v", err)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("disk full") }

func TestExecOutputTruncated(t *testing.T) {
	f, sbx := newFake(t)
	f.keepOutput = 4 // the node kept the last 4 bytes of each stream
	sb, _ := sbx.Create(ctx, sandbox.CreateRequest{Image: "alpine"})
	ex, _ := sb.ExecDetach(ctx, sandbox.ExecRequest{Command: []string{"seq", "5"}})
	if err := ex.Refresh(ctx); err != nil || ex.StdoutTruncatedBefore != 6 || ex.StderrTruncatedBefore != 1 {
		t.Fatalf("%+v %v", ex, err)
	}
	var seen []string
	if _, err := ex.Follow(ctx, sandbox.OutputQuery{}, func(ev sandbox.OutputEvent) error {
		seen = append(seen, fmt.Sprintf("%s %s%q %d %d/%d", ev.Name, ev.Stream, ev.Data, ev.TruncatedBefore, ev.Next.StdoutOffset, ev.Next.StderrOffset))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	want := `truncated stdout"" 6 6/0|stdout "4\n" 0 8/0|stdout "5\n" 0 10/0|truncated stderr"" 1 10/1|stderr "arn\n" 0 10/5|exit "" 0 10/5`
	if strings.Join(seen, "|") != want {
		t.Fatalf("\n got %s\nwant %s", strings.Join(seen, "|"), want)
	}
	o, err := ex.Output(ctx, sandbox.OutputQuery{})
	if err != nil || o.Stdout != "4\n5\n" || o.StdoutTruncatedBefore != 6 {
		t.Fatalf("%+v %v", o, err)
	}
}
