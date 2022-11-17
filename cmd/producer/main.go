package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/lovoo/goka"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
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

var (
	extraEvents   = make(ExtraEvents)
	correctEvents = make(CorrectEvents)
)

func main() {
	mainCtx, done := context.WithCancel(context.Background())

	midlog.LogCommandLine()
	config.ReadGlobal()

	blockstore.NewBlockStore(
		mainCtx, config.Global.BlockStore, "thorchain")

	sync.InitGlobalSync(mainCtx)
	s := sync.GlobalSync

	loadAllCorrections(extraEvents, correctEvents)

	go func() {
		inSync := false

		brokers := config.Global.Kafka.Brokers
		topic := config.Global.Kafka.BlockTopic

		for {

			lastHeight, lastIndex, err := GetLastHeight(brokers, topic)
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

			midlog.Info("Terminating chain sync routine")
		}
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
					EventIndex: kafka.EventIdx{
						Height: block.Height,
						Offset: idx,
					},
					BlockTimestamp: block.Time,
					Event:          &block.Results.BeginBlockEvents[i],
				}
				idx++
				out <- iEvent
			}

			for _, tx := range block.Results.TxsResults {
				for i := range tx.Events {
					iEvent := kafka.IndexedEvent{
						EventIndex: kafka.EventIdx{
							Height: block.Height,
							Offset: idx,
						},
						BlockTimestamp: block.Time,
						Event:          &tx.Events[i],
					}
					idx++
					out <- iEvent
				}
			}

			for i := range block.Results.EndBlockEvents {
				iEvent := kafka.IndexedEvent{
					EventIndex: kafka.EventIdx{
						Height: block.Height,
						Offset: idx,
					},
					BlockTimestamp: block.Time,
					Event:          &block.Results.EndBlockEvents[i],
				}
				idx++
				out <- iEvent
			}

			// Add any corrections that are supposed to be in this block
			if extras, ok := extraEvents[block.Height]; ok {
				for _, v := range extras {
					midlog.WarnF("Sending extra event: %v", v.Type)
					iEvent := kafka.IndexedEvent{
						EventIndex: kafka.EventIdx{
							Height: block.Height,
							Offset: idx,
						},
						BlockTimestamp: block.Time,
						Event:          v,
					}
					idx++
					out <- iEvent
				}
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
	defer func(eventEmitter *goka.Emitter) {
		ierr := eventEmitter.Finish()
		if ierr != nil {
			midlog.WarnF("Error trying to gracefully close emitter: %v", ierr)
		}
	}(eventEmitter)

	for {
		select {
		case <-ctx.Done():
			return
		case iEvent := <-in:
			if iEvent.EventIndex.Height == lastHeight && iEvent.EventIndex.Offset <= lastIndex {
				continue
			}

			if mainnetFilter(&iEvent) == record.Discard {
				iEvent.Event = nil
			}

			if corrections, ok := correctEvents[iEvent.EventIndex.Height]; ok {
				for _, correctFunc := range corrections {
					if correctFunc(&iEvent) == record.Discard {
						iEvent.Event = nil
					}
				}
			}

			keyS, err := iEvent.KeyAsString()
			if err != nil {
				midlog.ErrorE(err, "Error getting key to emit event message")
			}

			_, err = eventEmitter.Emit(keyS, iEvent)
			if err != nil {
				midlog.ErrorE(err, "Error trying to emit message")
			}
		}
	}
}

var consumer sarama.Consumer // replaceable in tests
func GetLastHeight(brokers []string, topic string) (int64, int16, error) {
	if consumer == nil {
		c, err := sarama.NewConsumer(brokers, sarama.NewConfig())
		if err != nil {
			return 0, 0, err
		}
		defer func(c sarama.Consumer) {
			ierr := c.Close()
			if ierr != nil {
				midlog.WarnF("Error trying to close kafka client %v", ierr)
			}
		}(c)

		consumer = c
	}

	hwmall := consumer.HighWaterMarks()
	hwm, ok := hwmall[topic]
	if !ok {
		return 0, 0, errors.New("unable to find topic in high water mark map")
	}

	var (
		partition int32
		high      int64
	)

	for k, v := range hwm {
		if v > high {
			partition = k
		}
	}

	pc, err := consumer.ConsumePartition(topic, partition, high)
	if err != nil {
		return 0, 0, err
	}

	var (
		lastHeight int64
		lastOffset int16
	)

	select {
	case msg := <-pc.Messages():
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
		lastHeight = ie.EventIndex.Height
		lastOffset = ie.EventIndex.Offset

	case <-time.After(3 * time.Second):
		return 0, 0, errors.New("timeout getting last message in topic")
	}

	return lastHeight, lastOffset, nil
}
