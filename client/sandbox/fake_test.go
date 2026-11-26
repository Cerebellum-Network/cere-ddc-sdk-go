package sandbox_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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

	// What the tests check.
	creates    []map[string]any // create bodies as sent
	execs      []string         // exec bodies as sent
	tokens     []string         // the token of each request, in order
	createFail []int            // statuses to answer the next creates with
	startAs    string           // the status a create answers (default running)
	failStart  bool             // a created sandbox fails to start
}

func newFake(t *testing.T) (*fake, *sandbox.Service) {
	f := &fake{t: t, sandboxes: map[string]map[string]any{}, files: map[string]map[string][]byte{},
		secrets: map[string]map[string]any{}, runSecret: map[string]map[string]string{}, startAs: sandbox.StatusRunning}
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
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/"), "/")
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
	case parts[0] == "usage" && len(parts) == 2:
		reply(w, 200, f.page(r, f.records, func(rec map[string]any) bool {
			id, _ := rec["sandbox_id"].(string)
			return (r.URL.Query().Get("sandbox_id") == "" || r.URL.Query().Get("sandbox_id") == id) &&
				f.labelsMatch(f.sandboxes[id], r.URL.Query()["label"])
		}))
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
	if len(f.createFail) > 0 {
		status := f.createFail[0]
		f.createFail = f.createFail[1:]
		apiError(w, status, map[int]string{503: "no_capacity", 409: "limit_active_sandboxes", 502: "node_unreachable"}[status], "refused")
		return
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
	sb := map[string]any{
		"id": id, "status": f.startAs, "bucket": 4116, "node": strings.Repeat("ab", 32), "runtime": body["runtime"],
		"image": body["image"], "command": body["command"], "workdir": "/workspace", "resources": res,
		"timeout_s": body["timeout_s"], "labels": body["labels"], "egress": egress, "secrets": secrets,
		"created_at": time.Now().UTC().Format(time.RFC3339), "exit_reason": nil, "exit_code": nil, "retained": []string{},
		"usage": map[string]any{"wall_s": 0, "billing_basis": ""},
	}
	if f.failStart {
		sb["status"] = sandbox.StatusStarting
	}
	f.sandboxes[id] = sb
	f.order = append([]string{id}, f.order...)
	f.files[id] = map[string][]byte{}
	f.runSecret[id] = values
	reply(w, 201, sb)
}

func (f *fake) labelsMatch(sb map[string]any, filters []string) bool {
	labels, _ := sb["labels"].(map[string]any)
	for _, flt := range filters {
		k, v, _ := strings.Cut(flt, ":")
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
				sb["status"], sb["exit_reason"] = "failed", "failed: image pull no such image"
			} else {
				sb["status"] = "running"
			}
		}
		reply(w, 200, sb)
	case route == "DELETE ":
		if !ended(sb) {
			reason := "terminated by API"
			if q := r.URL.Query().Get("reason"); q != "" {
				reason += ": " + q
			}
			var kept []string
			if q := r.URL.Query().Get("retain"); q != "" {
				kept = strings.Split(q, ",")
			}
			sb["status"], sb["exit_reason"], sb["retained"] = "terminated", reason, kept
			sb["usage"] = map[string]any{"wall_s": 12.5, "billing_basis": "request_or_usage", "billed_cpu_s": 3.2}
			f.records = append(f.records, map[string]any{"record_id": "r-" + id, "sandbox_id": id, "final": true,
				"billed_cpu_s": 3.2, "billed_ram_gib_s": 1.5, "cpu_s": 0.5, "memory_limit": 1024, "dac_status": "pushed"})
		}
		reply(w, 200, sb)
	case ended(sb) && route == "GET files":
		path := r.URL.Query().Get("path")
		if !slices.Contains(sb["retained"].([]string), path) {
			apiError(w, 410, "file_gone", "not retained")
			return
		}
		_, _ = w.Write(f.files[id][path])
	case ended(sb):
		apiError(w, 409, "not_running", "the sandbox has ended")
	case route == "POST exec":
		var req struct {
			Command  []string `json:"command"`
			Stdin    *string  `json:"stdin"`
			TimeoutS int      `json:"timeout_s"`
		}
		raw, _ := io.ReadAll(r.Body)
		f.execs = append(f.execs, string(raw))
		_ = json.Unmarshal(raw, &req)
		out := strings.Join(req.Command, " ")
		if req.Stdin != nil {
			out += " <" + *req.Stdin
		}
		if r.URL.Query().Get("stream") == "1" {
			events(w, [2]string{"stdout", `{"data":"out:"}`}, [2]string{"stderr", `{"data":"err\n"}`},
				[2]string{"stdout", mustJSON(map[string]string{"data": out + "\n"})},
				[2]string{"exit", fmt.Sprintf(`{"exit_code":%d,"duration_ms":5,"timed_out":false}`, req.TimeoutS)})
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
		if _, named := body["name"]; named || body["type"] != "header" {
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
