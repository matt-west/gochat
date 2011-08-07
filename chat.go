package main

import (
	"flag"
	"http"
	"log"
	"template"
	"websocket"
)

var addr = flag.String("addr", ":8080", "http service address")

func main() {
	flag.Parse()
	go hub()
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/ws", webSocketProtocolSwitch)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func webSocketProtocolSwitch(c http.ResponseWriter, req *http.Request) {
	// Handle old and new versions of protocol.
	if _, found := req.Header["Sec-Websocket-Key1"]; found {
		websocket.Handler(clientHandler).ServeHTTP(c, req)
	} else {
		websocket.Draft75Handler(clientHandler).ServeHTTP(c, req)
	}
}

var messageChan = make(chan []byte)

type subscription struct {
	conn      *websocket.Conn
	subscribe bool
}

var subscriptionChan = make(chan subscription)

func hub() {
	conns := make(map[*websocket.Conn]int)
	for {
		select {
		case subscription := <-subscriptionChan:
			conns[subscription.conn] = 0, subscription.subscribe
		case message := <-messageChan:
			for conn, _ := range conns {
				if _, err := conn.Write(message); err != nil {
					conn.Close()
				}
			}
		}
	}
}

func clientHandler(ws *websocket.Conn) {
	defer func() {
		subscriptionChan <- subscription{ws, false}
		ws.Close()
	}()

	subscriptionChan <- subscription{ws, true}

	for {
	    buf := make([]byte, 256)
		n, err := ws.Read(buf)
		if err != nil {
			break
		}
		messageChan <- buf[0:n]
	}
}

// Handle home page requests.
func homeHandler(c http.ResponseWriter, req *http.Request) {
	homeTempl.Execute(c, req.Host)
}

var homeTempl *template.Template

func init() {
	homeTempl = template.New(nil)
	homeTempl.SetDelims("<<", ">>")
	if err := homeTempl.ParseFile("index.html"); err != nil {
		panic("template error: " + err.String())
	}
}
