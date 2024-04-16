package main

import (
	"flag"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

type wsServer struct {
	gnet.BuiltinEventEngine

	addr      string
	multicore bool
	eng       gnet.Engine
}

func (wss *wsServer) OnBoot(eng gnet.Engine) gnet.Action {
	wss.eng = eng
	logging.Infof("echo server with multi-core=%t is listening on %s", wss.multicore, wss.addr)
	return gnet.None
}

func (wss *wsServer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	c.SetContext(new(wsCodec))
	return nil, gnet.None
}

func (wss *wsServer) OnTraffic(c gnet.Conn) (action gnet.Action) {
	ws := c.Context().(*wsCodec)
	if ws.readBufferBytes(c) == gnet.Close {
		return gnet.Close
	}
	ok, action := ws.upgrade(c)
	if !ok {
		return
	}

	if ws.buf.Len() <= 0 {
		return gnet.None
	}
	messages, err := ws.Decode(c)
	if err != nil {
		return gnet.Close
	}
	if messages == nil {
		return
	}

	for _, message := range messages {
		msgLen := len(message.Payload)
		// This is the echo server
		err = wsutil.WriteServerMessage(c, message.OpCode, message.Payload)
		if err != nil {
			logging.Infof("conn[%v] [err=%v]", c.RemoteAddr().String(), err.Error())
			return gnet.Close
		}
	}

	return gnet.None
}

func main() {
	var port int
	var multicore bool

	// Example command: go run main.go --port 8080 --multicore=true
	flag.IntVar(&port, "port", 9080, "server port")
	flag.Parse()

	wss := &wsServer{addr: fmt.Sprintf("tcp://127.0.0.1:%d", port), multicore: true}

	// Start serving!
	log.Println("server exits:", gnet.Run(wss, wss.addr, gnet.WithMulticore(multicore), gnet.WithReusePort(true), gnet.WithTicker(true)))
}
