package websockets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/tendermint/tendermint/libs/rand"
	"gitlab.com/thorchain/midgard/chain"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/util"

	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"sync"
	"syscall"
)

// TODO(acsaba): change everyh Warnf call to log maximum once every X min.
// TODO(acsaba): add counters next to every Warnf

// TODO(kano): Info logs is used on a lot of places. We should delete them, or perhaps we could
//   allow logging for debugging with loglevel or logmodule.

// TODO(kano): Rename to connectionPools.
// TODO(kano): consider making this a global variable like epoll.
type pools struct {
	// => // pools[$asset] => {$connName : new.Conn}
	connections map[string]map[string]net.Conn

	// TODO(kano): make it embeding (anonymous field)
	// allows concurrent read/read, but will be exclusive in the scenario on read/write or write/write
	mutex sync.RWMutex
}

var (
	epoller *epoll
	// TODO(kano): - extend logger to application name to prefix all output with service name => in this case websockets.
	logger = util.NewLogger()
)

// Entrypoint.
func Serve(listenPort int, connectionLimit int) {
	logger.Infof("Starting Websocket process for pool prices on %d with connection limit %d", listenPort, connectionLimit)
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		logger.Warnf("Can't get the rLimit %v", err)
		return
	}
	// TODO(acsaba): Let's discuss. How about setrlimit(rLimit.Max),
	// but actually throttling ourselves to limit. Or throttling only this single port?
	rLimit.Cur = uint64(connectionLimit)
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		logger.Warnf("Can't set the syscall limitr %v", err)
		return
	}

	// Start epoll
	var err error
	epoller, err = MkEpoll()
	if err != nil {
		logger.Warnf("Can't create the socket poller %v", err)
		return
	}

	// TODO(kano): rename to allConnections.
	instance := &pools{
		connections: map[string]map[string]net.Conn{},
	}

	go instance.read()

	// TODO(acsaba): register in api.go if possible
	http.HandleFunc("/v2/ws", wsHandler)
	// Listen for incoming connections ...
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", listenPort), nil); err != nil {
			logger.Warnf("error spinning up websocket connection listener %v", err)
		}
	}()

	for {
		<-*chain.WebsocketNotify
		// If more notifications happened, eat all future ones.
		hadMore := true
		for hadMore {
			select {
			case <-*chain.WebsocketNotify:
			default:
				hadMore = false
			}
		}

		logger.Info("Sending push price updates")

		// TODO(acsaba): populate me with real price info
		// maybe validate that our validatedPool has the asset.
		p := &Payload{
			Price: fmt.Sprintf("%f", rand.Float32()),
			Asset: "BTC.BTC",
		}
		instance.write(p)
	}
}

// ------------------------------- //
//   Pools Instance Methods
// ------------------------------- //

// TODO(kano): let's discuss. Is this reading messages or accepting connections or both?
//   let's change name or document after discussion.
func (p *pools) read() {
	for {
		// TODO(kano): rename to newConnections
		connections, err := epoller.Wait()
		if err != nil {
			// TODO(kano): Maybe have a number of retries but it it fails it might need to exit
			//   the websockets go routing
			logger.Warnf("Failed to epoll wait %v", err)
			time.Sleep(time.Second * 3)
			continue
		}

		for _, conn := range connections {

			if conn == nil {
				logger.Warnf("Nil connections in the pool !")
				// TODO(kano): should it be continue?
				break
			}

			msg, _, err := wsutil.ReadClientData(conn)
			if err != nil {
				logger.Warnf("Error reading from socket %v", err)
				// TODO(kano): let's discuss how closing channel looks from users point of view.
				//     do they get error messages?
				p.clearConnectionFromPools(conn)
				if err := epoller.Remove(conn); err != nil {
					logger.Warnf("Failed to remove %v", err)
				}
				conn.Close()
				continue
			}

			logger.Warnf("received msg \n %s", string(msg))

			i := &Instruction{}
			if err := json.Unmarshal(msg, i); err != nil {
				// TODO(acsaba): make
				logger.Warnf("Error marshaling message from %s \n closing connection", nameConn(conn))
				p.clearConnectionFromPools(conn)
				if err := epoller.Remove(conn); err != nil {
					logger.Warnf("Failed to remove %v", err)
				}
				conn.Close()
				continue
			}

			// TODO(acsaba): add metric for i.Assets
			logger.Infof("instruction received for assets %s", strings.Join(i.Assets, ","))
			pools := fetchValidatedPools()
			validPools := []string{}

			for _, a := range i.Assets {
				if !validateAsset(a, pools) {
					// TODO(kano): don't silently discard pool errors.
					//     Report to client and close connection.
					logger.Warnf("Invalid pool %s ignored", a)
					continue
				}
				validPools = append(validPools, a)
			}

			if len(validPools) == 0 {
				p.messageAndDisconnect("No valid assets were provided", conn)
				continue
			}

			if len(validPools) != len(i.Assets) {
				logger.Warnf("validPools len %d, incoming pools len %d", len(validPools), len(i.Assets))
				// TODO(kano): let's discuss. Is this is not a client error? If yes then we
				//     don't need to return error, just report error messag and close connection.
				//
				//  returning an error, but not sure where, it's just a go routine ...
				// perhaps we need a backchannel errors, back to main.
			}

			if i.Message == Connect {
				p.subscribeToPools(validPools, conn)
			} else if i.Message == Disconnect {
				// TODO(kano): let's discuss this. It's not clear to acsaba if
				//   Wait returns new connections or messages too. Let's clarify the working of
				//   unsubscribe and wait and then let's change the function names or add
				//   documentation.
				p.unsubscribeToPools(validPools, conn)
			} // TODO(kano): what if it's Message is neither? handle or document.
		}
	}
}

func (p *pools) write(update *Payload) {
	// Only write to connections concerned with update.Asset.
	p.mutex.RLock()
	pool, ok := p.connections[update.Asset]
	// TODO(kano): we need this mutex until we use pool.
	p.mutex.RUnlock()

	if !ok {
		logger.Infof("no pool in memory for %s, ignoring ...", update.Asset)
		return
	}

	for name, conn := range pool {
		// TODO(kano): what is 128? add a named constant.
		writer := wsutil.NewWriterSize(conn, ws.StateServerSide, ws.OpText, 128)
		// TODO(kano): initialize this before the loop, because it doesn't change.
		// TODO(kano): rename i. Maybe msg?
		i, err := json.Marshal(update)

		if err != nil {
			logger.Warnf("marshalling err on write %v", err)
			continue
		}

		logger.Infof("Try write %s", update.ToString())
		if _, err := io.Copy(writer, bytes.NewReader(i)); err != nil {
			logger.Infof("Failed to copy message to buffer %v", err)
			return
		}
		if err := writer.Flush(); err != nil {
			// TODO(kano): let's discuss. Does one iteration have to wait until the client
			//     accepts the message? What if the client is slow, will the other connections
			//     have to wait? Will this loop be as long as the sum of all ping times?

			// Just remove the connection from everything if we can't write to it.
			p.clearConnectionFromPools(conn)
			return
		}
		// TODO(kano): remove once we are happy with performance.
		logger.Infof("Write succeeded for asset %s on connection %s", update.Asset, name)
	}
}

func (p *pools) subscribeToPools(pools []string, conn net.Conn) {
	for _, a := range pools {
		// Read lock
		p.mutex.RLock()
		pool, okPool := p.connections[a]
		p.mutex.RUnlock()

		if !okPool {
			// Don't have a pool, create a new one, easy
			logger.Infof("no pool in memory for %s, create new pool", a)
			// Stop everything lock, we're writing.
			p.mutex.Lock()
			p.connections[a] = map[string]net.Conn{nameConn(conn): conn}
			p.mutex.RUnlock()
			continue
		}

		//we have pool, before we add conn, Double check we don't have it already, buggy clients will be foolish.
		_, okConn := pool[nameConn(conn)]
		if okConn {
			logger.Infof("Connection %s already in %s pool, ignoring", nameConn(conn), a)
			continue
		}

		// Stop everything lock, we're writing, Add connection to existing pool
		p.mutex.Lock()
		p.connections[a][nameConn(conn)] = conn
		p.mutex.Unlock()

		logger.Infof("successfully added connection %s to existing pool: %s, len(%d)", nameConn(conn), a, len(pool))
	}
}

func (p *pools) unsubscribeToPools(pools []string, conn net.Conn) {
	logger.Infof("Unsubscribe connection %s from pools", nameConn(conn))
	for _, a := range pools {
		p.mutex.RLock()
		// TODO(kano): Using pool like this would make it an overloaded term in Midgard.
		//      Pool usually is a string. Let's rename this to something else, maybe poolConnections
		pool, okPool := p.connections[a]
		p.mutex.RUnlock()
		if !okPool {
			// Don't have a pool, create a new one, easy
			logger.Warnf("no pool in memory for %s, to unsubscribe conn from, weird", a)
			continue
		}

		// TODO(kano): we need the lock all the time while we work with pool.

		// we have pool, make sure the conn is there...
		_, okConn := pool[nameConn(conn)]
		if !okConn {
			logger.Infof("Connection %s not in pool, ignoring", nameConn(conn), a)
			continue
		}
		// delete it
		delete(pool, nameConn(conn))

		p.mutex.Lock()
		p.connections[a] = pool
		p.mutex.Unlock()

		logger.Infof("successfully removed connection from pool: %s, len(%d)", nameConn(conn), len(pool))
	}
}

func (p *pools) clearConnectionFromPools(conn net.Conn) {
	// TODO(kano): Don't fetch pools, but iterate through the map in unsubscribeToPools.
	pools := fetchValidatedPools()
	keys := make([]string, 0, len(pools))
	for k := range pools {
		keys = append(keys, k)
	}
	p.unsubscribeToPools(keys, conn)
}

func (p *pools) messageAndDisconnect(message string, conn net.Conn) {
	logger.Infof("messageAndDisconnect %s for conn %s", message, nameConn(conn))
	writer := wsutil.NewWriterSize(conn, ws.StateServerSide, ws.OpText, 128)
	i := &Instruction{
		Message: message,
	}

	pi, err := json.Marshal(i)
	if err != nil {
		logger.Warnf("marshalling err %v", err)
	}

	if _, err := io.Copy(writer, bytes.NewReader(pi)); err != nil {
		logger.Warnf("Failed to copy message to buffer %v", err)
	}

	if err := writer.Flush(); err != nil {
		logger.Warnf("Failed to flush buffer %v", err)
	}

	p.clearConnectionFromPools(conn)

	if err := epoller.Remove(conn); err != nil {
		logger.Warnf("Failed to remove %v", err)
	}
	conn.Close()
}

// ------------------------------- //
//   Helper Methods
// ------------------------------- //
func nameConn(conn net.Conn) string {
	// TODO(kano): add comment with an example result
	// TODO(kano): let's discuss if this is unique. If the key could be anything how about using
	//   a unique int
	return conn.LocalAddr().String() + " > " + conn.RemoteAddr().String()
}

// TODO (acsaba): factor out pool validation.
func fetchValidatedPools() map[string]int {
	result := map[string]int{}
	assetE8DepthPerPool, runeE8DepthPerPool, _ := timeseries.AssetAndRuneDepths()
	for k := range assetE8DepthPerPool {
		_, assetOk := assetE8DepthPerPool[k]
		_, runeOk := runeE8DepthPerPool[k]
		if !assetOk && !runeOk {
			logger.Infof("Invalid Pool %s", k)
			continue
		}
		result[k] = 1
	}
	return result
}

func validateAsset(asset string, pools map[string]int) bool {
	_, ok := pools[asset]
	return ok
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	logger.Info("incoming websocket connection, upgrade, efficiently.")

	// TODO(kano): check hard conn limit and reject anything when we are up again that limit
	// TODO(kano): Hard DDOS prevention.

	// Upgrade connection
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		return
	}
	if err := epoller.Add(conn); err != nil {
		logger.Warnf("Failed to add connection to EPoll %v", err)
		conn.Close()
	}
}
