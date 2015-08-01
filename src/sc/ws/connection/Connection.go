 
package connection

import (
	"github.com/gorilla/websocket"
	"net/http"
	"time"
	"fmt"
	"encoding/json"
	"io"

	"sc/logger"
	"sc/errors"
	"sc/ws/command"
)

const (
	WriteWait		= 10 * time.Second
	PongWait		= 60 * time.Second
	PingPeriod		= (PongWait * 9) / 10
	MaxMessageSize	= 4096
)

type Connection struct {
	Id				uint32
	ws				*websocket.Conn
	closeChannel	chan *Connection	
	writeChannel	*WriteChannel
	commands		map[string]command.Generator
	commandContext	*command.Context
}

type CommandDetector struct {
	Command string `json:"command"`
}

type WriteMessage struct {
	Connection *Connection
	Message string
}

type WriteChannel chan *WriteMessage


func New(
	ws *websocket.Conn,
	id uint32,
	closeChannel chan *Connection,
	writeChannel *WriteChannel,
	commands map[string]command.Generator,
	ctx *command.Context) *Connection {

	c := Connection {
		commands:		commands,
		Id:				id,
		ws:				ws,
		closeChannel:	closeChannel,
		writeChannel:	writeChannel,
		commandContext:	ctx,
	}

	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(PongWait)); return nil })

	return &c
}

func (c *Connection) Reading() {

	c.ws.SetReadLimit(MaxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(PongWait))	

	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			if err != io.EOF {
				logger.Error(errors.New(err))
			}
			break
		}
		smessage := string(message)
		logger.String(fmt.Sprintf("message: %v", smessage))

		commandDetector := CommandDetector{}
		err = json.Unmarshal(message, &commandDetector)
		if err != nil {
			logger.Error(errors.New(err))
			continue
		}

		logger.String(fmt.Sprintf("message: %+v", commandDetector))
		// c.Send(smessage)

		generator, ok := c.commands[commandDetector.Command]
		if ok {
			command := generator(c, c.commandContext)
			go command.Execute(message)
		}
	}

}

func (c *Connection) Send(message string) {
	m := WriteMessage{ Connection: c, Message: message }
	*c.writeChannel <- &m
}

func (c *Connection) Write(mt int, message []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(WriteWait))
	return c.ws.WriteMessage(mt, message)
}

func (c *Connection) Ping() error {

	c.ws.SetWriteDeadline(time.Now().Add(WriteWait))

	if err := c.ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
		return err
	}

	return nil
}

func (c *Connection) Close() {
	c.ws.Close()
	c.closeChannel <- c
}


var Upgrader = websocket.Upgrader {
	ReadBufferSize:		4096,
	WriteBufferSize:	4096,
	CheckOrigin:		func(r *http.Request) bool { return true },
}
