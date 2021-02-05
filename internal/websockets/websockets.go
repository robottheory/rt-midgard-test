// +build linux

package websockets

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/tendermint/tendermint/libs/rand"
	"gitlab.com/thorchain/midgard/chain"
	"gitlab.com/thorchain/midgard/internal/timeseries"

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
func Serve(listenPort int, connectionLimit int) {
	logger.Infof("Starting Websocket process for pool prices on %d with connection limit %d", listenPort, connectionLimit)
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		logger.Warnf("Can't get the rLimit %v", err)
		return
	}

	rLimit.Cur = uint64(rLimit.Max)
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		logger.Warnf("Can't set the syscall limitr %v", err)
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
		write(p)
	}
}

// TODO(kano): let's discuss. Is this reading messages or accepting connections or both?
//   let's change name or document after discussion.
//   Kano: It's doing both
func readMessagesWaiting() {
	for {
		// TODO(kano): rename to newConnections
		// Not sure about this. there are not new connections, these are
		// connections that have a message ready to be processed.
		// these should be
		// a) connect to certain number of asset notifications
		// b) disconnect from a certain number of asset notifications
		// c) unrecognizable messages that should be ignored or flag rogue connection
		connections, err := connManager.WaitOnReceive()
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
				clearConnEntirely(fd, "connection is nil")
				continue
			}

			msg, _, err := wsutil.ReadClientData(conn)
			if err != nil {
				logger.Warnf("Error reading from socket %v", err)
				// TODO(kano): let's discuss how closing channel looks from users point of view.
				//     do they get error messages?
				//
				// TODO(acsaba):
				// Let's use the flush attempts to determine if we should delete connection entirely
				clearConnEntirely(fd, "unable to read msg waiting from socket")
				continue
			}

			logger.Warnf("received msg \n %s", string(msg))
			i := &Instruction{}

			if err := json.Unmarshal(msg, i); err != nil {
				// TODO(acsaba): make
				logger.Warnf("Error marshaling message from %s \n closing connection", nameConn(conn))
				clearConnEntirely(fd, "unable to marshall message from connection, check format of payload sent")
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
					// Note(kano):
					//   But what if some of the assets requested were valid and one wasn't ?
					//   Should it then close the connection just because of one wrong asset.
					//   I think we should send a message to client though to inform them of a bogus asset request
					// Note(acsaba):
					//   I strongly think the whole request should shut down because one wrong asset.
					//   This will force the user to make correct queries. The the alternative iway
					//   maybe they won't the error which is bad.
					logger.Warnf("Invalid pool %s ignored", a)
					continue
				}
				validPools = append(validPools, a)
			}

			// Note(kano):
			//   I think this is where we should  msg and close the connection, since the client has requested
			//   access to assets we don't have pools for, hasn't requested access to any asset/pool notifications that we
			//   recognize
			// Note(acsaba):
			//   WOW, I didn't think of the case of a pool becoming unavailable after the start of the request.
			//   Interesting edge case and not that clear cut. This is a very rare situation, if ever.
			//   I still think we should disconnect.
			if len(validPools) == 0 {
				messageAndDisconnect(fd, "No valid assets were provided")
				continue
			}

			if len(validPools) != len(i.Assets) {
				logger.Warnf("validPools len %d, incoming pools len %d", len(validPools), len(i.Assets))
				// TODO(kano): let's discuss. Is this is not a client error? If yes then we
				//     don't need to return error, just report error messag and close connection.
				// TODO(acsaba):
				//  It's a mistake, I don't think it's an error that merits hard disconnect.
				// What if a pool with a certain asset becomes unavailable in between the time this was first run and then run again
			}

			if i.Message == Connect {
				subscribeToPools(validPools, fd, conn)
			} else if i.Message == Disconnect {
				// TODO(kano): let's discuss this. It's not clear to acsaba if
				//   Wait returns new connections or messages too. Let's clarify the working of
				//   unsubscribe and wait and then let's change the function names or add
				//   documentation.
				unsubscribeFromPools(fd, validPools)
			}
			// TODO(kano): what if it's Message is neither? handle or document.
			// Maybe we should kill the connection then.
			// In reality if the client is sending garble, more than likely we won't be able to unmarshall the msg
		}
	}
}

func write(update *Payload) {
	// Only write to connections concerned with update.Asset.
	connManager.assetMutex.RLock()
	assetConns, ok := connManager.assetFDs[update.Asset]
	// TODO(kano): defer unlock, because we still access assetConns later.
	connManager.assetMutex.RUnlock()
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

func subscribeToPools(assets []string, fd int, conn net.Conn) {
	for _, asset := range assets {
		// TODO(acsaba): refactor to use less locking/unlicking,
		//   maybe add one lock grab at the beginning.

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

// TODO(kano): revers parameter order to match the subscribe.
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

		// TODO(kano): double locking. Consider locking just unce in the beginning.
		connManager.assetMutex.RLock()
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
	assets := []string{}
	for asset := range connManager.assetFDs {
		assets = append(assets, asset)
	}
	unsubscribeFromPools(fd, assets)
	messageAndDisconnect(fd, disconnMsg)
}

func messageAndDisconnect(fd int, message string) {
	logger.Infof("messageAndDisconnect %s for conn %d", message, fd)
	// TODO(kano): Not locked. Consider making helper functions for accessing the connection,
	//   that way the locking logic will not be forgotten.
	writer := wsutil.NewWriterSize(connManager.connections[fd], ws.StateServerSide, ws.OpText, MAX_BYTE_LENGTH_FLUSH)
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

	if err := connManager.Remove(connManager.connections[fd]); err != nil {
		logger.Warnf("Failed to remove %v", err)
	}
}

// ------------------------------- //
//   Helper Methods
// ------------------------------- //
func nameConn(conn net.Conn) string {
	// TODO(kano): add comment with an example result
	// TODO(kano): let's discuss if this is unique. If the key could be anything how about using
	//   a unique int
	// In only use this for logging right now, the FD int is used as the unique key to identify the conn
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
	if err := connManager.Add(conn); err != nil {
		logger.Warnf("Failed to add connection to EPoll %v", err)
		conn.Close()
	}
}
