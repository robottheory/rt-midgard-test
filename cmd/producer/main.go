package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/lovoo/goka"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/fetch/sync"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/blockstore"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/chain"
	"gitlab.com/thorchain/midgard/internal/util/kafka"
	"gitlab.com/thorchain/midgard/internal/util/midlog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "gitlab.com/thorchain/midgard/internal/globalinit"
)

func main() {
	mainCtx, done := context.WithCancel(context.Background())

	midlog.LogCommandLine()
	config.ReadGlobal()

	blockstore.NewBlockStore(
		mainCtx, config.Global.BlockStore, "thorchain")

	sync.InitGlobalSync(mainCtx)
	s := sync.GlobalSync

	go func() {
		inSync := false

		for {

			lastHeight, lastIndex, err := GetLastHeight()
			if err != nil {
				midlog.FatalE(err, "Unable to get last stored height")
			}

			if !inSync {
				midlog.InfoF("Resuming sync from: block %v, offset %v", lastHeight, lastIndex)
			}

			blocks := make(chan chain.Block)
			events := make(chan kafka.IndexedEvent)

			loopCtx, loopCancel := context.WithCancel(mainCtx)

			go processEvents(loopCtx, events, lastHeight, lastIndex)
			go processBlocks(loopCtx, blocks, events)

			// The docs for this function mention errquit and errnodata but these don't seem to exist
			_, inSync, err = s.CatchUp(blocks, lastHeight)
			if err != nil {
				midlog.WarnF("Resuming sync due to error: %v", err)
			}

			loopCancel()

			select {
			case <-mainCtx.Done():
				break
			default:
				// Give it a few secs to drain the kafka queues
				time.Sleep(time.Second * 5)
			}
		}

		midlog.Info("Terminating chain sync routine")

	}()

	sigs := make(chan os.Signal)
	go func() {
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	}()

	<-sigs
	done()

	midlog.Info("Signal received, exiting")
}

func processBlocks(ctx context.Context, in chan chain.Block, out chan kafka.IndexedEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case block := <-in:
			idx := int16(0)

			for i := range block.Results.BeginBlockEvents {
				iEvent := kafka.IndexedEvent{
					Height: block.Height,
					Offset: idx,
					Event:  &block.Results.BeginBlockEvents[i],
				}
				idx++
				out <- iEvent
			}

			for _, tx := range block.Results.TxsResults {
				for i := range tx.Events {
					iEvent := kafka.IndexedEvent{
						Height: block.Height,
						Offset: idx,
						Event:  &tx.Events[i],
					}
					idx++
					out <- iEvent
				}
			}

			for i := range block.Results.EndBlockEvents {
				iEvent := kafka.IndexedEvent{
					Height: block.Height,
					Offset: idx,
					Event:  &block.Results.EndBlockEvents[i],
				}
				idx++
				out <- iEvent
			}

		}
	}
}

func processEvents(ctx context.Context, in chan kafka.IndexedEvent, lastHeight int64, lastIndex int16) {
	brokers := config.Global.Kafka.Brokers
	topic := config.Global.Kafka.BlockTopic

	eventEmitter, err := goka.NewEmitter(brokers, goka.Stream(topic), new(kafka.IndexedEventCodec))
	if err != nil {
		midlog.FatalE(err, "Failed to create event emitter")
	}
	defer eventEmitter.Finish()

	for {
		select {
		case <-ctx.Done():
			return
		case iEvent := <-in:
			if iEvent.Height == lastHeight && iEvent.Offset <= lastIndex {
				continue
			}

			keyB, err := iEvent.KeyAsBinary()
			if err != nil {
				midlog.ErrorE(err, "Error getting key to emit event message")
			}

			keyS := base64.StdEncoding.EncodeToString(keyB[:])
			eventEmitter.Emit(keyS, iEvent)
		}
	}
}

func GetLastHeight() (int64, int16, error) {
	brokers := config.Global.Kafka.Brokers
	topic := config.Global.Kafka.BlockTopic

	client, err := sarama.NewClient(brokers, sarama.NewConfig())
	defer client.Close()

	p, _ := client.Partitions(topic)

	lastPartition := int32(0)
	lastMessageOffset := int64(0)
	for _, v := range p {
		offset, err := client.GetOffset(topic, v, sarama.OffsetNewest)
		if err != nil {
			return 0, 0, err
		}

		if offset > lastMessageOffset {
			lastPartition = v
			lastMessageOffset = offset
		}
	}

	// Handle completely empty topic
	if lastMessageOffset == 0 {
		return 1, 0, nil
	}

	consumer, err := sarama.NewConsumer(brokers, sarama.NewConfig())
	if err != nil {
		return 0, 0, err
	}
	defer consumer.Close()

	partitionConsumer, err := consumer.ConsumePartition(topic, lastPartition, lastMessageOffset-1)
	if err != nil {
		return 0, 0, err
	}
	defer partitionConsumer.Close()

	var (
		lastHeight int64
		lastOffset int16
	)

	select {
	case msg := <-partitionConsumer.Messages():
		val := msg.Value
		decoder := kafka.IndexedEventCodec{}
		ieD, err := decoder.Decode(val)
		if err != nil {
			return 0, 0, err
		}

		if _, isEvent := ieD.(kafka.IndexedEvent); !isEvent {
			return 0, 0, errors.New(fmt.Sprintf("message should be type kafka.IndexedEvent, got %T", ieD))
		}

		ie := ieD.(kafka.IndexedEvent)
		lastHeight = ie.Height
		lastOffset = ie.Offset

	case <-time.After(3 * time.Second):
		return 0, 0, errors.New("timeout getting last message in topic")
	}

	return lastHeight, lastOffset, nil
}
