package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go-websocket-benchmark/config"
	"go-websocket-benchmark/logging"

	"github.com/lesismal/nbio/nbhttp/websocket"
	"github.com/libp2p/go-reuseport"
)

var (
	_ = flag.Int("b", 1024, `read buffer size`)
	_ = flag.Int("mrb", 4096, `max read buffer size`)
	_ = flag.Int64("m", 1024*1024*1024*2, `memory limit`)
	_ = flag.Int("mb", 10000, `max blocking online num, e.g. 10000`)

	upgrader = websocket.NewUpgrader()

	chExit = make(chan struct{})
)

func main() {
	flag.Parse()

	upgrader.OnMessage(func(c *websocket.Conn, messageType websocket.MessageType, data []byte) {
		c.WriteMessage(messageType, data)
	})

	// ports := strings.Split(config.Ports[config.NbioBasedonStdhttp], ":")
	// minPort, err := strconv.Atoi(ports[0])
	// if err != nil {
	// 	log.Fatalf("invalid port range: %v, %v", ports, err)
	// }
	// maxPort, err := strconv.Atoi(ports[1])
	// if err != nil {
	// 	log.Fatalf("invalid port range: %v, %v", ports, err)
	// }
	// addrs := []string{}
	// for i := minPort; i <= maxPort; i++ {
	// 	addrs = append(addrs, fmt.Sprintf(":%d", i))
	// }
	addrs, err := config.GetFrameworkServerAddrs(config.NbioBasedonStdhttp)
	if err != nil {
		logging.Fatalf("GetFrameworkBenchmarkAddrs(%v) failed: %v", config.NbioBasedonStdhttp, err)
	}
	startServers(addrs)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	close(chExit)
}

func startServers(addrs []string) {
	for _, v := range addrs {
		go func(addr string) {
			mux := &http.ServeMux{}
			mux.HandleFunc("/ws", onWebsocket)
			mux.HandleFunc("/pid", onServerPid)
			server := http.Server{
				// Addr:    addr,
				Handler: mux,
			}
			ln, err := reuseport.Listen("tcp", addr)
			if err != nil {
				logging.Fatalf("Listen failed: %v", err)
			}
			go func() {
				<-chExit
				ln.Close()
			}()
			logging.Fatalf("server exit: %v", server.Serve(ln))
		}(v)
	}
}

func onServerPid(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%d", os.Getpid())
}

func onWebsocket(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade failed: %v", err)
		return
	}
	c.SetReadDeadline(time.Time{})
}
