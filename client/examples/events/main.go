// Command events follows the bucket's sandbox events as they happen and
// keeps a cursor, the ID of the last event it handled, in a file. Stop it and
// run it again: it goes on after that event and misses none in between (for
// up to 72 hours, how long events are kept).
//
//	CERE_TOKEN=<token> go run github.com/cerebellum-network/cere-ddc-sdk-go/client/examples/events@latest -label task:t-81
//
// -label key:value narrows to the sandboxes with that label; -cursor names the
// cursor's file (events.cursor in the working directory).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client/sandbox"
)

func main() {
	cursorFile := flag.String("cursor", "events.cursor", "file that keeps the ID of the last event handled")
	label := flag.String("label", "", "only the events of sandboxes with this label, key:value")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	c, err := client.NewFromEnv(client.WithUserAgent("cere-example-events"))
	if err != nil {
		log.Fatal(err)
	}
	sbx := sandbox.New(c)

	q := sandbox.EventQuery{Types: []string{sandbox.EventSandboxRunning, sandbox.EventSandboxExited, sandbox.EventExecExited}}
	if *label != "" {
		k, v, ok := strings.Cut(*label, ":")
		if !ok {
			log.Fatalf("-label %q: want key:value", *label)
		}
		q.Labels = map[string]string{k: v}
	}
	if b, err := os.ReadFile(*cursorFile); err == nil {
		q.After = strings.TrimSpace(string(b))
		fmt.Printf("going on after event %s\n", q.After)
	} else {
		fmt.Println("no cursor yet: from now on")
	}

	for {
		err = sbx.StreamEvents(ctx, q, func(ev sandbox.Event) error {
			show(ev)
			// Keep the cursor once the event is handled: a restart goes on
			// after it. An event may come twice across restarts; ev.ID tells.
			q.After = ev.ID
			return os.WriteFile(*cursorFile, []byte(ev.ID+"\n"), 0o600)
		})
		if client.IsCode(err, sandbox.CodeCursorExpired) {
			// Stopped for longer than events are kept: read the sandboxes you
			// follow as they are now (sbx.List, sbx.Get), then go on from now.
			fmt.Println("the cursor is older than the events kept; going on from now")
			q.After = ""
			continue
		}
		break
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}

func show(ev sandbox.Event) {
	switch ev.Type {
	case sandbox.EventSandboxRunning:
		fmt.Printf("%s %s running (started in %d ms)\n", ev.ID, ev.SandboxID, ev.StartupMS)
	case sandbox.EventSandboxExited:
		fmt.Printf("%s %s %s: %s (%s)", ev.ID, ev.SandboxID, ev.Status, ev.ExitReason, ev.ExitKind)
		if len(ev.Retained) > 0 && ev.RetainedUntil != nil {
			fmt.Printf(", %d files kept until %s", len(ev.Retained), ev.RetainedUntil.Format("15:04 MST"))
		}
		fmt.Println()
	case sandbox.EventExecExited:
		code := "none"
		if ev.ExitCode != nil {
			code = fmt.Sprint(*ev.ExitCode)
		}
		fmt.Printf("%s %s exec %s exited: code %s after %d ms\n", ev.ID, ev.SandboxID, ev.ExecID, code, ev.DurationMS)
	}
}
