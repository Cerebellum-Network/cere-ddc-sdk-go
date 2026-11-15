package agentgateway

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvokeRequest_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	req := InvokeRequest{
		Bundle: BundleRef{
			Cluster:  "dragon-1",
			BucketID: "bafybucket123",
			CID:      "bafycid",
		},
		Handler: "onMessage",
		Event:   json.RawMessage(`{"text":"hi"}`),
		Context: json.RawMessage(`{"endpoints":{"cubby":"http://o/cubby"}}`),
		Labels:  map[string]string{"vaultId": "v1", "jobId": "j1"},
		Limits:  &InvokeLimits{TimeoutMs: 60000, CPUSeconds: 60, MemoryBytes: 268435456},
	}

	b, err := json.Marshal(req)
	require.NoError(t, err, "marshal")

	var out InvokeRequest
	require.NoError(t, json.Unmarshal(b, &out), "unmarshal")
	assert.Equal(t, "bafycid", out.Bundle.CID, "bundle.cid not preserved")
	assert.Equal(t, "onMessage", out.Handler, "handler not preserved")
	assert.Equal(t, "v1", out.Labels["vaultId"], "labels[vaultId] not preserved")
	require.NotNil(t, out.Limits, "limits not preserved")
	assert.Equal(t, 60000, out.Limits.TimeoutMs, "limits.TimeoutMs not preserved")
}

func TestInvokeRequest_OmitsEmptyFields(t *testing.T) {
	t.Parallel()
	req := InvokeRequest{
		Bundle: BundleRef{
			BucketID: "b",
			CID:      "c",
		},
		Handler: "h",
	}
	b, err := json.Marshal(req)
	require.NoError(t, err, "marshal")
	s := string(b)
	for _, omitted := range []string{`"cluster"`, `"event"`, `"context"`, `"labels"`, `"limits"`} {
		assert.NotContains(t, s, omitted, "expected field to be omitted")
	}
	// Required fields should be present.
	assert.Contains(t, s, `"bucketId":"b"`, "bucketId missing")
	assert.Contains(t, s, `"cid":"c"`, "cid missing")
	assert.Contains(t, s, `"handler":"h"`, "handler missing")
}

func TestInvokeResponse_SuccessShape(t *testing.T) {
	t.Parallel()
	resp := InvokeResponse{
		Result: json.RawMessage(`{"reply":"ok"}`),
		Status: "success",
		Metrics: &InvocationMetrics{
			CPUTimeMs:       150,
			MemoryPeakBytes: 45678901,
			WallClockMs:     234,
		},
	}
	b, err := json.Marshal(resp)
	require.NoError(t, err, "marshal")
	s := string(b)
	assert.Contains(t, s, `"status":"success"`, "status missing")
	assert.NotContains(t, s, `"error"`, "error should be omitted on success")
}

func TestInvokeResponse_ErrorShape(t *testing.T) {
	t.Parallel()
	resp := InvokeResponse{
		Status: "error",
		Error: &InvokeError{
			Code:      "HANDLER_NOT_FOUND",
			Message:   "no such handler",
			Retryable: false,
		},
	}
	b, err := json.Marshal(resp)
	require.NoError(t, err, "marshal")
	s := string(b)
	for _, want := range []string{
		`"status":"error"`,
		`"code":"HANDLER_NOT_FOUND"`,
		`"message":"no such handler"`,
		`"retryable":false`,
	} {
		assert.Contains(t, s, want, "missing field in error response")
	}
}
