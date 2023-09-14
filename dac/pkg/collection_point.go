package dac

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type (
	CollectionPoint interface {
		SaveFulfillment(fulfillment Fulfillment) error
	}

	dacCollectionPoint struct {
		url        url.URL
		httpClient http.Client
	}

	Fulfillment struct {
		SessionId          []byte `json:"sessionId"`
		RequestId          string `json:"requestId"`
		Cid                string `json:"cid"`
		OpCode             uint8  `json:"opCode"`
		BytesSent          uint32 `json:"bytesSent"`
		FulfilledTimestamp uint64 `json:"fulfilledTimestamp"`
		WorkerSignature    []byte `json:"workerSignature"`
		WorkerAddress      string `json:"workerAddress"`
	}
)

const (
	dacTimeout      = 10 * time.Second
	fulfillmentPath = "fulfillment"
)

func CreateCollectionPoint(url url.URL, httpClient http.Client) CollectionPoint {
	return &dacCollectionPoint{
		url,
		httpClient,
	}
}

func (d dacCollectionPoint) SaveFulfillment(fulfillment Fulfillment) error {
	json, err := json.Marshal(fulfillment)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), dacTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", d.url.JoinPath(fulfillmentPath).String(), bytes.NewBuffer(json))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if _, err := d.httpClient.Do(req); err != nil {
		return fmt.Errorf("DAC collection point put: %w", err)
	}

	return nil
}
