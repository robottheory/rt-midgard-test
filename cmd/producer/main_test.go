package main

import (
	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"testing"
)

func TestGetLastHeight2(t *testing.T) {
	sc := mocks.NewConsumer(t, nil)
	sc.exp
	b := initMockBroker(t, "testgroup", "test.topic")
	lh, lo, err := GetLastHeight2([]string{b.Addr()}, "test.topic")

	if err != nil {
		t.Errorf("Failed to get last height: %v", err)
	}

	t.Logf("Got %v.%v", lh, lo)
}

func initMockBroker(t *testing.T, group, topic string) *sarama.MockBroker {
	mockBroker := sarama.NewMockBroker(t, 0)
	//mockMetadataResponse := sarama.NewMockMetadataResponse(t).
	//	SetBroker(mockBroker.Addr(), mockBroker.BrokerID()).
	//	SetLeader("test.topic", 0, mockBroker.BrokerID()).
	//	SetController(mockBroker.BrokerID())
	//mockProducerResponse := sarama.NewMockProduceResponse(t).SetError("test.topic", 0, sarama.ErrNoError)
	//mockOffsetResponse := sarama.NewMockOffsetResponse(t).
	//	SetOffset("test.topic", 0, sarama.OffsetOldest, 0).
	//	SetOffset("test.topic", 0, sarama.OffsetNewest, 1).SetVersion(1)
	//mockFetchResponse := sarama.NewMockFetchResponse(t, 1).
	//	SetMessage("test.topic", 0, 0, sarama.StringEncoder("testing 123")).
	//	SetMessage("test.topic", 0, 1, sarama.StringEncoder("testing 123")).
	//	SetMessage("test.topic", 0, 2, sarama.StringEncoder("testing 123")).
	//	SetMessage("test.topic", 0, 3, sarama.StringEncoder("testing 123")).
	//	SetMessage("test.topic", 0, 4, sarama.StringEncoder("testing 123")).
	//	SetMessage("test.topic", 0, 5, sarama.StringEncoder("testing 123")).SetVersion(11)
	//mockCoordinatorResponse := sarama.NewMockFindCoordinatorResponse(t).SetCoordinator(sarama.CoordinatorType(0), group, mockBroker)
	//mockJoinGroupResponse := sarama.NewMockJoinGroupResponse(t)
	//mockSyncGroupResponse := sarama.NewMockSyncGroupResponse(t).
	//	SetMemberAssignment(&sarama.ConsumerGroupMemberAssignment{
	//		Version:  0,
	//		Topics:   map[string][]int32{"test.topic": {0}},
	//		UserData: nil,
	//	})
	//mockHeartbeatResponse := sarama.NewMockHeartbeatResponse(t)
	//mockOffsetFetchResponse := sarama.NewMockOffsetFetchResponse(t).
	//	SetOffset(group, "test.topic", 0, 0, "", sarama.KError(0))
	//mockApiVersionsResponse := sarama.NewMockApiVersionsResponse(t)
	//mockOffsetCommitResponse := sarama.NewMockOffsetCommitResponse(t)
	//
	//mockBroker.SetHandlerByMap(map[string]sarama.MockResponse{
	//	"MetadataRequest":        mockMetadataResponse,
	//	"ProduceRequest":         mockProducerResponse,
	//	"OffsetRequest":          mockOffsetResponse,
	//	"OffsetFetchRequest":     mockOffsetFetchResponse,
	//	"FetchRequest":           mockFetchResponse,
	//	"FindCoordinatorRequest": mockCoordinatorResponse,
	//	"JoinGroupRequest":       mockJoinGroupResponse,
	//	"SyncGroupRequest":       mockSyncGroupResponse,
	//	"HeartbeatRequest":       mockHeartbeatResponse,
	//	"ApiVersionsRequest":     mockApiVersionsResponse,
	//	"OffsetCommitRequest":    mockOffsetCommitResponse,
	//})

	testMsg := sarama.StringEncoder("Foo")
	mockFetchResponse := sarama.NewMockFetchResponse(t, 1)
	for i := int64(0); i < 10; i++ {
		mockFetchResponse.SetMessage(topic, 0, i, testMsg)
	}

	mockBroker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(mockBroker.Addr(), mockBroker.BrokerID()).
			SetLeader(topic, 0, mockBroker.BrokerID()),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetOffset(topic, 0, sarama.OffsetOldest, 0).
			SetOffset(topic, 0, sarama.OffsetNewest, 10),
		"FetchRequest": mockFetchResponse,
		"OffsetFetchRequest": sarama.NewMockOffsetFetchResponse(t).
			SetOffset(group, topic, 0, 10, "", 0),
	})

	return mockBroker
}
