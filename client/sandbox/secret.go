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

// Secret types.
const (
	// SecretTypeHeader is a credential the egress proxy sets as a header.
	SecretTypeHeader = "header"
	// SecretTypeRegistry is a registry credential, used only to pull a
	// private image (CreateRequest.RegistryCredential).
	SecretTypeRegistry = "registry"
)

// Secret is a credential, held by the developer platform and never given
// back. It is of one of two types:
//
//   - header (the default): the egress proxy sets Header to Value on the
//     requests an egress rule naming it allows, and only to Hosts. The
//     sandbox never sees the value.
//   - registry (RegistrySecret): Username and Value (a password or token) to
//     pull a private image from the registry in Hosts, named by
//     CreateRequest.RegistryCredential. It is used on the node for the pull
//     only; an egress rule cannot name it.
//
// Printing a Secret with fmt never shows the value.
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
	// Type is SecretTypeHeader (the default when empty) or
	// SecretTypeRegistry.
	Type string
	// Header is the header the proxy sets (default "Authorization"); a
	// header secret's only.
	Header string
	// Username is a registry secret's user name. It is not secret: listing
	// shows it.
	Username string
	// Value is the header's value, or the registry password or token: at
	// most 8 KiB, one line.
	Value string
	// Hosts are the only hosts the value is ever sent to: exact names, or
	// "*.example.com" for exactly one more label. A registry secret's are
	// registry hosts ("ghcr.io", "docker.io" for Docker Hub). Required.
	Hosts []string
}

// BearerSecret is a secret that carries "Authorization: Bearer <token>" to
// hosts, for example one person's OAuth access token.
func BearerSecret(name, token string, hosts ...string) Secret {
	return Secret{Name: name, Header: "Authorization", Value: "Bearer " + strings.TrimSpace(token), Hosts: hosts}
}

// RegistrySecret is a registry credential for private images on hosts (for
// example "ghcr.io"): store it with Service.SetSecret or SetAgentSecret and
// name it in CreateRequest.RegistryCredential.
func RegistrySecret(name, username, password string, hosts ...string) Secret {
	return Secret{Name: name, Type: SecretTypeRegistry, Username: username, Value: password, Hosts: hosts}
}

// Placeholder is what a sandbox writes in a request header to name a
// credential, "cere:" + name: with a rule whose Placeholders list name, the
// egress proxy replaces the header with the secret's.
func Placeholder(name string) string { return PlaceholderPrefix + name }

// MarshalJSON writes the secret as the API takes it, value included.
func (s Secret) MarshalJSON() ([]byte, error) {
	b := s.body()
	b.Name = s.Name
	return json.Marshal(b)
}

type secretBody struct {
	Name     string   `json:"name,omitempty"`
	Type     string   `json:"type"`
	Header   string   `json:"header,omitempty"`
	Username string   `json:"username,omitempty"`
	Value    string   `json:"value"`
	Hosts    []string `json:"hosts"`
}

// body is the secret as PUT /v1/secrets/{name} takes it.
func (s Secret) body() secretBody {
	if s.typ() == SecretTypeRegistry {
		return secretBody{Type: SecretTypeRegistry, Username: s.Username, Value: s.Value, Hosts: s.Hosts}
	}
	return secretBody{Type: SecretTypeHeader, Header: s.header(), Value: s.Value, Hosts: s.Hosts}
}

// String never shows the value.
func (s Secret) String() string {
	if s.typ() == SecretTypeRegistry {
		return fmt.Sprintf("Secret(%s, registry, username=%s, hosts=%v)", s.Name, s.Username, s.Hosts)
	}
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

func (s Secret) typ() string {
	if s.Type == "" {
		return SecretTypeHeader
	}
	return s.Type
}

func (s Secret) validate() error {
	if !secretName.MatchString(s.Name) {
		return fmt.Errorf("sandbox: secret name %q: lower-case letters, digits and '-', at most 63 characters", s.Name)
	}
	switch s.typ() {
	case SecretTypeHeader:
		if s.Username != "" {
			return fmt.Errorf("sandbox: secret %s: a header secret has no username (a registry secret does)", s.Name)
		}
	case SecretTypeRegistry:
		if s.Header != "" {
			return fmt.Errorf("sandbox: secret %s: a registry secret has no header", s.Name)
		}
		if strings.TrimSpace(s.Username) == "" {
			return fmt.Errorf("sandbox: secret %s: a registry secret needs a username", s.Name)
		}
	default:
		return fmt.Errorf("sandbox: secret %s: type %q: want %q or %q", s.Name, s.Type, SecretTypeHeader, SecretTypeRegistry)
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
	Name string `json:"name"`
	// Type is SecretTypeHeader or SecretTypeRegistry.
	Type string `json:"type"`
	// Header is a header secret's header.
	Header string `json:"header,omitempty"`
	// Username is a registry secret's user name.
	Username string   `json:"username,omitempty"`
	Hosts    []string `json:"hosts"`
	// Agent is set on an agent's secret.
	Agent     string    `json:"agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SetSecret stores (or replaces) a secret of the bucket: any sandbox of the
// bucket whose rule names it may use it, or, for a registry secret, whose
// create names it in RegistryCredential. Storing again replaces the value and
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
		JSON: secret.body(),
	}, nil)
}

// ListSecrets lists the bucket's and its agents' secrets, never values (a
// registry secret's username is shown). A sandbox's own secrets are on the
// sandbox (Sandbox.Secrets).
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
	c, err := sb.api()
	if err != nil {
		return err
	}
	return c.Do(ctx, &client.Request{Method: http.MethodPut, Path: sb.path("/secrets/" + url.PathEscape(name)),
		JSON: map[string]string{"value": value}}, nil)
}
