package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
)

// MaxSecretValueBytes is the longest secret value.
const MaxSecretValueBytes = 8 << 10

// PlaceholderPrefix starts a credential placeholder in a request header.
const PlaceholderPrefix = "cere:"

var secretName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// Secret is a credential the egress proxy adds to a sandbox's requests: the
// proxy sets Header to Value on the requests an egress rule naming it allows,
// and only to Hosts. The value goes in and never comes out, and the sandbox
// never sees it. Printing a Secret with fmt never shows the value.
//
// A secret belongs to one of three scopes, looked up narrowest first:
//
//   - the sandbox's own: CreateRequest.Secrets; held for that sandbox only,
//     deleted when it ends; Sandbox.SetSecret replaces a value while it runs
//     (for example an OAuth access token its owner refreshed);
//   - an agent's: Service.SetAgentSecret; used only by sandboxes labelled
//     cere.agent=<agent>;
//   - the bucket's: Service.SetSecret; any sandbox of the bucket whose rule
//     names it.
type Secret struct {
	// Name matches ^[a-z0-9][a-z0-9-]{0,62}$.
	Name string
	// Header is the header the proxy sets (default "Authorization").
	Header string
	// Value is the header's value, at most 8 KiB, one line.
	Value string
	// Hosts are the only hosts the value is ever sent to: exact names, or
	// "*.example.com" for exactly one more label. Required.
	Hosts []string
}

// BearerSecret is a secret that carries "Authorization: Bearer <token>" to
// hosts, for example one person's OAuth access token.
func BearerSecret(name, token string, hosts ...string) Secret {
	return Secret{Name: name, Header: "Authorization", Value: "Bearer " + strings.TrimSpace(token), Hosts: hosts}
}

// Placeholder is what a sandbox writes in a request header to name a
// credential, "cere:" + name: with a rule whose Placeholders list name, the
// egress proxy replaces the header with the secret's.
func Placeholder(name string) string { return PlaceholderPrefix + name }

// MarshalJSON writes the secret as the API takes it, value included.
func (s Secret) MarshalJSON() ([]byte, error) {
	return json.Marshal(secretBody{Name: s.Name, Type: "header", Header: s.header(), Value: s.Value, Hosts: s.Hosts})
}

type secretBody struct {
	Name   string   `json:"name,omitempty"`
	Type   string   `json:"type"`
	Header string   `json:"header"`
	Value  string   `json:"value"`
	Hosts  []string `json:"hosts"`
}

// String never shows the value.
func (s Secret) String() string {
	return fmt.Sprintf("Secret(%s, header=%s, hosts=%v)", s.Name, s.header(), s.Hosts)
}

// GoString never shows the value.
func (s Secret) GoString() string { return s.String() }

func (s Secret) header() string {
	if s.Header == "" {
		return "Authorization"
	}
	return s.Header
}

func (s Secret) validate() error {
	if !secretName.MatchString(s.Name) {
		return fmt.Errorf("sandbox: secret name %q: lower-case letters, digits and '-', at most 63 characters", s.Name)
	}
	if err := checkValue(s.Value); err != nil {
		return fmt.Errorf("sandbox: secret %s: %w", s.Name, err)
	}
	if len(s.Hosts) == 0 {
		return fmt.Errorf("sandbox: secret %s: Hosts is required: the hosts the egress proxy may send this "+
			"value to, for example api.github.com; a secret with no hosts is sent nowhere", s.Name)
	}
	for _, h := range s.Hosts {
		if h = strings.TrimSpace(h); h == "" || strings.ContainsAny(h, "/:@ ") {
			return fmt.Errorf("sandbox: secret %s: host %q: give a host name such as api.github.com, not a URL",
				s.Name, h)
		}
	}
	return nil
}

func checkValue(v string) error {
	switch {
	case v == "":
		return errors.New("the value is empty")
	case len(v) > MaxSecretValueBytes:
		return fmt.Errorf("the value is longer than %d bytes", MaxSecretValueBytes)
	case strings.ContainsAny(v, "\r\n\x00"):
		return errors.New("the value must be a single line")
	}
	return nil
}

// SecretInfo describes a stored secret, never its value.
type SecretInfo struct {
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	Header string   `json:"header"`
	Hosts  []string `json:"hosts"`
	// Agent is set on an agent's secret.
	Agent     string    `json:"agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SetSecret stores (or replaces) a secret of the bucket: any sandbox of the
// bucket whose rule names it may use it. Storing again replaces the value and
// the hosts together.
func (s *Service) SetSecret(ctx context.Context, secret Secret) error {
	return s.putSecret(ctx, "", secret)
}

// SetAgentSecret stores (or replaces) a secret of one agent: only that
// agent's runs (sandboxes labelled cere.agent=<agent>) use it, before the
// bucket's secret of the same name.
func (s *Service) SetAgentSecret(ctx context.Context, agent string, secret Secret) error {
	if !secretName.MatchString(agent) {
		return fmt.Errorf("sandbox: agent name %q: lower-case letters, digits and '-', at most 63 characters", agent)
	}
	return s.putSecret(ctx, agent, secret)
}

func (s *Service) putSecret(ctx context.Context, agent string, secret Secret) error {
	if err := secret.validate(); err != nil {
		return err
	}
	return s.c.Do(ctx, &client.Request{
		Method: http.MethodPut, Path: "/v1/secrets/" + url.PathEscape(secret.Name), Query: agentQuery(agent),
		JSON: secretBody{Type: "header", Header: secret.header(), Value: secret.Value, Hosts: secret.Hosts},
	}, nil)
}

// ListSecrets lists the bucket's and its agents' secrets, never values. A
// sandbox's own secrets are on the sandbox (Sandbox.Secrets).
func (s *Service) ListSecrets(ctx context.Context) ([]SecretInfo, error) {
	var page client.Page[SecretInfo]
	if err := s.c.Do(ctx, &client.Request{Method: http.MethodGet, Path: "/v1/secrets"}, &page); err != nil {
		return nil, err
	}
	return page.Items, nil
}

// DeleteSecret deletes a secret of the bucket. A running sandbox whose rule
// names it gets 502 "credential missing" on that host from then on.
func (s *Service) DeleteSecret(ctx context.Context, name string) error {
	return s.deleteSecret(ctx, "", name)
}

// DeleteAgentSecret deletes a secret of one agent.
func (s *Service) DeleteAgentSecret(ctx context.Context, agent, name string) error {
	return s.deleteSecret(ctx, agent, name)
}

func (s *Service) deleteSecret(ctx context.Context, agent, name string) error {
	if name == "" {
		return errors.New("sandbox: DeleteSecret needs a name")
	}
	return s.c.Do(ctx, &client.Request{Method: http.MethodDelete, Path: "/v1/secrets/" + url.PathEscape(name),
		Query: agentQuery(agent)}, nil)
}

func agentQuery(agent string) url.Values {
	if agent == "" {
		return nil
	}
	return url.Values{"agent": {agent}}
}

// SetSecret replaces the value of one of the sandbox's own secrets while it
// runs, for example an OAuth access token its owner refreshed:
// sb.SetSecret(ctx, "notion", "Bearer "+newToken). The egress proxy uses it
// from the sandbox's next request. Its header and hosts stay as created. A
// name the sandbox was not created with answers CodeSecretNotFound, an ended
// sandbox CodeNotRunning.
func (sb *Sandbox) SetSecret(ctx context.Context, name, value string) error {
	if err := checkValue(value); err != nil {
		return fmt.Errorf("sandbox: secret %s: %w", name, err)
	}
	return sb.svc.c.Do(ctx, &client.Request{Method: http.MethodPut, Path: sb.path("/secrets/" + url.PathEscape(name)),
		JSON: map[string]string{"value": value}}, nil)
}
