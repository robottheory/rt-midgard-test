package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/julienschmidt/httprouter"
	log "github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"golang.org/x/net/websocket"
)

////////////////////////////////////////////////////////////////////////////////////////
// Config
////////////////////////////////////////////////////////////////////////////////////////

const (
	// MaxEventBuffer is the maximum number of events we will buffer for the client
	// before we abort and close the connection.
	MaxEventBuffer = 128
)

////////////////////////////////////////////////////////////////////////////////////////
// Init
////////////////////////////////////////////////////////////////////////////////////////

// Init must be called to start background subscriptions required for websocket queries.
func Init() {
	log.Info().Msg("initializing websockets...")

	// initialize price feed
	priceFeed = NewPriceFeed()
	go priceFeed.Run()
}

////////////////////////////////////////////////////////////////////////////////////////
// Events
////////////////////////////////////////////////////////////////////////////////////////

type ErrorEvent struct {
	Error string `json:"error"`
}

type PriceEvent struct {
	Asset     string  `json:"asset"`
	RunePrice float64 `json:"runePrice"`
}

////////////////////////////////////////////////////////////////////////////////////////
// Handler
////////////////////////////////////////////////////////////////////////////////////////

var httpHandler http.Handler = websocket.Handler(handler)

// Handler is the httprouter.Handle function that wraps the underlying websocket.
func Handler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	httpHandler.ServeHTTP(w, r)
}

func handler(ws *websocket.Conn) {
	log.Info().Msg("websocket connected")

	// setup
	dec := json.NewDecoder(ws)
	enc := json.NewEncoder(ws)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// close the websocket on context cancel
	go func() {
		<-ctx.Done()
		ws.Close()
	}()

	// ensure we cancel
	defer cancel()

	for {
		// read the query
		q := &Query{
			enc:    enc,
			ctx:    ctx,
			cancel: cancel,
		}
		err := dec.Decode(&q)
		switch {
		case err == io.EOF:
			return
		case err != nil:
			log.Info().Err(err).Msg("bad query")
			enc.Encode(ErrorEvent{err.Error()}) // nolint:errcheck
			return
		}

		// handle the query
		err = q.handle()
		if err != nil {
			log.Debug().Err(err).Msg("query error")
			enc.Encode(ErrorEvent{err.Error()}) // nolint:errcheck
			return
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// Query
////////////////////////////////////////////////////////////////////////////////////////

const (
	MethodSubscribePrice = "subscribePrice"
)

type Query struct {
	mu     sync.Mutex    // lock for writes in case of multiple subscriptions
	enc    *json.Encoder // encoder for websocket responses
	ctx    context.Context
	cancel func()

	Method string   `json:"method"`
	Assets []string `json:"assets"`
}

func (q *Query) handle() error {
	switch q.Method {
	case MethodSubscribePrice:
		return q.subscribePrice()
	default:
		return errors.New("unknown method")
	}
}

func (q *Query) send(event interface{}) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.enc.Encode(event)
}

// ------------------------------ handlers ------------------------------

func (q *Query) subscribePrice() error {
	if len(q.Assets) == 0 {
		return errors.New("must provide list of assets")
	}

	// register all assets
	sub := &Subscriber{q.ctx, q.cancel, make(chan *PriceEvent, MaxEventBuffer)}
	for _, asset := range q.Assets {
		err := priceFeed.Subscribe(asset, sub)
		if err != nil {
			return err
		}
	}

	// start routine to stream encoded events back to the client
	go func() {
		for {
			select {
			case e := <-sub.events:
				err := q.send(e)
				if err != nil {
					log.Debug().Err(err).Msg("error sending websocket event, closing connection")
					q.cancel()
					return
				}
			case <-q.ctx.Done():
				return
			}
		}
	}()

	return nil
}

////////////////////////////////////////////////////////////////////////////////////////
// Subscriber
////////////////////////////////////////////////////////////////////////////////////////

type Subscriber struct {
	ctx    context.Context
	cancel func()
	events chan *PriceEvent
}

////////////////////////////////////////////////////////////////////////////////////////
// PriceFeed
////////////////////////////////////////////////////////////////////////////////////////

var priceFeed *PriceFeed

type PriceFeed struct {
	sync.Mutex
	subscriptions map[string]([]*Subscriber)
	latest        map[string]timeseries.PoolDepths
	initialized   chan struct{}
}

func NewPriceFeed() *PriceFeed {
	return &PriceFeed{
		subscriptions: make(map[string]([]*Subscriber)),
		latest:        make(map[string]timeseries.PoolDepths),
		initialized:   make(chan struct{}),
	}
}

func (p *PriceFeed) Run() {
	once := sync.Once{}

	for {
		<-*db.WebsocketNotify
		state := timeseries.Latest.GetState()

		p.Lock()
		for asset, info := range state.Pools {
			// skip if there was no change
			if lastInfo, ok := p.latest[asset]; ok && lastInfo == info {
				continue
			}
			p.latest[asset] = info

			// send to all subscriptions
			e := &PriceEvent{
				Asset:     asset,
				RunePrice: info.AssetPrice(),
			}
			removeSubscribers := []int{}
			for i, sub := range p.subscriptions[asset] {
				select {
				case <-sub.ctx.Done():
				case sub.events <- e:
					continue
				default:
				}
				// close the connection and mark subscriber for removal if buffer is full
				sub.cancel()
				removeSubscribers = append(removeSubscribers, i)
			}

			// remove all subscribers we shutdown, gc will cleanup the channels
			for i, j := range removeSubscribers {
				copy(p.subscriptions[asset][j-i:], p.subscriptions[asset[j-i+1:]])
			}
			p.subscriptions[asset] = p.subscriptions[asset][:len(p.subscriptions[asset])-len(removeSubscribers)]
		}
		p.Unlock()

		// unblock calls to Subscribe
		once.Do(func() { close(p.initialized) })
	}
}

func (p *PriceFeed) Subscribe(asset string, sub *Subscriber) error {
	// wait for the one block so we can verify the requested asset exists
	<-p.initialized

	// ensure the asset is valid
	if _, ok := p.latest[asset]; !ok {
		return fmt.Errorf("unknown asset: %s", asset)
	}

	// add subscription
	p.Lock()
	defer p.Unlock()
	p.subscriptions[asset] = append(p.subscriptions[asset], sub)

	// send the latest price for the asset if available
	if info, ok := p.latest[asset]; ok {
		e := &PriceEvent{
			Asset:     asset,
			RunePrice: info.AssetPrice(),
		}
		select {
		case sub.events <- e:
		default:
		}
	}

	return nil
}
