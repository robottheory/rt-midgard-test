// +build linux

package websockets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/julienschmidt/httprouter"
	"gitlab.com/thorchain/midgard/chain"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/util/timer"

	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
)

// TODO(acsaba): change everyh Warnf call to log maximum once every X min.
// TODO(acsaba): add counters next to every Warnf

// TODO(kano): Info logs is used on a lot of places. We should delete them, or perhaps we could
//   allow logging for debugging with loglevel or logmodule.

var (
	connManager *connectionManager
	// TODO(kano): - extend logger to application name to prefix all output with service name => in this case websockets.
	logger                = NewLogger()
	MAX_BYTE_LENGTH_FLUSH = 128
)

// Entrypoint.
func Serve(connectionLimit int) {
	logger.Infof("Starting Websocket goroutine for pool prices with connection limit %d", connectionLimit)
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		logger.Warnf("Can't get the rLimit %v", err)
		return
	}

	rLimit.Cur = uint64(rLimit.Max)
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		logger.Warnf("Can't set the syscall rlimit %v", err)
		return
	}

	// Start connectionManger
	var err error
	connManager, err = ConnectionManagerInit(connectionLimit)
	if err != nil {
		logger.Warnf("Can't create the connectionManager %v", err)
		return
	}

	go readMessagesWaiting()

	for {
		waitForBlock()
		logger.Info("Sending push price updates")

		notifyClients()
	}
}

func waitForBlock() {
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
	// TODO(acsaba): notify only after block is digested.
	//     Unfortunately we get notification when the block arrives from the node.
	//     It would be better if notification is sent after the block is digested.
	//     We can change this logic after the digestion knows about sync status,
	//     (we don't want to update durring catchup phase).
	time.Sleep(500 * time.Millisecond)
}

var TestChannel *chan Payload

// TODO(kano): change unit test to connect through real websockets, then delete this.
func NotifyTest(p Payload) {
	// if TestChannel !=
	if TestChannel != nil {
		logger.Info("send to test: ", p)
		*TestChannel <- p
		logger.Info("sent :)", p)
	}
}

func notifyClients() {
	// TODO(acsaba): refactor depthrecorder, updates prices only if there was a change.
	//     Currently for testing purposes we update even if the price stayed the same.
	state := timeseries.Latest.GetState()
	for pool, info := range state.Pools {
		payload := &Payload{
			Price: strconv.FormatFloat(info.AssetPrice(), 'g', -1, 64),
			Asset: pool,
		}

		logger.Info("send price info to websockets: ", payload)
		NotifyTest(*payload)

		write(payload)

	}

}

var recieveWaitTimer = timer.NewMilli("websocket_recieve_wait")
var recieveProcessTimer = timer.NewMilli("websocket_recieve_process")

// Listens for connections subscribing/unsubscribing from pools.
func readMessagesWaiting() {
	for {
		waitTimer := recieveWaitTimer.One()
		connections, err := connManager.WaitOnReceive()
		waitTimer()

		recieveTimer := recieveProcessTimer.One()
		if err != nil {
			// To be expected...
			if err.Error() == "interrupted system call" {
				continue
			}
			logger.Warnf("Failed to epoll wait %v", err)
			return
		}

		for fd, conn := range connections {
			if conn == nil {
				logger.Warnf("Nil connections in the pool, this shouldn't happen.")
				connManager.Remove(connManager.connections[fd])
				continue
			}

			msg, _, err := wsutil.ReadClientData(conn)
			if err != nil {
				logger.Warnf("Error reading from socket %v", err)
				clearConnEntirely(fd, "unable to read msg waiting from socket")
				continue
			}

			logger.Infof("received msg \n %s", string(msg))
			i := &Instruction{}

			if err := json.Unmarshal(msg, i); err != nil {
				logger.Warnf("Error marshaling message from %s \n closing connection", nameConn(conn))
				clearConnEntirely(fd, "unable to unmarshall message from connection, check format of payload sent")
				continue
			}

			// TODO(acsaba): add metric for i.Assets
			logger.Infof("instruction received for assets %s", strings.Join(i.Assets, ","))
			pools := timeseries.Latest.GetState()
			validPools := []string{}
			valid := true
			for _, a := range i.Assets {
				if !pools.PoolExists(a) {
					clearConnEntirely(fd, fmt.Sprintf("invalid asset %s was provided", a))
					valid = false
					break
				}
				validPools = append(validPools, a)
			}
			if !valid {
				continue
			}

			if len(validPools) == 0 {
				clearConnEntirely(fd, "No valid assets were provided")
				continue
			}

			if i.Message == MessageConnect {
				subscribeToPools(fd, validPools, conn)
			} else if i.Message == MessageDisconnect {
				unsubscribeFromPools(fd, validPools)
			} else {
				// What the hell is this.
				clearConnEntirely(fd, fmt.Sprintf("Message not recognized, %s", i.Message))
			}
		}
		recieveTimer()
	}
}

func write(update *Payload) {
	// Only write to connections concerned with update.Asset.
	connManager.assetMutex.RLock()
	defer connManager.assetMutex.RUnlock()
	assetConns, ok := connManager.assetFDs[update.Asset]

	if !ok {
		logger.Infof("no pool in memory for %s, ignoring ...", update.Asset)
		return
	}

	payload, err := json.Marshal(update)
	if err != nil {
		logger.Warnf("marshalling err on write %v", err)
		return
	}

	for fd, connectionAttempts := range assetConns {
		// TODO(kano): get the connections mutex.
		// Note(acsaba):
		//   I think we need the mutex of connections here.
		//   Probably not the case programatically, but this might even modify connections.
		writer := wsutil.NewWriterSize(connManager.connections[fd], ws.StateServerSide, ws.OpText, MAX_BYTE_LENGTH_FLUSH)
		if _, err := io.Copy(writer, bytes.NewReader(payload)); err != nil {
			logger.Infof("Failed to copy message to buffer %v", err)
			return
		}
		if err := writer.Flush(); err != nil {
			if connectionAttempts >= MAX_FLUSH_ATTEMPT {
				clearConnEntirely(fd, "3 attempted flushs to connection have failed, disconnecting")
			} else {
				connManager.assetMutex.Lock()
				connManager.assetFDs[update.Asset][fd] = connectionAttempts + 1
				connManager.assetMutex.Unlock()
				continue
			}
			// reset connection attempts
			connManager.assetMutex.Lock()
			connManager.assetFDs[update.Asset][fd] = INIT_FLUSH_COUNT
			connManager.assetMutex.Unlock()
		}
		// TODO(kano): remove once we are happy with performance.
		logger.Infof("Write succeeded for asset %s on connection %d", update.Asset, fd)
	}
}

func subscribeToPools(fd int, assets []string, conn net.Conn) {
	for _, asset := range assets {
		// TODO(acsaba): refactor to use less locking/unlicking,
		//   maybe add one lock grab at the beginning.
		// In order to keep it working at optimal performance, we need to use the
		// right type of lock at the write point, reads or write locks

		// Read lock
		connManager.assetMutex.RLock()
		assetConns, ok := connManager.assetFDs[asset]
		connManager.assetMutex.RUnlock()

		if !ok {
			// Don't have a assetConns, create a new one, easy
			logger.Infof("no assetConns in memory for %s, create new assetConns entry", asset)
			// Stop everything lock, we're writing.
			connManager.assetMutex.Lock()
			connManager.assetFDs[asset] = map[int]int{fd: INIT_FLUSH_COUNT}
			connManager.assetMutex.Unlock()
			continue
		}

		//we have assetConns, before we add conn, Double check we don't have it already, buggy clients will be foolish.
		connManager.assetMutex.RLock()
		_, okConn := assetConns[fd]
		if okConn {
			logger.Infof("Connection %s already in %s assetConns, ignoring", nameConn(conn), asset)
			connManager.assetMutex.RUnlock()
			continue
		}
		connManager.assetMutex.RUnlock()

		// Stop everything lock, we're writing, Add connection to existing pool
		connManager.assetMutex.Lock()
		connManager.assetFDs[asset][fd] = INIT_FLUSH_COUNT
		connManager.assetMutex.Unlock()

		logger.Infof("successfully added connection %s to existing pool: %s, len(%d)", nameConn(conn), asset, len(assetConns))
	}
}

func unsubscribeFromPools(fd int, assets []string) {
	logger.Infof("Unsubscribe connection %d from pools", fd)
	for _, asset := range assets {
		connManager.assetMutex.RLock()
		assetConns, ok := connManager.assetFDs[asset]
		if !ok {

			// Don't have a assetConns
			logger.Warnf("no assetConns in memory for %s, to unsubscribe conn from, weird", asset)
			connManager.assetMutex.RUnlock()
			continue
		}

		_, okConn := assetConns[fd]
		if !okConn {
			logger.Infof("Connection %d not in assetConns, ignoring %v", fd, asset)
			connManager.assetMutex.RUnlock()
			continue
		}

		connManager.assetMutex.RUnlock()

		// delete it
		connManager.assetMutex.Lock()
		delete(assetConns, fd)
		connManager.assetFDs[asset] = assetConns
		connManager.assetMutex.Unlock()

		logger.Infof("successfully removed connection from assetConns: %s, len(%d)", asset, len(assetConns))
	}
}

func clearConnEntirely(fd int, disconnMsg string) {
	// TODO(kano): Not locked. Do lock assetFDs
	// Todo(kano): Consider: if we make a struct where we have the connection and the
	//   connection attempts together, we might want to put the followed pools there too.
	// We can redo the structure of how things are done once we have things working well with unit tests
	// but that forloop below is not expensive but one whereby you have to continuously iterate over arrays of connections
	// is not ideal nor duplicating any references anywhere, hence the 2 maps in connManager, one with fd -> conn the other with
	// asset -> fd -> connectionAttempt ->-> quick look up.
	assets := []string{}
	connManager.assetMutex.RLock()
	for asset := range connManager.assetFDs {
		assets = append(assets, asset)
	}
	connManager.assetMutex.RUnlock()

	unsubscribeFromPools(fd, assets)
	messageAndDisconnect(fd, disconnMsg)
}

func messageAndDisconnect(fd int, message string) {
	logger.Infof("messageAndDisconnect %s for conn %d", message, fd)
	con := connManager.GetConnection(fd)
	if con == nil {
		logger.Warn("Was not able to find connection:", fd)
		return
	}
	writer := wsutil.NewWriterSize(*con, ws.StateServerSide, ws.OpText, MAX_BYTE_LENGTH_FLUSH)
	i := &Instruction{
		Message: message,
	}

	pi, err := json.Marshal(i)
	if err != nil {
		logger.Warnf("marshalling err %v", err)
		return
	}

	if _, err := io.Copy(writer, bytes.NewReader(pi)); err != nil {
		logger.Warnf("Failed to copy message to buffer %v", err)
	} else if err := writer.Flush(); err != nil {
		logger.Warnf("Failed to flush buffer %v", err)
	}

	connManager.Remove(connManager.connections[fd])
}

// ------------------------------- //
//   Helper Methods
// ------------------------------- //
func nameConn(conn net.Conn) string {
	return conn.LocalAddr().String() + " > " + conn.RemoteAddr().String()
}

func WsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if len(connManager.connections) >= connManager.connLimit {
		logger.Info("reject incoming connection, we are maxed out")
		fmt.Fprintf(w, "max websocket connections on this node, no room left ... ")
		return
	}

	logger.Info("incoming websocket connection, upgrade, efficiently.")

	// Upgrade connection
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		fmt.Fprint(w, "Failed to ugrade connection", err)
		logger.Warn("Failed to upgrade connection: ", err)
		return
	}
	if err := connManager.Add(conn); err != nil {
		logger.Warnf("Failed to add connection to EPoll %v", err)
		conn.Close()
	}
}
