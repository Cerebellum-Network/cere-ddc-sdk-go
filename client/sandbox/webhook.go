package sandbox

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
)

// Webhook limits.
const (
	// MaxWebhooks is how many webhooks a bucket may have; one more answers
	// CodeLimitWebhooks.
	MaxWebhooks = 10
	// DefaultWebhookTolerance is how far a delivery's timestamp may be from
	// now for VerifyWebhook, unless it is given another tolerance.
	DefaultWebhookTolerance = 5 * time.Minute
)

// The headers of a webhook delivery.
const (
	// HeaderWebhookID is the event's ID: dedupe and order deliveries on it.
	HeaderWebhookID = "webhook-id"
	// HeaderWebhookTimestamp is when the delivery was signed, in Unix seconds.
	HeaderWebhookTimestamp = "webhook-timestamp"
	// HeaderWebhookSignature is "v1,<base64 HMAC-SHA256>" of
	// "<webhook-id>.<webhook-timestamp>.<body>", keyed with the secret.
	HeaderWebhookSignature = "webhook-signature"
)

// States of a webhook delivery.
const (
	DeliveryDelivered = "delivered"
	DeliveryRetrying  = "retrying"
	DeliveryFailed    = "failed"
)

// WebhookRequest is a webhook to create: the bucket's events (as
// Service.Events reads them) that happen after it is created and match its
// filters are POSTed to URL, one event per request, signed with Secret.
// Printing a WebhookRequest never shows the secret.
type WebhookRequest struct {
	// URL is https://, at most 2048 bytes, and its host must not be an
	// address in a private, loopback, link-local or shared range.
	URL string
	// Types are the event types sent (the Event constants); all when empty.
	Types []string
	// Label, "key:value", sends only the events of sandboxes with that
	// label; every sandbox's when empty.
	Label string
	// Secret (16 to 256 characters) signs each delivery: check it with
	// VerifyWebhook. It is never returned.
	Secret string
}

// String never shows the secret.
func (r WebhookRequest) String() string {
	return fmt.Sprintf("WebhookRequest(%s, types=%v, label=%q)", r.URL, r.Types, r.Label)
}

// GoString never shows the secret.
func (r WebhookRequest) GoString() string { return r.String() }

func (r WebhookRequest) validate() error {
	u, err := url.Parse(r.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("sandbox: webhook URL %q: want https://host/path", r.URL)
	}
	if n := len(r.Secret); n < 16 || n > 256 {
		return errors.New("sandbox: a webhook secret is 16 to 256 characters")
	}
	if r.Label != "" && !strings.Contains(r.Label, ":") {
		return fmt.Errorf("sandbox: webhook label %q: want key:value", r.Label)
	}
	return nil
}

// Webhook is a webhook of the bucket; its secret is never returned.
type Webhook struct {
	ID    string   `json:"id"`
	URL   string   `json:"url"`
	Types []string `json:"types"`
	// Label is "key:value", or empty for every sandbox.
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
}

// WebhookDelivery is one attempt to deliver an event to a webhook. A 2xx
// answer within 10 s is a delivery; anything else (another status, a
// timeout, a refused connection, a redirect) is retried after 30 s, 2 min,
// 10 min, 30 min, 1 h, 2 h, 4 h, 8 h and 12 h, then the delivery has
// failed.
type WebhookDelivery struct {
	EventID string    `json:"event_id"`
	Type    string    `json:"type"`
	Attempt int       `json:"attempt"`
	At      time.Time `json:"at"`
	// StatusCode is what the URL answered; 0 when it answered nothing.
	StatusCode int    `json:"status_code"`
	Error      string `json:"error"`
	// State is DeliveryDelivered, DeliveryRetrying or DeliveryFailed.
	State string `json:"state"`
	// NextAt is when a retrying delivery is tried next.
	NextAt *time.Time `json:"next_at"`
}

// CreateWebhook creates a webhook of the bucket. A bucket has at most
// MaxWebhooks (CodeLimitWebhooks).
func (s *Service) CreateWebhook(ctx context.Context, req WebhookRequest) (*Webhook, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	body := struct {
		URL    string   `json:"url"`
		Types  []string `json:"types,omitempty"`
		Label  string   `json:"label,omitempty"`
		Secret string   `json:"secret"`
	}{URL: req.URL, Types: req.Types, Label: req.Label, Secret: req.Secret}
	var out Webhook
	if err := s.c.Do(ctx, &client.Request{Method: http.MethodPost, Path: "/v1/webhooks", JSON: body}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListWebhooks lists the bucket's webhooks.
func (s *Service) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	var page client.Page[Webhook]
	if err := s.c.Do(ctx, &client.Request{Method: http.MethodGet, Path: "/v1/webhooks"}, &page); err != nil {
		return nil, err
	}
	return page.Items, nil
}

// DeleteWebhook deletes a webhook; an unknown ID answers client.CodeNotFound.
func (s *Service) DeleteWebhook(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("sandbox: DeleteWebhook needs an ID")
	}
	return s.c.Do(ctx, &client.Request{Method: http.MethodDelete, Path: "/v1/webhooks/" + url.PathEscape(id)}, nil)
}

// WebhookDeliveries lists a webhook's latest delivery attempts, newest
// first: at most limit (0 for the API's default).
func (s *Service) WebhookDeliveries(ctx context.Context, id string, limit int) ([]WebhookDelivery, error) {
	if id == "" {
		return nil, errors.New("sandbox: WebhookDeliveries needs an ID")
	}
	var q url.Values
	if limit > 0 {
		q = url.Values{"limit": {strconv.Itoa(limit)}}
	}
	var page client.Page[WebhookDelivery]
	if err := s.c.Do(ctx, &client.Request{Method: http.MethodGet,
		Path: "/v1/webhooks/" + url.PathEscape(id) + "/deliveries", Query: q}, &page); err != nil {
		return nil, err
	}
	return page.Items, nil
}

// VerifyWebhook checks a webhook delivery, in the handler that receives it,
// before its body is trusted: the webhook-signature header must hold a
// "v1,<base64>" HMAC-SHA256 of "<webhook-id>.<webhook-timestamp>.<body>"
// keyed with secret's bytes, and webhook-timestamp must be within tolerance
// of now (0 for DefaultWebhookTolerance), which turns a replayed old
// delivery away. body is the request's body as received; decode it into an
// Event once it verifies. Several space-separated signatures may be sent;
// one that matches is enough.
func VerifyWebhook(secret string, header http.Header, body []byte, tolerance time.Duration) error {
	if secret == "" {
		return errors.New("sandbox: webhook: no secret to verify with")
	}
	id, ts, sigs := header.Get(HeaderWebhookID), header.Get(HeaderWebhookTimestamp), header.Get(HeaderWebhookSignature)
	if id == "" || ts == "" || sigs == "" {
		return errors.New("sandbox: webhook: the webhook-id, webhook-timestamp and webhook-signature headers are required")
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return fmt.Errorf("sandbox: webhook: webhook-timestamp %q is not Unix seconds", ts)
	}
	if tolerance <= 0 {
		tolerance = DefaultWebhookTolerance
	}
	if d := time.Since(time.Unix(sec, 0)); d > tolerance || d < -tolerance {
		return fmt.Errorf("sandbox: webhook: signed %s from now, more than the %s allowed", d.Round(time.Second), tolerance)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(id + "." + ts + "."))
	mac.Write(body)
	want := mac.Sum(nil)
	for _, sig := range strings.Fields(sigs) {
		version, b64, ok := strings.Cut(sig, ",")
		if !ok || version != "v1" {
			continue
		}
		if got, err := base64.StdEncoding.DecodeString(b64); err == nil && hmac.Equal(got, want) {
			return nil
		}
	}
	return errors.New("sandbox: webhook: no signature matches")
}
