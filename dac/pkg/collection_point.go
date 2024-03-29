package dac

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/cerebellum-network/cere-ddc-sdk-go/core/pkg/crypto"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
		SessionId          string `json:"sessionId"`
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
	fulfillmentPath = "/fulfillment"
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

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		d.url.String()+fulfillmentPath,
		bytes.NewBuffer(json),
	)

	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if response, err := d.httpClient.Do(req); err != nil {
		return err
	} else if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("DAC collection point post: %d %s", response.StatusCode, string(body))
	}

	return nil
}

func SignFulfillment(fulfillment *Fulfillment, scheme crypto.Scheme) error {
	signature, err := scheme.Sign([]byte(fulfillment.Cid + fulfillment.SessionId + fulfillment.RequestId + strconv.FormatUint(fulfillment.FulfilledTimestamp, 10)))
	if err != nil {
		return err
	}
	fulfillment.WorkerSignature = signature
	return nil
}
