package main

import (
	"encoding/base64"
	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetLastHeight(t *testing.T) {
	// Block 305 is the first one with events. The b64 below is the first event in that block, which is this:
	// {EventIndex:{Height:305 Offset:0} BlockTimestamp:2021-04-10 13:02:17.911198133 +0000 UTC Event:type:"message" attributes:<key:"action" value:"set_network_fee" index:true > }
	testEventB64 := "AEn/gQMBAQxJbmRleGVkRXZlbnQB/4IAAQMBCkV2ZW50SW5kZXgB/4QAAQ5CbG9ja1RpbWVzdGFtcAH/hgABBUV2ZW50Af+IAAAALP+DAwEBCEV2ZW50SWR4Af+EAAECAQZIZWlnaHQBBAABBk9mZnNldAEEAAAAEP+FBQEBBFRpbWUB/4YAAAAs/4cDAQEFRXZlbnQB/4gAAQIBBFR5cGUBDAABCkF0dHJpYnV0ZXMB/4wAAAAl/4sCAQEWW110eXBlcy5FdmVudEF0dHJpYnV0ZQH/jAAB/4oAADj/iQMBAQ5FdmVudEF0dHJpYnV0ZQH/igABAwEDS2V5AQoAAQVWYWx1ZQEKAAEFSW5kZXgBAgAAAEP/ggEB/gJiAAEPAQAAAA7YA5jZNk/Htf//AQEHbWVzc2FnZQEBAQZhY3Rpb24BD3NldF9uZXR3b3JrX2ZlZQEBAAAA"
	testEvent, _ := base64.StdEncoding.DecodeString(testEventB64)

	testConsumer := mocks.NewConsumer(t, mocks.NewTestConfig())
	testConsumer.ExpectConsumePartition("test.topic", 0, 0).
		YieldMessage(&sarama.ConsumerMessage{Value: testEvent})

	// Overrides the consumer in main with our test consumer
	consumer = testConsumer
	//b := initMockBroker(t, "testgroup", "test.topic")
	b := sarama.NewMockBroker(t, 0)
	lh, lo, err := GetLastHeight([]string{b.Addr()}, "test.topic")

	assert.NoError(t, err, "Failed to get last height: %v", err)
	assert.Equal(t, int64(305), lh)
	assert.Equal(t, int16(0), lo)
}
