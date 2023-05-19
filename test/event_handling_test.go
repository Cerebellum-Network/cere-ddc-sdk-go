package test

import (
	"context"
	"testing"
	"time"

	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg"
	"github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/bucket"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/vedhavyas/go-subkey"
)

const URL = "ws://127.0.0.1:9944/ws"

func (a *ApplicationTestSuite) TestEventHandling() {
	// given
	t := a.T()
	var err error

	client := pkg.CreateBlockchainClient(URL)
	var contractAddress string
	t.Run("Deploy bucket contract", func(t *testing.T) {
		c, err := a.deployBucketContract(client)
		assert.NoError(t, err)
		contractAddress, err = subkey.SS58Address(c.ToBytes(), 54)
		assert.NoError(t, err)
	})

	buck := bucket.CreateDdcBucketContract(client, contractAddress)
	log.Infof("Contract: %s", buck.GetContractAddress())

	bucketCreatedChan := a.subscribeToBucketCreateUpdates(t, buck)
	bucketAvailabilityUpdatedChan := a.subscribeToBucketAvailabilityChangeUpdates(t, buck)
	err = client.SetEventDispatcher(contractAddress, buck.GetEventDispatcher())
	assert.NoError(t, err)

	var bucketId bucket.BucketId

	t.Run("Create bucket", func(t *testing.T) {
		_, err := a.bucketCreate(contractAddress, client, context.Background())
		assert.NoError(t, err)
		select {
		case event := <-bucketCreatedChan:
			bucketId = event.BucketId
		case <-time.After(time.Minute):
			log.Errorf("Timeout - bucket creation")
			t.FailNow()
		}
	})

	t.Run("Change bucket avaliability", func(t *testing.T) {
		_, err := a.bucketSetAvailability(contractAddress, client, context.Background(), bucketId, true)
		assert.NoError(t, err)

		log.Info("Waiting for availability update event")
		var availabilityEvent *bucket.BucketAvailabilityUpdatedEvent
		select {
		case availabilityEvent = <-bucketAvailabilityUpdatedChan:
		case <-time.After(time.Minute):
			log.Errorf("Timeout - bucket creation")
			t.FailNow()
		}
		assert.True(t, availabilityEvent.PublicAvailability)

		t.Run("Check availability change in the contract", func(t *testing.T) {
			b, err := buck.BucketGet(bucketId)
			assert.NoError(t, err)
			assert.Equal(t, true, b.Bucket.PublicAvailability)
		})
	})
}

func (a *ApplicationTestSuite) subscribeToBucketAvailabilityChangeUpdates(t *testing.T, buck bucket.DdcBucketContract) chan *bucket.BucketAvailabilityUpdatedEvent {
	bucketAvailabilityUpdatedChan := make(chan *bucket.BucketAvailabilityUpdatedEvent)
	err := buck.AddContractEventHandler(bucket.BucketAvailabilityUpdatedId, func(raw interface{}) {
		args := raw.(*bucket.BucketAvailabilityUpdatedEvent)
		log.Info("BucketAvailabilityUpdated args:", args)
		bucketAvailabilityUpdatedChan <- args
	})
	assert.NoError(t, err)
	return bucketAvailabilityUpdatedChan
}

func (a *ApplicationTestSuite) subscribeToBucketCreateUpdates(t *testing.T, buck bucket.DdcBucketContract) chan *bucket.BucketCreatedEvent {
	bucketCreatedChan := make(chan *bucket.BucketCreatedEvent)
	err := buck.AddContractEventHandler(bucket.BucketCreatedEventId, func(raw interface{}) {
		args := raw.(*bucket.BucketCreatedEvent)
		log.Info("BucketCreated args:", args)
		bucketCreatedChan <- args
	})
	assert.NoError(t, err)
	return bucketCreatedChan
}
