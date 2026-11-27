package sandbox_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client/sandbox"
)

// fake is a compute gateway in memory: the sandbox API's routes, shapes and
// codes, enough to drive every call of the package.
type fake struct {
	t  *testing.T
	mu sync.Mutex

	sandboxes map[string]map[string]any // id -> the sandbox as JSON
	order     []string
	files     map[string]map[string][]byte // id -> path -> data
	secrets   map[string]map[string]any    // agent ("" for the bucket) + "/" + name -> info and value
	runSecret map[string]map[string]string // id -> name -> value
	records   []map[string]any
	egressLog []map[string]any
	keys      map[string]string      // Idempotency-Key -> sandbox id
	keyBodies map[string]string      // Idempotency-Key -> the create body, secret values left out
	detached  map[string][]*fakeExec // sandbox id -> its detached execs, newest first
	feed      []map[string]any       // the bucket's events, oldest first; an event's id is its index + 1
	webhooks  []map[string]any
	delivered map[string][]map[string]any // webhook id -> its delivery attempts, newest first

	// What the tests check.
	creates    []map[string]any // create bodies as sent
	execs      []string         // exec bodies as sent
	tokens     []string         // the token of each request, in order
	createFail []int            // statuses to answer the next creates with
	startAs    string           // the status a create answers (default running)
	failStart  bool             // a created sandbox fails to start
	loseCreate int              // creates that make the sandbox but answer 502, as if the answer was lost
	keepOutput int              // bytes of each stream of a detached exec kept (0: all)
	maxRead    int              // bytes of each stream one output read returns (0: 1 MiB)
	dropOutput []int            // per output stream connection, in turn: drop it after this many chunks
	outputs    []string         // the query of each output request
	pruned     int              // event ids below this are no longer kept (cursor_expired)
	sseBatch   int              // new events sent per event stream connection before the server ends it (0: all)
	sseRepeat  bool             // a reconnect is first sent the event its Last-Event-ID names again
	onSSE      func(n int)      // called, with the lock held, at the n-th event stream connection
	sseConns   []string         // "after=<after> last=<Last-Event-ID>" of each event stream connection
	polls      []string         // the query of each events poll
	hookBodies []map[string]any // webhook create bodies as sent
}

// fakeExec is a detached exec: the Exec as JSON, its body (for its key) and
// what it wrote. It runs until it is read twice, or streamed to its end.
type fakeExec struct {
	m              map[string]any
	body           string
	stdout, stderr string
	reads          int
}

// exitExec ends a detached exec of sandbox id and publishes exec.exited.
func (f *fake) exitExec(id string, e *fakeExec, code any, errMsg any) {
	if e.m["status"] == sandbox.ExecStatusExited {
		return
	}
	e.m["status"], e.m["exit_code"], e.m["error"] = sandbox.ExecStatusExited, code, errMsg
	e.m["ended_at"], e.m["duration_ms"] = time.Now().UTC().Format(time.RFC3339), 1200
	ev := map[string]any{}
	for _, k := range []string{"exec_id", "status", "exit_code", "timed_out", "duration_ms", "stdout_bytes", "stderr_bytes", "idempotency_key", "error"} {
		ev[k] = e.m[k]
	}
	f.publish(f.sandboxes[id], sandbox.EventExecExited, ev)
}

// publish appends an event of sb to the feed.
func (f *fake) publish(sb map[string]any, typ string, fields map[string]any) {
	ev := map[string]any{"id": strconv.Itoa(len(f.feed) + 1), "type": typ, "at": time.Now().UTC().Format(time.RFC3339Nano),
		"sandbox_id": sb["id"], "bucket": sb["bucket"], "labels": sb["labels"]}
	for k, v := range fields {
		ev[k] = v
	}
	f.feed = append(f.feed, ev)
}

func (f *fake) publishRunning(sb map[string]any) {
	f.publish(sb, sandbox.EventSandboxRunning, map[string]any{"status": "running", "started_at": sb["started_at"],
		"ready_at": sb["ready_at"], "startup_ms": 900, "image": sb["image"]})
}

func (f *fake) publishExited(sb map[string]any) {
	ev := map[string]any{}
	for _, k := range []string{"status", "exit_reason", "exit_kind", "exit_code", "ended_at", "retained", "not_retained", "retained_until", "usage"} {
		ev[k] = sb[k]
	}
	f.publish(sb, sandbox.EventSandboxExited, ev)
}

func newFake(t *testing.T) (*fake, *sandbox.Service) {
	f := &fake{t: t, sandboxes: map[string]map[string]any{}, files: map[string]map[string][]byte{},
		secrets: map[string]map[string]any{}, runSecret: map[string]map[string]string{}, startAs: sandbox.StatusRunning,
		keys: map[string]string{}, keyBodies: map[string]string{}, detached: map[string][]*fakeExec{},
		delivered: map[string][]map[string]any{}}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	c, err := client.New(client.WithBaseURL(srv.URL), client.WithToken("tok"),
		client.WithRetry(client.RetryPolicy{MaxAttempts: 3, MinBackoff: time.Millisecond, MaxBackoff: 2 * time.Millisecond}))
	if err != nil {
		t.Fatal(err)
	}
	return f, sandbox.New(c)
}

func apiError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg, "code": code})
}

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func events(w http.ResponseWriter, evs ...[2]string) {
	w.Header().Set("Content-Type", "text/event-stream")
	_, _ = io.WriteString(w, ": ping\n\n")
	for _, ev := range evs {
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev[0], ev[1])
	}
}

func (f *fake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	tok, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || tok == "" {
		apiError(w, 401, "UNAUTHORIZED", "no token")
		return
	}
	f.tokens = append(f.tokens, tok)
	for _, l := range r.URL.Query()["label"] {
		if !labelFilterRE.MatchString(l) {
			apiError(w, 400, "invalid_request", fmt.Sprintf("label %q must be key=value, the key matching [a-z0-9._/-]{1,63}", l))
			return
		}
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/"), "/")
	// The bucket-wide routes live under /v1/sandboxes/ (their names are never
	// sandbox IDs) and nowhere else.
	switch {
	case bucketRoute[parts[0]]:
		apiError(w, 404, "not_found", r.Method+" "+r.URL.Path+" is not a route of this API")
		return
	case parts[0] == "sandboxes" && len(parts) >= 2 && bucketRoute[parts[1]]:
		parts = parts[1:]
	}
	switch {
	case parts[0] == "sandboxes" && len(parts) == 1 && r.Method == http.MethodPost:
		f.create(w, r)
	case parts[0] == "sandboxes" && len(parts) == 1 && r.Method == http.MethodGet:
		f.list(w, r)
	case parts[0] == "sandboxes" && len(parts) >= 2:
		sb := f.sandboxes[parts[1]]
		if sb == nil {
			apiError(w, 404, "not_found", "no such sandbox")
			return
		}
		f.onSandbox(w, r, sb, parts[2:])
	case parts[0] == "secrets":
		f.bucketSecrets(w, r, parts[1:])
	case parts[0] == "usage" && len(parts) == 2 && parts[1] == "summary":
		f.usageSummary(w, r)
	case parts[0] == "egress" && len(parts) == 2 && parts[1] == "summary":
		f.egressSummary(w, r)
	case parts[0] == "limits" && len(parts) == 1:
		active := 0
		for _, sb := range f.sandboxes {
			if !ended(sb) {
				active++
			}
		}
		reply(w, 200, map[string]any{"bucket": 4116, "max_active_sandboxes": 20, "active_sandboxes": active})
	case parts[0] == "usage" && len(parts) == 2:
		reply(w, 200, f.page(r, f.records, func(rec map[string]any) bool {
			id, _ := rec["sandbox_id"].(string)
			dac := r.URL.Query().Get("dac_status")
			return (r.URL.Query().Get("sandbox_id") == "" || r.URL.Query().Get("sandbox_id") == id) &&
				(dac == "" || dac == rec["dac_status"]) && f.labelsMatch(f.sandboxes[id], r.URL.Query()["label"])
		}))
	case parts[0] == "events" && len(parts) == 1 && r.Method == http.MethodGet:
		f.events(w, r)
	case parts[0] == "webhooks":
		f.hooks(w, r, parts[1:])
	case parts[0] == "egress" && len(parts) == 2:
		reply(w, 200, f.page(r, f.egressLog, func(e map[string]any) bool {
			return r.URL.Query().Get("sandbox_id") == "" || r.URL.Query().Get("sandbox_id") == e["sandbox_id"]
		}))
	default:
		apiError(w, 404, "not_found", "no route")
	}
}

func (f *fake) create(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, 400, "invalid_request", err.Error())
		return
	}
	f.creates = append(f.creates, body)
	key := r.Header.Get("Idempotency-Key")
	if id, ok := f.keys[key]; ok && key != "" {
		if f.keyBodies[key] != withoutSecretValues(body) {
			apiError(w, 409, "idempotency_conflict", "the idempotency key was used with another body")
			return
		}
		w.Header().Set("Idempotent-Replayed", "true")
		reply(w, 200, f.sandboxes[id])
		return
	}
	if len(f.createFail) > 0 {
		status := f.createFail[0]
		f.createFail = f.createFail[1:]
		apiError(w, status, map[int]string{503: "no_capacity", 409: "limit_active_sandboxes", 502: "node_unreachable"}[status], "refused")
		return
	}
	if cred, _ := body["registry_credential"].(string); cred != "" {
		if code, msg := f.checkRegistry(cred, body["image"].(string)); code != "" {
			apiError(w, 400, code, msg)
			return
		}
	}
	id := fmt.Sprintf("sb-%026d", len(f.order)+1)
	res := map[string]any{"cpu": 0.25, "memory": 128, "cpu_limit": 8, "memory_limit": 1024, "pids": 2048, "disk": 512}
	if req, ok := body["resources"].(map[string]any); ok {
		for k, v := range req {
			res[k] = v
		}
	}
	var secrets []any
	values := map[string]string{}
	if list, ok := body["secrets"].([]any); ok {
		for _, s := range list {
			m := s.(map[string]any)
			values[m["name"].(string)] = m["value"].(string)
			secrets = append(secrets, map[string]any{"name": m["name"], "type": m["type"], "header": m["header"], "hosts": m["hosts"]})
		}
	}
	egress := body["egress"]
	if egress == nil {
		egress = map[string]any{"allow": []any{}}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var keyField any
	if key != "" {
		keyField = key
	}
	sb := map[string]any{
		"id": id, "status": f.startAs, "bucket": 4116, "node": strings.Repeat("ab", 32), "runtime": body["runtime"],
		"image":   map[string]any{"ref": body["image"], "digest": "sha256:" + strings.Repeat("0f", 32), "size_bytes": 52428800, "cached": false, "pull_ms": 840},
		"command": body["command"], "workdir": "/workspace", "resources": res,
		"timeout_s": body["timeout_s"], "labels": body["labels"], "egress": egress, "secrets": secrets,
		"created_at": now, "started_at": nil, "ready_at": nil, "exit_reason": nil, "exit_kind": nil, "exit_code": nil,
		"retain": body["retain"], "retained": []any{}, "not_retained": []any{}, "retained_until": nil,
		"registry_credential": body["registry_credential"], "idempotency_key": keyField,
		"usage": map[string]any{"wall_s": 0, "billing_basis": ""},
	}
	if f.startAs == sandbox.StatusRunning {
		sb["started_at"], sb["ready_at"] = now, now
		f.publishRunning(sb)
	}
	if f.failStart {
		sb["status"] = sandbox.StatusStarting
	}
	f.sandboxes[id] = sb
	f.order = append([]string{id}, f.order...)
	f.files[id] = map[string][]byte{}
	f.runSecret[id] = values
	if key != "" {
		f.keys[key], f.keyBodies[key] = id, withoutSecretValues(body)
	}
	if f.loseCreate > 0 {
		f.loseCreate--
		apiError(w, 502, "node_unreachable", "the answer was lost")
		return
	}
	reply(w, 201, sb)
}

// withoutSecretValues is a create body as an idempotency key compares it.
func withoutSecretValues(body map[string]any) string {
	cp := map[string]any{}
	for k, v := range body {
		cp[k] = v
	}
	if list, ok := body["secrets"].([]any); ok {
		var out []any
		for _, s := range list {
			m := map[string]any{}
			for k, v := range s.(map[string]any) {
				if k != "value" {
					m[k] = v
				}
			}
			out = append(out, m)
		}
		cp["secrets"] = out
	}
	return mustJSON(cp)
}

// checkRegistry resolves a registry credential as the API does: the bucket's
// secret of that name, of type registry, bound to the image's registry host.
func (f *fake) checkRegistry(name, image string) (code, msg string) {
	s := f.secrets["/"+name]
	switch {
	case s == nil:
		return "unknown_credential", "no secret " + name
	case s["type"] != sandbox.SecretTypeRegistry:
		return "invalid_request", name + " is not a registry secret"
	}
	host := "docker.io"
	if first, _, ok := strings.Cut(image, "/"); ok && strings.ContainsAny(first, ".:") {
		host = first
	}
	for _, h := range s["hosts"].([]any) {
		if h == host {
			return "", ""
		}
	}
	return "credential_not_bound", "not bound to " + host
}

// end ends a running sandbox as its node would: kind and reason, the exit
// code, the files kept (the create's retain, unless it is on failure and
// the end is not one, and the paths of a DELETE's ?retain=), and a final
// usage record carrying its labels.
func (f *fake) end(sb map[string]any, kind, reason string, code any, extra []string) {
	id := sb["id"].(string)
	now := time.Now().UTC()
	var paths []string
	ttl := 86400.0
	if retain, ok := sb["retain"].(map[string]any); ok {
		if t, ok := retain["ttl_s"].(float64); ok {
			ttl = t
		}
		success := kind == "terminated" || (kind == "main_exited" && code == 0)
		if retain["on"] != sandbox.RetainOnFailure || !success {
			for _, p := range retain["paths"].([]any) {
				paths = append(paths, p.(string))
			}
		}
	}
	paths = append(paths, extra...)
	kept, missing := []any{}, []any{}
	for _, p := range paths {
		var files []string
		for name := range f.files[id] {
			if name == p || (strings.HasSuffix(p, "/") && strings.HasPrefix(name, p)) || strings.HasPrefix(name, p+"/") {
				files = append(files, name)
			}
		}
		slices.Sort(files)
		if len(files) == 0 {
			missing = append(missing, map[string]any{"path": p, "reason": "it does not exist in the sandbox"})
		}
		for _, name := range files {
			kept = append(kept, map[string]any{"path": name, "size": len(f.files[id][name])})
		}
	}
	for _, ex := range f.detached[id] {
		f.exitExec(id, ex, nil, "the sandbox ended while the command ran")
	}
	sb["status"], sb["exit_kind"], sb["exit_reason"], sb["exit_code"] = "terminated", kind, reason, code
	sb["ended_at"], sb["retained"], sb["not_retained"] = now.Format(time.RFC3339), kept, missing
	if len(kept) > 0 {
		sb["retained_until"] = now.Add(time.Duration(ttl) * time.Second).Format(time.RFC3339)
	}
	sb["usage"] = map[string]any{"wall_s": 12.5, "billing_basis": "request_or_usage", "billed_cpu_s": 3.2, "cpu_avg": 0.04}
	f.publishExited(sb)
	f.records = append(f.records, map[string]any{"record_id": "r-" + id, "sandbox_id": id, "final": true,
		"labels": sb["labels"], "billed_cpu_s": 3.2, "billed_ram_gib_s": 1.5, "cpu_s": 0.5, "wall_s": 12.5,
		"memory_limit": 1024, "dac_status": "pushed"})
}

// exitMain ends a sandbox by its main process exiting with code.
func (f *fake) exitMain(id string, code int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	sb := f.sandboxes[id]
	if sb["started_at"] == nil {
		sb["started_at"] = time.Now().UTC().Format(time.RFC3339)
	}
	f.end(sb, "main_exited", fmt.Sprintf("exited with code %d", code), code, nil)
}

// labelFilterRE is a label filter as the gateway takes it, key=value.
var labelFilterRE = regexp.MustCompile(`^[a-z0-9._/-]{1,63}=`)

func (f *fake) labelsMatch(sb map[string]any, filters []string) bool {
	labels, _ := sb["labels"].(map[string]any)
	for _, flt := range filters {
		k, v, _ := strings.Cut(flt, "=")
		if labels[k] != v {
			return false
		}
	}
	return true
}

// page serves items that match keep, a page of ?limit= at ?cursor= (an index).
func (f *fake) page(r *http.Request, items []map[string]any, keep func(map[string]any) bool) map[string]any {
	var match []map[string]any
	for _, it := range items {
		if keep(it) {
			match = append(match, it)
		}
	}
	start, _ := strconv.Atoi(r.URL.Query().Get("cursor"))
	n, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if n <= 0 || n > 500 {
		f.t.Errorf("limit %q out of range", r.URL.Query().Get("limit"))
		n = 100
	}
	end := min(start+n, len(match))
	var next any
	if end < len(match) {
		next = strconv.Itoa(end)
	}
	page := match[start:end]
	if page == nil {
		page = []map[string]any{}
	}
	return map[string]any{"items": page, "next_cursor": next}
}

func (f *fake) list(w http.ResponseWriter, r *http.Request) {
	var all []map[string]any
	for _, id := range f.order {
		all = append(all, f.sandboxes[id])
	}
	reply(w, 200, f.page(r, all, func(sb map[string]any) bool {
		st := r.URL.Query().Get("status")
		return (st == "" || sb["status"] == st) && f.labelsMatch(sb, r.URL.Query()["label"])
	}))
}

func ended(sb map[string]any) bool { return sb["status"] == "terminated" || sb["status"] == "failed" }

func (f *fake) onSandbox(w http.ResponseWriter, r *http.Request, sb map[string]any, rest []string) {
	id := sb["id"].(string)
	route := r.Method + " " + strings.Join(rest, "/")
	switch {
	case route == "GET ":
		if sb["status"] == sandbox.StatusStarting {
			if f.failStart {
				sb["status"], sb["exit_reason"], sb["exit_kind"] = "failed", "failed: image pull no such image", "failed"
				f.publishExited(sb)
			} else {
				now := time.Now().UTC().Format(time.RFC3339)
				sb["status"], sb["started_at"], sb["ready_at"] = "running", now, now
				f.publishRunning(sb)
			}
		}
		reply(w, 200, sb)
	case route == "DELETE ":
		if !ended(sb) {
			reason := "terminated by API"
			if q := r.URL.Query().Get("reason"); q != "" {
				reason += ": " + q
			}
			var extra []string
			if q := r.URL.Query().Get("retain"); q != "" {
				extra = strings.Split(q, ",")
			}
			f.end(sb, "terminated", reason, nil, extra)
		}
		reply(w, 200, sb)
	case ended(sb) && route == "GET files":
		path := r.URL.Query().Get("path")
		for _, kept := range sb["retained"].([]any) {
			if kept.(map[string]any)["path"] == path {
				_, _ = w.Write(f.files[id][path])
				return
			}
		}
		apiError(w, 410, "file_gone", "not retained")
	case r.Method == http.MethodGet && len(rest) >= 1 && rest[0] == "exec":
		f.readExec(w, r, id, rest[1:])
	case ended(sb):
		apiError(w, 409, "not_running", "the sandbox has ended")
	case route == "POST exec" && f.detach(w, r, id):
		// answered: a detached exec
	case route == "POST exec":
		var req struct {
			Command  []string `json:"command"`
			Stdin    *string  `json:"stdin"`
			TimeoutS int      `json:"timeout_s"`
		}
		raw := f.execs[len(f.execs)-1]
		_ = json.Unmarshal([]byte(raw), &req)
		out := strings.Join(req.Command, " ")
		if req.Stdin != nil {
			out += " <" + *req.Stdin
		}
		if r.URL.Query().Get("stream") == "1" && len(req.Command) > 0 && req.Command[0] == "no-such-program" {
			events(w, [2]string{"exit", `{"exit_code":-1,"duration_ms":0,"timed_out":false,` +
				`"error":"exec: \"no-such-program\": executable file not found in $PATH"}`})
			return
		}
		if r.URL.Query().Get("stream") == "1" {
			events(w, [2]string{"stdout", `{"data":"out:"}`}, [2]string{"stderr", `{"data":"err\n"}`},
				[2]string{"stdout", mustJSON(map[string]string{"data": out + "\n"})},
				[2]string{"exit", fmt.Sprintf(`{"exit_code":%d,"duration_ms":5,"timed_out":false}`, req.TimeoutS)})
			return
		}
		if len(req.Command) > 0 && req.Command[0] == "flood" {
			// 9 MiB of output, under the API's 16 MiB cap, whose JSON is six
			// times as long (54 MiB): each control character is sent as \u00XX.
			reply(w, 200, map[string]any{"exit_code": 0, "stdout": strings.Repeat("\x01", 9<<20), "stderr": "",
				"duration_ms": 4, "timed_out": false})
			return
		}
		reply(w, 200, map[string]any{"exit_code": 0, "stdout": out + "\n", "stderr": "", "duration_ms": 4,
			"timed_out": false})
	case route == "PUT files":
		data, _ := io.ReadAll(r.Body)
		f.files[id][r.URL.Query().Get("path")] = data
		w.WriteHeader(204)
	case route == "GET files":
		data, ok := f.files[id][r.URL.Query().Get("path")]
		if !ok {
			apiError(w, 404, "file_not_found", "no such file")
			return
		}
		_, _ = w.Write(data)
	case route == "GET logs":
		events(w, [2]string{"stdout", `{"data":"booted\n"}`}, [2]string{"stderr", `{"data":"warn\n"}`}, [2]string{"end", `{}`},
			[2]string{"stdout", `{"data":"after the end\n"}`})
	case route == "GET usage":
		reply(w, 200, map[string]any{"wall_s": 7.5, "cpu_s": 0.4, "billing_basis": "request_or_usage", "billed_cpu_s": 1.9})
	case route == "PUT egress":
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		allow, ok := body["allow"].([]any)
		if !ok {
			apiError(w, 400, "invalid_request", "no allow")
			return
		}
		for _, rule := range allow {
			if cred, _ := rule.(map[string]any)["credential"].(string); cred == "nope" {
				apiError(w, 400, "unknown_credential", "credential nope does not resolve")
				return
			}
		}
		sb["egress"] = map[string]any{"allow": allow}
		f.egressLog = append(f.egressLog, map[string]any{"id": len(f.egressLog) + 1, "sandbox_id": id, "decision": "policy",
			"reason": "rule added", "path": nil, "status": nil})
		reply(w, 200, sb)
	case r.Method == http.MethodPut && len(rest) == 2 && rest[0] == "secrets":
		if _, ok := f.runSecret[id][rest[1]]; !ok {
			apiError(w, 404, "secret_not_found", "no such secret")
			return
		}
		var body struct{ Value string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.runSecret[id][rest[1]] = body.Value
		w.WriteHeader(204)
	default:
		apiError(w, 405, "method_not_allowed", route)
	}
}

// detach records an exec's body and, when it asks for a detached exec,
// starts one and answers 202; it reports whether it answered.
func (f *fake) detach(w http.ResponseWriter, r *http.Request, id string) bool {
	raw, _ := io.ReadAll(r.Body)
	f.execs = append(f.execs, string(raw))
	var req struct {
		Command []string `json:"command"`
		Detach  bool     `json:"detach"`
	}
	_ = json.Unmarshal(raw, &req)
	key := r.Header.Get("Idempotency-Key")
	if !req.Detach {
		if key != "" {
			apiError(w, 400, "invalid_request", "Idempotency-Key is only taken with detach")
			return true
		}
		return false
	}
	brief := func(e *fakeExec) map[string]any {
		return map[string]any{"exec_id": e.m["exec_id"], "status": e.m["status"], "started_at": e.m["started_at"]}
	}
	for _, e := range f.detached[id] {
		if key != "" && e.m["idempotency_key"] == key {
			if e.body != string(raw) {
				apiError(w, 409, "idempotency_conflict", "the idempotency key was used with another body")
				return true
			}
			w.Header().Set("Idempotent-Replayed", "true")
			reply(w, 202, brief(e))
			return true
		}
	}
	// The command "seq N" writes the numbers 1 to N, a line each, and a line
	// to stderr; any other its words.
	var stdout string
	if len(req.Command) == 2 && req.Command[0] == "seq" {
		n, _ := strconv.Atoi(req.Command[1])
		for i := 1; i <= n; i++ {
			stdout += strconv.Itoa(i) + "\n"
		}
	} else {
		stdout = strings.Join(req.Command, " ") + "\n"
	}
	var keyField any
	if key != "" {
		keyField = key
	}
	e := &fakeExec{body: string(raw), stdout: stdout, stderr: "warn\n", m: map[string]any{
		"exec_id": fmt.Sprintf("ex-%024d", len(f.execs)), "status": sandbox.ExecStatusRunning, "command": req.Command,
		"started_at": time.Now().UTC().Format(time.RFC3339), "ended_at": nil, "duration_ms": nil, "exit_code": nil,
		"timed_out": false, "error": nil, "stdout_bytes": len(stdout), "stderr_bytes": 5,
		"stdout_truncated_before": 0, "stderr_truncated_before": 0, "output_interrupted": false, "idempotency_key": keyField,
	}}
	if f.keepOutput > 0 {
		e.m["stdout_truncated_before"] = max(0, len(e.stdout)-f.keepOutput)
		e.m["stderr_truncated_before"] = max(0, len(e.stderr)-f.keepOutput)
	}
	f.detached[id] = append([]*fakeExec{e}, f.detached[id]...)
	reply(w, 202, brief(e))
	return true
}

// readExec serves the GET routes of a sandbox's detached execs.
func (f *fake) readExec(w http.ResponseWriter, r *http.Request, id string, rest []string) {
	if len(rest) == 0 {
		items := []any{}
		for _, e := range f.detached[id] {
			items = append(items, e.m)
		}
		reply(w, 200, map[string]any{"items": items})
		return
	}
	var e *fakeExec
	for _, x := range f.detached[id] {
		if x.m["exec_id"] == rest[0] {
			e = x
		}
	}
	if e == nil {
		apiError(w, 404, "exec_not_found", "no exec "+rest[0])
		return
	}
	if len(rest) == 1 {
		if e.reads++; e.reads >= 2 {
			f.exitExec(id, e, 0, nil)
		}
		reply(w, 200, e.m)
		return
	}
	q := r.URL.Query()
	f.outputs = append(f.outputs, r.URL.RawQuery)
	from := map[string]int{}
	for _, name := range []string{"stdout", "stderr"} {
		from[name], _ = strconv.Atoi(q.Get(name + "_offset"))
	}
	kept := map[string]int{"stdout": e.m["stdout_truncated_before"].(int), "stderr": e.m["stderr_truncated_before"].(int)}
	data := map[string]string{"stdout": e.stdout, "stderr": e.stderr}
	if q.Get("stream") != "1" {
		maxRead := f.maxRead
		if maxRead == 0 {
			maxRead = 1 << 20
		}
		out := map[string]any{"status": e.m["status"], "exit_code": e.m["exit_code"]}
		for _, name := range []string{"stdout", "stderr"} {
			start := min(max(from[name], kept[name]), len(data[name]))
			end := min(start+maxRead, len(data[name]))
			out[name], out[name+"_offset"], out[name+"_truncated_before"] = data[name][start:end], end, kept[name]
		}
		reply(w, 200, out)
		return
	}
	drop := -1
	if len(f.dropOutput) > 0 {
		drop, f.dropOutput = f.dropOutput[0], f.dropOutput[1:]
	}
	w.Header().Set("Content-Type", "text/event-stream")
	_, _ = io.WriteString(w, ": ping\n\n")
	sent := 0
	for _, name := range []string{"stdout", "stderr"} {
		at := from[name]
		if at < kept[name] {
			fmt.Fprintf(w, "event: truncated\ndata: %s\n\n", mustJSON(map[string]any{"stream": name, "truncated_before": kept[name]}))
			at = kept[name]
		}
		// A chunk is a line.
		for at < len(data[name]) {
			if sent == drop {
				w.(http.Flusher).Flush()
				panic(http.ErrAbortHandler) // the connection drops
			}
			end := at + strings.IndexByte(data[name][at:], '\n') + 1
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, mustJSON(map[string]any{"data": data[name][at:end], "offset": end}))
			at, sent = end, sent+1
		}
	}
	f.exitExec(id, e, 0, nil)
	fmt.Fprintf(w, "event: exit\ndata: %s\n\n", mustJSON(map[string]any{"exit_code": e.m["exit_code"],
		"duration_ms": e.m["duration_ms"], "timed_out": false, "error": e.m["error"]}))
}

// events serves GET /v1/sandboxes/events: a long-poll, or an event stream that sends
// what it has (sseBatch new events at most) and ends.
func (f *fake) events(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	cursor, last := q.Get("after"), r.Header.Get("Last-Event-ID")
	if last != "" {
		cursor = last
	}
	start := len(f.feed) // after absent: from now on
	if cursor != "" {
		start, _ = strconv.Atoi(cursor)
		if start < f.pruned {
			apiError(w, 410, "cursor_expired", "the cursor is older than the events kept")
			return
		}
	}
	var types []string
	if t := q.Get("types"); t != "" {
		types = strings.Split(t, ",")
	}
	match := func(ev map[string]any) bool {
		sb := f.sandboxes[ev["sandbox_id"].(string)]
		return (len(types) == 0 || slices.Contains(types, ev["type"].(string))) &&
			(q.Get("sandbox_id") == "" || q.Get("sandbox_id") == ev["sandbox_id"]) && f.labelsMatch(sb, q["label"])
	}
	var out []map[string]any
	for _, ev := range f.feed[min(start, len(f.feed)):] {
		if match(ev) {
			out = append(out, ev)
		}
	}
	if !strings.HasPrefix(r.Header.Get("Accept"), "text/event-stream") {
		f.polls = append(f.polls, r.URL.RawQuery)
		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit == 0 {
			limit = 100
		}
		out = out[:min(limit, len(out))]
		next := cursor
		switch {
		case len(out) > 0:
			next = out[len(out)-1]["id"].(string)
		case cursor == "":
			next = strconv.Itoa(len(f.feed))
		}
		if out == nil {
			out = []map[string]any{}
		}
		reply(w, 200, map[string]any{"events": out, "next": next})
		return
	}
	f.sseConns = append(f.sseConns, fmt.Sprintf("after=%s last=%s", q.Get("after"), last))
	if f.onSSE != nil {
		f.onSSE(len(f.sseConns))
		out = nil
		for _, ev := range f.feed[min(start, len(f.feed)):] {
			if match(ev) {
				out = append(out, ev)
			}
		}
	}
	if f.sseBatch > 0 {
		out = out[:min(f.sseBatch, len(out))]
	}
	if n, _ := strconv.Atoi(last); f.sseRepeat && n > 0 && match(f.feed[n-1]) {
		out = append([]map[string]any{f.feed[n-1]}, out...) // at least once
	}
	w.Header().Set("Content-Type", "text/event-stream")
	_, _ = io.WriteString(w, ": ping\n\n")
	for _, ev := range out {
		fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", ev["id"], ev["type"], mustJSON(ev))
	}
}

// hooks serves /v1/sandboxes/webhooks.
func (f *fake) hooks(w http.ResponseWriter, r *http.Request, rest []string) {
	find := func(id string) int {
		return slices.IndexFunc(f.webhooks, func(h map[string]any) bool { return h["id"] == id })
	}
	switch {
	case r.Method == http.MethodPost && len(rest) == 0:
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.hookBodies = append(f.hookBodies, body)
		if len(f.webhooks) >= sandbox.MaxWebhooks {
			apiError(w, 409, "limit_webhooks", "the bucket has its 10 webhooks")
			return
		}
		types := body["types"]
		if types == nil {
			types = []any{}
		}
		h := map[string]any{"id": fmt.Sprintf("wh-%d", len(f.hookBodies)), "url": body["url"], "types": types,
			"label": body["label"], "created_at": time.Now().UTC().Format(time.RFC3339)}
		f.webhooks = append(f.webhooks, h)
		reply(w, 201, h)
	case r.Method == http.MethodGet && len(rest) == 0:
		reply(w, 200, map[string]any{"items": f.webhooks})
	case r.Method == http.MethodDelete && len(rest) == 1:
		i := find(rest[0])
		if i < 0 {
			apiError(w, 404, "not_found", "no webhook "+rest[0])
			return
		}
		f.webhooks = slices.Delete(f.webhooks, i, i+1)
		w.WriteHeader(204)
	case r.Method == http.MethodGet && len(rest) == 2 && rest[1] == "deliveries":
		if find(rest[0]) < 0 {
			apiError(w, 404, "not_found", "no webhook "+rest[0])
			return
		}
		items := f.delivered[rest[0]]
		if n, _ := strconv.Atoi(r.URL.Query().Get("limit")); n > 0 {
			items = items[:min(n, len(items))]
		}
		reply(w, 200, map[string]any{"items": items})
	default:
		apiError(w, 405, "method_not_allowed", r.Method)
	}
}

// usageSummary serves GET /v1/sandboxes/usage/summary: the records summed per
// sandbox of the bucket whose labels match, newest first and paged, and in
// all.
func (f *fake) usageSummary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sum := func(into map[string]any, rec map[string]any) {
		into["records"] = into["records"].(float64) + 1
		for _, k := range []string{"wall_s", "cpu_s", "billed_cpu_s", "billed_ram_gib_s"} {
			v, _ := rec[k].(float64)
			into[k] = into[k].(float64) + v
		}
	}
	zero := func() map[string]any {
		return map[string]any{"records": 0.0, "wall_s": 0.0, "cpu_s": 0.0, "billed_cpu_s": 0.0, "billed_ram_gib_s": 0.0}
	}
	totals := zero()
	totals["sandboxes"] = 0
	var rows []map[string]any
	for _, id := range f.order {
		sb := f.sandboxes[id]
		if (q.Get("sandbox_id") != "" && q.Get("sandbox_id") != id) || !f.labelsMatch(sb, q["label"]) {
			continue
		}
		row := zero()
		for _, rec := range f.records {
			if rec["sandbox_id"] == id {
				sum(row, rec)
				sum(totals, rec)
			}
		}
		if row["records"].(float64) == 0 {
			continue
		}
		totals["sandboxes"] = totals["sandboxes"].(int) + 1
		row["sandbox_id"], row["labels"], row["status"] = id, sb["labels"], sb["status"]
		row["cpu_avg"] = row["cpu_s"].(float64) / row["wall_s"].(float64)
		rows = append(rows, row)
	}
	totals["cpu_avg"] = totals["cpu_s"].(float64) / totals["wall_s"].(float64)
	page := f.page(r, rows, func(map[string]any) bool { return true })
	reply(w, 200, map[string]any{"totals": totals, "sandboxes": page["items"], "next_cursor": page["next_cursor"]})
}

// egressSummary serves GET /v1/sandboxes/egress/summary: a sandbox's egress log
// counted.
func (f *fake) egressSummary(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("sandbox_id")
	if f.sandboxes[id] == nil {
		apiError(w, 404, "not_found", "no such sandbox")
		return
	}
	out := map[string]any{"sandbox_id": id, "requests": 0, "allowed": 0, "denied": 0, "error": 0, "policy": 0,
		"bytes_up": 0, "bytes_down": 0}
	hosts, creds := map[string]map[string]any{}, map[string]map[string]any{}
	var hostOrder, credOrder []string
	inc := func(m map[string]any, k string, n int) { m[k] = m[k].(int) + n }
	for _, e := range f.egressLog {
		if e["sandbox_id"] != id {
			continue
		}
		d := e["decision"].(string)
		if d == "policy" {
			inc(out, "policy", 1)
			continue
		}
		up, _ := e["bytes_up"].(int)
		down, _ := e["bytes_down"].(int)
		inc(out, "requests", 1)
		inc(out, d, 1)
		inc(out, "bytes_up", up)
		inc(out, "bytes_down", down)
		host := e["host"].(string)
		if hosts[host] == nil {
			hosts[host] = map[string]any{"host": host, "requests": 0, "allowed": 0, "denied": 0, "error": 0, "bytes_up": 0, "bytes_down": 0}
			hostOrder = append(hostOrder, host)
		}
		inc(hosts[host], "requests", 1)
		inc(hosts[host], d, 1)
		inc(hosts[host], "bytes_up", up)
		inc(hosts[host], "bytes_down", down)
		if cred, _ := e["credential"].(string); cred != "" {
			if creds[cred] == nil {
				creds[cred] = map[string]any{"credential": cred, "requests": 0, "allowed": 0, "denied": 0, "error": 0}
				credOrder = append(credOrder, cred)
			}
			inc(creds[cred], "requests", 1)
			inc(creds[cred], d, 1)
		}
	}
	byHost, byCred := []any{}, []any{}
	for _, h := range hostOrder {
		byHost = append(byHost, hosts[h])
	}
	for _, c := range credOrder {
		byCred = append(byCred, creds[c])
	}
	out["by_host"], out["by_credential"] = byHost, byCred
	reply(w, 200, out)
}

func (f *fake) bucketSecrets(w http.ResponseWriter, r *http.Request, rest []string) {
	agent := r.URL.Query().Get("agent")
	switch {
	case r.Method == http.MethodGet && len(rest) == 0:
		var items []map[string]any
		for _, s := range f.secrets {
			info := map[string]any{}
			for k, v := range s {
				if k != "value" {
					info[k] = v
				}
			}
			items = append(items, info)
		}
		slices.SortFunc(items, func(a, b map[string]any) int {
			return strings.Compare(fmt.Sprint(a["agent"], a["name"]), fmt.Sprint(b["agent"], b["name"]))
		})
		reply(w, 200, map[string]any{"items": items})
	case r.Method == http.MethodPut && len(rest) == 1:
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, named := body["name"]
		_, header := body["header"]
		_, user := body["username"]
		headerSecret := body["type"] == "header" && header && !user
		registrySecret := body["type"] == "registry" && user && !header
		if named || (!headerSecret && !registrySecret) {
			apiError(w, 400, "invalid_request", "unexpected body")
			return
		}
		body["name"] = rest[0]
		if agent != "" {
			body["agent"] = agent
		}
		f.secrets[agent+"/"+rest[0]] = body
		w.WriteHeader(204)
	case r.Method == http.MethodDelete && len(rest) == 1:
		if _, ok := f.secrets[agent+"/"+rest[0]]; !ok {
			apiError(w, 404, "not_found", "no such secret")
			return
		}
		delete(f.secrets, agent+"/"+rest[0])
		w.WriteHeader(204)
	default:
		apiError(w, 405, "method_not_allowed", r.Method)
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// bucketRoute names the bucket-wide routes under /v1/sandboxes/.
var bucketRoute = map[string]bool{"events": true, "webhooks": true, "usage": true, "limits": true, "egress": true}
