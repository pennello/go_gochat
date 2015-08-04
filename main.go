package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"html/template"
	"net/http"
	"os/signal"

	"golang.org/x/net/websocket"
)

const (
	httpPort      = 8080
	websocketPort = 8081
)

var allConns = make(map[*websocket.Conn]interface{})
var mutex = sync.Mutex{}

func addConn(conn *websocket.Conn) {
	mutex.Lock()
	defer mutex.Unlock()
	allConns[conn] = nil
	log.Print("client connected")
}

func rmConn(conn *websocket.Conn) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(allConns, conn)
	log.Print("client disconnected")
}

func writeAll(msg string) {
	mutex.Lock()
	defer mutex.Unlock()
	for conn, _ := range allConns {
		err := websocket.Message.Send(conn, msg)
		if err != nil {
			log.Print(err)
		}
	}
}

func getUrl(proto string, port int) string {
	server, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("%s://%s:%d", proto, server, port)
}

type indexServer struct{}

func (i indexServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	t, err := template.ParseFiles("index.tmpl")
	if err != nil {
		log.Fatal(err)
	}
	serverUrl := getUrl("ws", websocketPort)
	w.Header()["Content-Type"] = []string{"text/html; charset=utf-8"}
	err = t.Execute(w, serverUrl)
	if err != nil {
		log.Print(err)
	}
}

func serve(handler http.Handler, port int) {
	addr := fmt.Sprintf(":%d", port)
	log.Fatal(http.ListenAndServe(addr, handler))
}

func serveIndex() {
	serve(indexServer{}, httpPort)
}

func websocketHandler(conn *websocket.Conn) {
	addConn(conn)
	var msg string
	for {
		err := websocket.Message.Receive(conn, &msg)
		if err != nil {
			if err == io.EOF {
				rmConn(conn)
				return
			}
			log.Print(err)
			continue
		}
		writeAll(msg)
	}
}

func serveWebsocket() {
	serverUrl := getUrl("ws", websocketPort)
	originUrl := getUrl("http", httpPort)
	conf, err := websocket.NewConfig(serverUrl, originUrl)
	if err != nil {
		log.Fatal(err)
	}
	server := websocket.Server{
		Config:  *conf,
		Handler: websocketHandler,
	}
	serve(server, websocketPort)
}

func main() {
	go serveIndex()
	go serveWebsocket()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	<-c
}
