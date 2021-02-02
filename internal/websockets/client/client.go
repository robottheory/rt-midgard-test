package main

import (
	"encoding/json"
	"flag"
	"io"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"gitlab.com/thorchain/midgard/internal/websockets"
)

var (
	ip          = flag.String("ip", "127.0.0.1", "server IP")
	connections = flag.Int("conn", 1, "number of websocket connections")
	logger = websockets.NewLogger()
)


var disconnect = &websockets.Instruction{
	Message: "MessageDisconnect",
	Assets:  []string{"BTC.BTC"},
}

var connect = &websockets.Instruction{
	Message: "MessageConnect",
	Assets:  []string{"BTC.BTC"},
}

func diconnectConnection(conn *websocket.Conn) {
	marshalled, err := json.Marshal(disconnect)
	if err != nil {
		logger.Warnf("error marshalling %v", err)
	} else {
		conn.WriteMessage(websocket.TextMessage, marshalled)
	}

	time.Sleep(time.Second * 1)

	conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	conn.Close()
	logger.Infof("Connection closed on %s", conn.RemoteAddr().String())
}

func main() {
	flag.Usage = func() {
		io.WriteString(os.Stderr, `Websockets client generator
Example usage: ./client -ip=172.17.0.1 -conn=10
`)
		flag.PrintDefaults()
	}
	flag.Parse()

	u := url.URL{Scheme: "ws", Host: *ip + ":6060", Path: "/v2/ws"}
	logger.Infof("Connecting to %s", u.String())
	var conns []*websocket.Conn
	tts := time.Second
	if *connections > 100 {
		tts = time.Millisecond * 5
	}

	for i := 0; i < *connections; i++ {
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			logger.Warnf("Failed to connect", i, err)
			break
		}
		conns = append(conns, c)

		defer func() {
		}()
	}

	logger.Infof("Finished initializing %d connections", len(conns))

	// Send the connect to assetPool message
	for i := 0; i < len(conns); i++ {
		time.Sleep(tts)
		conn := conns[i]
		logger.Infof("Conn %d sending message", i)
		if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second*5)); err != nil {
			logger.Warnf("Failed to receive pong: %v", err)
		}

		marshalled, err := json.Marshal(connect)
		if err != nil {
			logger.Warnf("error marshalling %v", err)
		} else {
			if err := conn.WriteMessage(websocket.TextMessage, marshalled); err != nil {
				logger.Infof("err %v", err)
			}
		}
	}

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)

	for {
		select {
		case <-signalChannel:
			logger.Info("someone or something hit ctrl c, canceling input...")
			logger.Info("sleep for 3 secs, have hit cancel, give the routines a few ticks to settle up, then quit input")
			for _, conn := range conns {
				// Disconnect everything.
				diconnectConnection(conn)
				time.Sleep(1 * time.Second)
			}
			time.Sleep(1 * time.Second)
			return
		}

		for i, conn := range conns {
			_, message, err := conn.ReadMessage()
			if err != nil {
				logger.Infof("read message on conn %d err :", err, i)
				return
			}

			p := &websockets.Payload{}

			if err := json.Unmarshal(message, p); err != nil {
				logger.Warnf("error unmarshalling %v", err)
			}

			logger.Infof("Symbol %s on conn %d", p.Asset, i)
			logger.Infof("\nPrice %s on conn %d", p.Price, i)
		}
	}
}
