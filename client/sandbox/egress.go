package sandbox

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
)

// Allow is one egress rule: the sandbox may call Host (exact, or "*.suffix")
// with Methods (upper case, or "*"; TRACE, TRACK and CONNECT never), and with
// PathPrefix only under that path (matched on whole segments: "/repos/org/app"
// admits "/repos/org/app/..." but not "/repos/org/app-private"). A rule names
// credentials in one of two ways:
//
//   - Credential: header mode. The egress proxy sets that secret's header on
//     every request the rule allows.
//   - Placeholders: placeholder mode. The sandbox names one per request by
//     writing Placeholder(name) ("cere:<name>") in a header, for example
//     "Authorization: Bearer cere:slack-team-a", and the proxy replaces that
//     header with the secret's. This tells two accounts on one host apart.
//
// A name resolves to the sandbox's own secret, else its agent's, else the
// bucket's; the secret must be bound to the rule's host.
type Allow struct {
	Host         string   `json:"host"`
	Methods      []string `json:"methods"`
	PathPrefix   string   `json:"path_prefix,omitempty"`
	Credential   string   `json:"credential,omitempty"`
	Placeholders []string `json:"placeholders,omitempty"`
}

// ParseAllow reads a rule as the `cere` CLI writes it, and as the CLI reads
// it: "host[:METHODS[:path_prefix[:credential]]]": "pypi.org:GET:/simple/",
// "api.github.com:GET,HEAD::github-readonly". Methods are comma-separated and
// default to GET. A credential part of placeholders,
// "api.example.com:POST:/openai/v1:cere:ddc-inference" or
// "mcp.slack.com:*::cere:slack-a,cere:slack-b", is Placeholders (placeholder
// mode), never a Credential.
func ParseAllow(rule string) (Allow, error) {
	parts := strings.SplitN(rule, ":", 4)
	a := Allow{Host: strings.TrimSpace(parts[0]), Methods: []string{http.MethodGet}}
	if a.Host == "" {
		return Allow{}, fmt.Errorf("sandbox: egress rule %q: no host", rule)
	}
	if len(parts) > 1 && parts[1] != "" {
		a.Methods = nil
		for _, m := range strings.Split(parts[1], ",") {
			if m = strings.ToUpper(strings.TrimSpace(m)); m != "" {
				a.Methods = append(a.Methods, m)
			}
		}
		if len(a.Methods) == 0 {
			return Allow{}, fmt.Errorf("sandbox: egress rule %q: no methods", rule)
		}
	}
	if len(parts) > 2 && parts[2] != "" {
		if !strings.HasPrefix(parts[2], "/") {
			return Allow{}, fmt.Errorf("sandbox: egress rule %q: the path prefix must start with /", rule)
		}
		a.PathPrefix = parts[2]
	}
	if len(parts) > 3 {
		if !strings.HasPrefix(parts[3], PlaceholderPrefix) {
			a.Credential = parts[3]
			return a, nil
		}
		for _, p := range strings.Split(parts[3], ",") {
			name, ok := strings.CutPrefix(strings.TrimSpace(p), PlaceholderPrefix)
			if !ok || name == "" {
				return Allow{}, fmt.Errorf("sandbox: egress rule %q: placeholders are written cere:<name>,cere:<name>", rule)
			}
			a.Placeholders = append(a.Placeholders, name)
		}
	}
	return a, nil
}

// MustParseAllow is ParseAllow for rules written in the program; it panics on
// a malformed rule.
func MustParseAllow(rule string) Allow {
	a, err := ParseAllow(rule)
	if err != nil {
		panic(err)
	}
	return a
}

// normalize returns the rule with upper-case methods, GET when it names none.
func (a Allow) normalize() Allow {
	out := Allow{Host: a.Host, PathPrefix: a.PathPrefix, Credential: a.Credential,
		Placeholders: slices.Clone(a.Placeholders)}
	for _, m := range a.Methods {
		out.Methods = append(out.Methods, strings.ToUpper(m))
	}
	if len(out.Methods) == 0 {
		out.Methods = []string{http.MethodGet}
	}
	return out
}

func (a Allow) validate() error {
	if strings.TrimSpace(a.Host) == "" {
		return errors.New("sandbox: an egress rule needs a host")
	}
	if strings.ContainsAny(a.Host, "/: @") {
		return fmt.Errorf("sandbox: egress rule host %q: give a host name such as api.github.com, not a URL", a.Host)
	}
	if a.Credential != "" && len(a.Placeholders) > 0 {
		return fmt.Errorf("sandbox: egress rule for %s: a rule has a credential or placeholders, not both", a.Host)
	}
	return nil
}

// Credentials is every secret name the rule uses, either way.
func (a Allow) Credentials() []string {
	var out []string
	if a.Credential != "" {
		out = append(out, a.Credential)
	}
	return append(out, a.Placeholders...)
}

// sameTarget reports whether two rules are for the same host, path prefix and
// credentials, so that their methods can be merged.
func sameTarget(x, y Allow) bool {
	px, py := slices.Clone(x.Placeholders), slices.Clone(y.Placeholders)
	slices.Sort(px)
	slices.Sort(py)
	return strings.EqualFold(x.Host, y.Host) && x.PathPrefix == y.PathPrefix && x.Credential == y.Credential &&
		slices.Equal(px, py)
}

func unionMethods(x, y []string) []string {
	if slices.Contains(x, "*") || slices.Contains(y, "*") {
		return []string{"*"}
	}
	out := slices.Clone(x)
	for _, m := range y {
		if !slices.Contains(out, m) {
			out = append(out, m)
		}
	}
	return out
}

// mergeAllow is current with add: a rule for the same host, path prefix and
// credentials as one already there adds its methods to it; any other is put
// first, in the order given. The same as `cere sandbox egress allow`.
func mergeAllow(current, add []Allow) []Allow {
	out := make([]Allow, len(current))
	for i, a := range current {
		out[i] = a.normalize()
	}
	var fresh []Allow
	for _, r := range add {
		r = r.normalize()
		merged := false
		for _, group := range [][]Allow{out, fresh} {
			for i := range group {
				if sameTarget(group[i], r) {
					group[i].Methods = unionMethods(group[i].Methods, r.Methods)
					merged = true
					break
				}
			}
			if merged {
				break
			}
		}
		if !merged {
			fresh = append(fresh, r)
		}
	}
	return append(fresh, out...)
}

// removeAllow is current without host's rules or, given methods, without
// those methods of them (a rule left with none goes); and how many rules
// changed. The same as `cere sandbox egress remove`.
func removeAllow(current []Allow, host string, methods []string) ([]Allow, int, error) {
	ms := make([]string, len(methods))
	for i, m := range methods {
		ms[i] = strings.ToUpper(m)
	}
	out := []Allow{}
	changed := 0
	for _, a := range current {
		if !strings.EqualFold(a.Host, host) {
			out = append(out, a)
			continue
		}
		if len(ms) == 0 || slices.Contains(ms, "*") {
			changed++
			continue
		}
		if slices.Contains(a.Methods, "*") {
			return nil, 0, fmt.Errorf("sandbox: the rule for %s allows every method (*): remove the host without "+
				"methods, then allow the methods it may use", a.Host)
		}
		keep := slices.DeleteFunc(slices.Clone(a.Methods), func(m string) bool { return slices.Contains(ms, m) })
		if len(keep) == len(a.Methods) {
			out = append(out, a)
			continue
		}
		changed++
		if len(keep) > 0 {
			a.Methods = keep
			out = append(out, a)
		}
	}
	return out, changed, nil
}

// EgressRules is the sandbox's egress rules as the API reports them now; it
// refreshes the sandbox.
func (sb *Sandbox) EgressRules(ctx context.Context) ([]Allow, error) {
	if err := sb.Refresh(ctx); err != nil {
		return nil, err
	}
	return slices.Clone(sb.Egress.Allow), nil
}

// SetEgress replaces all of the sandbox's egress rules; an empty list removes
// all access. The egress proxy applies them from the sandbox's next request,
// also inside a connection it already has open. It returns the new rules. A
// sandbox that has ended answers CodeNotRunning; a rule whose credential does
// not resolve, or is not bound to its host, CodeUnknownCredential or
// CodeCredentialNotBound, as at create.
func (sb *Sandbox) SetEgress(ctx context.Context, rules []Allow) ([]Allow, error) {
	body := struct {
		Allow []Allow `json:"allow"`
	}{Allow: []Allow{}}
	for _, r := range rules {
		if err := r.validate(); err != nil {
			return nil, err
		}
		body.Allow = append(body.Allow, r.normalize())
	}
	var out Sandbox
	err := sb.svc.c.Do(ctx, &client.Request{Method: http.MethodPut, Path: sb.path("/egress"), JSON: body}, &out)
	if err != nil {
		return nil, err
	}
	sb.update(&out)
	return slices.Clone(sb.Egress.Allow), nil
}

// Allow adds rules to the sandbox's egress: a rule for the same host, path
// prefix and credentials as one it has adds its methods to that rule; any
// other is put first. It reads the rules and replaces them, so a change made
// elsewhere in between is overwritten. It returns the new rules.
func (sb *Sandbox) Allow(ctx context.Context, rules ...Allow) ([]Allow, error) {
	if len(rules) == 0 {
		return nil, errors.New("sandbox: Allow needs at least one rule")
	}
	for _, r := range rules {
		if err := r.validate(); err != nil {
			return nil, err
		}
	}
	current, err := sb.currentRules(ctx)
	if err != nil {
		return nil, err
	}
	return sb.SetEgress(ctx, mergeAllow(current, rules))
}

// RemoveEgress removes host's rules or, given methods, only those methods of
// them (a rule left with none goes). It returns the new rules, and an error
// when no rule matches.
func (sb *Sandbox) RemoveEgress(ctx context.Context, host string, methods ...string) ([]Allow, error) {
	current, err := sb.currentRules(ctx)
	if err != nil {
		return nil, err
	}
	rules, changed, err := removeAllow(current, host, methods)
	if err != nil {
		return nil, err
	}
	if changed == 0 {
		if len(methods) > 0 {
			return nil, fmt.Errorf("sandbox: no rule for %s allows %s", host, strings.Join(methods, ","))
		}
		return nil, fmt.Errorf("sandbox: no rule for %s", host)
	}
	return sb.SetEgress(ctx, rules)
}

func (sb *Sandbox) currentRules(ctx context.Context) ([]Allow, error) {
	rules, err := sb.EgressRules(ctx)
	if err != nil {
		return nil, err
	}
	if sb.Ended() {
		return nil, &client.Error{Status: http.StatusConflict, Code: CodeNotRunning,
			Message: fmt.Sprintf("sandbox %s is %s: its egress rules no longer apply", sb.ID, sb.Status)}
	}
	return rules, nil
}

// EgressLogQuery selects egress log entries; every field is optional.
type EgressLogQuery struct {
	SandboxID string
	// Since and Until bound the entries' time.
	Since, Until time.Time
	// Decision is one of the Decision constants.
	Decision string
	// Limit is the most entries returned: 0 is 100, a negative value all.
	Limit int
}

// EgressLog is every request the bucket's sandboxes made through the egress
// proxy, allowed or refused, and every change of their rules, newest first.
func (s *Service) EgressLog(ctx context.Context, q EgressLogQuery) ([]EgressEntry, error) {
	v := url.Values{}
	if q.SandboxID != "" {
		v.Set("sandbox_id", q.SandboxID)
	}
	if !q.Since.IsZero() {
		v.Set("since", rfc3339(q.Since))
	}
	if !q.Until.IsZero() {
		v.Set("until", rfc3339(q.Until))
	}
	if q.Decision != "" {
		v.Set("decision", q.Decision)
	}
	return client.List[EgressEntry](ctx, s.c,
		client.Request{Method: http.MethodGet, Path: "/v1/egress/log", Query: v}, limit(q.Limit))
}

// EgressCounts counts egress log entries: Requests is Allowed + Denied (by
// the egress proxy) + Errors.
type EgressCounts struct {
	Requests int64 `json:"requests"`
	Allowed  int64 `json:"allowed"`
	Denied   int64 `json:"denied"`
	Errors   int64 `json:"error"`
}

// EgressSummary counts a sandbox's egress log: its requests in all, per host
// and per credential (a request that used two counts for each), and Policy,
// its rule changes, which are not requests.
type EgressSummary struct {
	SandboxID string `json:"sandbox_id"`
	EgressCounts
	Policy       int64              `json:"policy"`
	BytesUp      int64              `json:"bytes_up"`
	BytesDown    int64              `json:"bytes_down"`
	ByHost       []HostEgress       `json:"by_host"`
	ByCredential []CredentialEgress `json:"by_credential"`
}

// HostEgress is a sandbox's requests to one host.
type HostEgress struct {
	Host string `json:"host"`
	EgressCounts
	BytesUp   int64 `json:"bytes_up"`
	BytesDown int64 `json:"bytes_down"`
}

// CredentialEgress is a sandbox's requests that used one credential.
type CredentialEgress struct {
	Credential string `json:"credential"`
	EgressCounts
}

// EgressSummary counts a sandbox's egress log between since and until
// (either zero for no bound). A sandbox of another bucket answers
// client.CodeNotFound.
func (s *Service) EgressSummary(ctx context.Context, sandboxID string, since, until time.Time) (*EgressSummary, error) {
	if sandboxID == "" {
		return nil, errors.New("sandbox: EgressSummary needs a sandbox ID")
	}
	v := url.Values{"sandbox_id": {sandboxID}}
	if !since.IsZero() {
		v.Set("since", rfc3339(since))
	}
	if !until.IsZero() {
		v.Set("until", rfc3339(until))
	}
	var out EgressSummary
	if err := s.c.Do(ctx, &client.Request{Method: http.MethodGet, Path: "/v1/egress/summary", Query: v}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func rfc3339(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// limit maps a query's Limit to client.List's: 0 is 100, negative is all.
func limit(n int) int {
	switch {
	case n == 0:
		return 100
	case n < 0:
		return 0
	}
	return n
}
