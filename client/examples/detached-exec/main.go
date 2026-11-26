// Command detached-exec runs a command that outlives its caller: it starts
// the command detached in a sandbox, follows its output, and saves where it
// got to after every chunk. Stop it at any time (Ctrl-C) and run it again: it
// picks the same command up from the saved byte offsets, and nothing is
// printed twice. When the command ends, it ends the sandbox and removes the
// saved state.
//
//	CERE_TOKEN=<token> go run github.com/cerebellum-network/cere-ddc-sdk-go/client/examples/detached-exec@latest
//
// The state is kept in detached-exec.json in the working directory (-state to
// change it).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client/sandbox"
)

// state is where a run got to.
type state struct {
	// Run is the run's idempotency key, saved before anything is created.
	Run       string              `json:"run"`
	SandboxID string              `json:"sandbox_id"`
	ExecID    string              `json:"exec_id"`
	At        sandbox.OutputQuery `json:"at"`
}

func main() {
	path := flag.String("state", "detached-exec.json", "where to keep the sandbox, the exec and the offsets reached")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	c, err := client.NewFromEnv(client.WithUserAgent("cere-example-detached-exec"))
	if err != nil {
		log.Fatal(err)
	}
	sbx := sandbox.New(c)

	st, err := load(*path)
	if err != nil {
		log.Fatal(err)
	}
	if st.Run == "" {
		st.Run = fmt.Sprintf("detached-exec-example-%d", time.Now().UnixNano())
		if err := save(*path, st); err != nil {
			log.Fatal(err)
		}
	}
	var sb *sandbox.Sandbox
	var ex *sandbox.Exec
	if st.ExecID == "" {
		// The keys, saved before the calls, make both safe to repeat: if this
		// program dies before it saves the IDs, the next run gets the same
		// sandbox and the same exec back instead of starting new ones.
		sb, err = sbx.Create(ctx, sandbox.CreateRequest{
			Image:          "python:3.12-slim",
			Timeout:        15 * time.Minute,
			Labels:         map[string]string{"app": "detached-exec-example"},
			IdempotencyKey: st.Run,
		})
		if err != nil {
			log.Fatal(err)
		}
		ex, err = sb.ExecDetach(ctx, sandbox.ExecRequest{
			Command:        []string{"sh", "-c", "for i in $(seq 1 30); do echo step $i; sleep 1; done; echo finished >&2"},
			IdempotencyKey: st.Run + ":1",
		})
		if err != nil {
			log.Fatal(err)
		}
		st.SandboxID, st.ExecID = sb.ID, ex.ExecID
		if err := save(*path, st); err != nil {
			log.Fatal(err)
		}
		verb := "started"
		if ex.Replayed() {
			verb = "found the run's"
		}
		fmt.Printf("%s %s in %s\n", verb, ex.ExecID, sb.ID)
	} else {
		if sb, err = sbx.Get(ctx, st.SandboxID); err != nil {
			log.Fatal(err)
		}
		if ex, err = sb.GetExec(ctx, st.ExecID); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("resuming %s (%s) from stdout byte %d, stderr byte %d\n", ex.ExecID, ex.Status,
			st.At.StdoutOffset, st.At.StderrOffset)
	}

	exit, err := ex.Follow(ctx, st.At, func(ev sandbox.OutputEvent) error {
		switch ev.Name {
		case "stdout":
			fmt.Print(ev.Data)
		case "stderr":
			fmt.Fprint(os.Stderr, ev.Data)
		case "truncated":
			fmt.Fprintf(os.Stderr, "[%s before byte %d is no longer kept]\n", ev.Stream, ev.TruncatedBefore)
		}
		st.At = ev.Next
		return save(*path, st)
	})
	if errors.Is(err, context.Canceled) {
		fmt.Printf("\nstopped; run again to resume from %s\n", *path)
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	if exit.ExitCode != nil {
		fmt.Printf("exit %d after %d ms\n", *exit.ExitCode, exit.DurationMS)
	} else {
		fmt.Printf("no exit code: %s\n", exit.Error)
	}
	if err := sb.Terminate(context.Background(), sandbox.TerminateOptions{Reason: "example done"}); err != nil {
		log.Fatal(err)
	}
	_ = os.Remove(*path)
}

func load(path string) (state, error) {
	var st state
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	return st, json.Unmarshal(b, &st)
}

func save(path string, st state) error {
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
