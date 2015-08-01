
package main

import (
	"github.com/gorilla/websocket"
	"net/http"
	"time"
	"fmt"
	"sync"
	"encoding/json"
	"io"

	"sc/ws/connection/factory"
	"sc/errors"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)


var connectionIdGenerator uint32 = 1

type WSConnectionFactory struct {
	mutex *sync.Mutex
	connectionIdGenerator uint32
	connections map[uint32]*WSConnection
	closeChannel chan *WSConnection
	closeFactory chan bool
}

func NewWSConnectionFactory() *WSConnectionFactory {

	f := WSConnectionFactory{
		closeChannel: make(chan *WSConnection),
		closeFactory: make(chan bool),
		connectionIdGenerator: 1, mutex: &sync.Mutex{},
		connections: make(map[uint32]*WSConnection),
	}

	go f.connectionsTreator()

	return &f
}

type WSConnectionFactoryDebugInfo struct {
	Connections map[string]interface{}
}

func (f *WSConnectionFactory) debugInfo () *WSConnectionFactoryDebugInfo {

	di := WSConnectionFactoryDebugInfo{ Connections: make(map[string]interface{}) }

	f.mutex.Lock()
	for index, _ := range f.connections {
		di.Connections[fmt.Sprintf("%d", index)] = make(map[string]interface{})
	}
	f.mutex.Unlock()

	return &di
}

func (f *WSConnectionFactory) connectionsTreator() {

	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()		
	}()

	for {
		select {

		case c := <-f.closeChannel:
			delete(f.connections, c.Id)

		case <-pingTicker.C:
			var cs map[uint32]*WSConnection = make(map[uint32]*WSConnection)

			f.mutex.Lock()
			for index, connection := range f.connections {
				cs[index] = connection
			}
			f.mutex.Unlock()

			for _, connection := range f.connections {
				err := connection.ping()
				if err != nil {
					connection.close()
				}
			}

		// case <-f.closeFactory:
		}
	}
}

func (f *WSConnectionFactory) CreateConnection(ws *websocket.Conn) *WSConnection {

	f.mutex.Lock()
	id := f.connectionIdGenerator
	f.connectionIdGenerator ++
	f.mutex.Unlock()

	c := NewWSConnection(ws, id, f.closeChannel)
	f.connections[id] = c

	return c
}

type WSConnection struct {
	Id uint32
	ws *websocket.Conn
	send chan []byte
	closeChannel chan *WSConnection	
	writerExit chan bool
}

func NewWSConnection(ws *websocket.Conn, id uint32, closeChannel chan *WSConnection) *WSConnection {

	c := WSConnection{
		Id: id,
		send: make(chan []byte, 256),
		ws: ws,
		closeChannel: closeChannel,
		writerExit: make(chan bool),
	}

	return &c
}

type CommandDetector struct {
	Command string `json:"command"`
}

func (c *WSConnection) reading() {

	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			if err != io.EOF {
				logger.Error(errors.New(err))
			}
			break
		}
		logger.String(fmt.Sprintf("message: %v", string(message)))

		commandDetector := CommandDetector{}
		err = json.Unmarshal(message, &commandDetector)
		if err != nil {
			logger.Error(errors.New(err))
			continue
		}

		logger.String(fmt.Sprintf("message: %+v", commandDetector))

		// h.broadcast <- message
	}

}

func (c *WSConnection) ping() error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
		return err
	}
	return nil
}

func (c *WSConnection) writing() {

	exitLabel: for {
		select {
/*
		case <-pingTicker.C:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
*/
		case _ = <-c.send:

		case <-c.writerExit:
			break exitLabel
		}
	}

	logger.String(fmt.Sprintf("close connection writer %v", c.Id))

}

func (c *WSConnection) close() {
	c.ws.Close()
	c.writerExit <- true
	c.closeChannel <- c
}


var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool { return true },
}

var connectionFactory = NewWSConnectionFactory()

func serveWs(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(errors.New(err))
		return
	}

	c := connectionFactory.CreateConnection(ws)
	logger.String(fmt.Sprintf("accept connection %v", c.Id))
	go c.writing()
	c.reading()
	c.close()
	logger.String(fmt.Sprintf("close connection %v", c.Id))

}

func serveDebug(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(connectionFactory.debugInfo())
	if err != nil {
		logger.Error(errors.New(err))
	}
	io.WriteString(w, string(b))
}
