package sandbox_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
	"github.com/cerebellum-network/cere-ddc-sdk-go/client/sandbox"
)

func TestWebhooks(t *testing.T) {
	f, sbx := newFake(t)
	req := sandbox.WebhookRequest{URL: "https://hooks.example.com/ddc", Types: []string{sandbox.EventSandboxExited},
		Label: "task:t-81", Secret: "a-secret-of-24-chars-xyz"}
	wh, err := sbx.CreateWebhook(ctx, req)
	if err != nil || !strings.HasPrefix(wh.ID, "wh-") || wh.Label != "task:t-81" || wh.Types[0] != sandbox.EventSandboxExited || wh.CreatedAt.IsZero() {
		t.Fatalf("%+v %v", wh, err)
	}
	if got := mustJSON(f.hookBodies[0]); got != `{"label":"task:t-81","secret":"a-secret-of-24-chars-xyz","types":["sandbox.exited"],"url":"https://hooks.example.com/ddc"}` {
		t.Errorf("body %s", got)
	}
	// Every type, every sandbox: the filters are left out.
	all, err := sbx.CreateWebhook(ctx, sandbox.WebhookRequest{URL: "https://hooks.example.com/all", Secret: "0123456789abcdef"})
	if err != nil || all.Label != "" || len(all.Types) != 0 {
		t.Fatalf("%+v %v", all, err)
	}
	if got := mustJSON(f.hookBodies[1]); got != `{"secret":"0123456789abcdef","url":"https://hooks.example.com/all"}` {
		t.Errorf("body %s", got)
	}
	list, err := sbx.ListWebhooks(ctx)
	if err != nil || len(list) != 2 || list[0].ID != wh.ID {
		t.Fatalf("%+v %v", list, err)
	}
	for i := 2; i < sandbox.MaxWebhooks; i++ {
		if _, err := sbx.CreateWebhook(ctx, req); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := sbx.CreateWebhook(ctx, req); !client.IsCode(err, sandbox.CodeLimitWebhooks) {
		t.Errorf("the 11th: %v", err)
	}

	f.delivered[wh.ID] = []map[string]any{
		{"event_id": "18342", "type": "sandbox.exited", "attempt": 2, "at": "2026-11-26T10:05:31Z", "status_code": nil,
			"error": "connection refused", "state": "retrying", "next_at": "2026-11-26T10:07:31Z"},
		{"event_id": "18342", "type": "sandbox.exited", "attempt": 1, "at": "2026-11-26T10:05:01Z", "status_code": 500,
			"error": nil, "state": "retrying", "next_at": nil},
	}
	ds, err := sbx.WebhookDeliveries(ctx, wh.ID, 1)
	if err != nil || len(ds) != 1 || ds[0].State != sandbox.DeliveryRetrying || ds[0].StatusCode != 0 || ds[0].NextAt == nil || ds[0].Attempt != 2 {
		t.Fatalf("%+v %v", ds, err)
	}
	if _, err := sbx.WebhookDeliveries(ctx, "wh-nope", 0); !client.IsCode(err, client.CodeNotFound) {
		t.Errorf("deliveries of an unknown webhook: %v", err)
	}
	if err := sbx.DeleteWebhook(ctx, wh.ID); err != nil {
		t.Fatal(err)
	}
	if err := sbx.DeleteWebhook(ctx, wh.ID); !client.IsCode(err, client.CodeNotFound) {
		t.Errorf("delete twice: %v", err)
	}

	n := len(f.hookBodies)
	for name, bad := range map[string]sandbox.WebhookRequest{
		"http":         {URL: "http://hooks.example.com", Secret: "0123456789abcdef"},
		"no host":      {URL: "https:///x", Secret: "0123456789abcdef"},
		"short secret": {URL: "https://hooks.example.com", Secret: "short"},
		"label":        {URL: "https://hooks.example.com", Secret: "0123456789abcdef", Label: "t-81"},
	} {
		if _, err := sbx.CreateWebhook(ctx, bad); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	if len(f.hookBodies) != n {
		t.Errorf("%d refused webhooks reached the API", len(f.hookBodies)-n)
	}
	if out := fmt.Sprintf("%v %+v %#v", req, req, req); strings.Contains(out, req.Secret) {
		t.Errorf("printed: %s", out)
	}
}

// The vector was computed outside this SDK:
//
//	printf '%s' "18342.1764151500.$body" | openssl dgst -sha256 -hmac 'test-secret-0123456789' -binary | base64
const (
	vectorSecret = "test-secret-0123456789"
	vectorID     = "18342"
	vectorTS     = "1764151500" // 2025-11-26T10:05:00Z
	vectorBody   = `{"id":"18342","type":"sandbox.exited","sandbox_id":"sb-01jdx3q4w8m2v6r9t0kz5c7b1n","labels":{"task":"t-81"},"exit_kind":"main_exited","exit_code":0}`
	vectorSig    = "v1,4Ez3WysU4Uid6OcdztSO89wRdzlacYU36iHKwj3LiOs="
)

func webhookHeader(id, ts, sig string) http.Header {
	h := http.Header{}
	h.Set("webhook-id", id)
	h.Set("webhook-timestamp", ts)
	h.Set("webhook-signature", sig)
	return h
}

func TestVerifyWebhookVector(t *testing.T) {
	// The vector's timestamp is a year old: only a wide tolerance takes it.
	decade := 10 * 365 * 24 * time.Hour
	if err := sandbox.VerifyWebhook(vectorSecret, webhookHeader(vectorID, vectorTS, vectorSig), []byte(vectorBody), decade); err != nil {
		t.Fatalf("the vector: %v", err)
	}
	// Another signature first (a secret being rotated, a later version).
	if err := sandbox.VerifyWebhook(vectorSecret, webhookHeader(vectorID, vectorTS, "v2,abc v1,AAAA "+vectorSig), []byte(vectorBody), decade); err != nil {
		t.Fatalf("among several: %v", err)
	}
	for name, tc := range map[string]struct {
		secret, id, ts, sig, body string
		tolerance                 time.Duration
	}{
		"too old":          {vectorSecret, vectorID, vectorTS, vectorSig, vectorBody, 0},
		"another body":     {vectorSecret, vectorID, vectorTS, vectorSig, strings.Replace(vectorBody, `"exit_code":0`, `"exit_code":1`, 1), decade},
		"another id":       {vectorSecret, "18343", vectorTS, vectorSig, vectorBody, decade},
		"another secret":   {"test-secret-0123456780", vectorID, vectorTS, vectorSig, vectorBody, decade},
		"no v1 prefix":     {vectorSecret, vectorID, vectorTS, strings.TrimPrefix(vectorSig, "v1,"), vectorBody, decade},
		"v2 only":          {vectorSecret, vectorID, vectorTS, "v2," + strings.TrimPrefix(vectorSig, "v1,"), vectorBody, decade},
		"no signature":     {vectorSecret, vectorID, vectorTS, "", vectorBody, decade},
		"not Unix seconds": {vectorSecret, vectorID, "2025-11-26T10:05:00Z", vectorSig, vectorBody, decade},
		"no secret":        {"", vectorID, vectorTS, vectorSig, vectorBody, decade},
	} {
		if err := sandbox.VerifyWebhook(tc.secret, webhookHeader(tc.id, tc.ts, tc.sig), []byte(tc.body), tc.tolerance); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestVerifyWebhookTolerance(t *testing.T) {
	sign := func(at time.Time) http.Header {
		ts := strconv.FormatInt(at.Unix(), 10)
		mac := hmac.New(sha256.New, []byte(vectorSecret))
		mac.Write([]byte(vectorID + "." + ts + "." + vectorBody))
		return webhookHeader(vectorID, ts, "v1,"+base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	}
	now := time.Now()
	for _, tc := range []struct {
		at        time.Time
		tolerance time.Duration
		ok        bool
	}{
		{now, 0, true},
		{now.Add(-4 * time.Minute), 0, true},
		{now.Add(-6 * time.Minute), 0, false},
		{now.Add(6 * time.Minute), 0, false},
		{now.Add(-6 * time.Minute), 10 * time.Minute, true},
	} {
		err := sandbox.VerifyWebhook(vectorSecret, sign(tc.at), []byte(vectorBody), tc.tolerance)
		if (err == nil) != tc.ok {
			t.Errorf("signed %s ago, tolerance %s: %v", now.Sub(tc.at).Round(time.Second), tc.tolerance, err)
		}
	}
}
