 
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
	"sc/model"

	model_auth_session "sc/models/auth_session"
	model_user "sc/models/user"
	// "sc/model"
)

const (
	WriteWait		= 10 * time.Second
	PongWait		= 10 * time.Second
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

	remoteAddr		string
	userAgent		string

	SessionExists	bool
	Session			*model_auth_session.Session

	IsServerAuth	bool
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
	ctx *command.Context,
	remoteAddr string,
	userAgent string) *Connection {

	c := Connection {
		IsServerAuth:	false,
		commands:		commands,
		Id:				id,
		ws:				ws,
		closeChannel:	closeChannel,
		writeChannel:	writeChannel,
		commandContext:	ctx,		
		SessionExists:	false,
		remoteAddr:		remoteAddr,
		userAgent:		userAgent,
	}

	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(PongWait)); return nil })

	return &c
}

func timeWrapper(commandName string, command command.Command, message []byte) {
	t := time.Now()

	command.Execute(message)

	d := time.Now().Sub(t)
	logger.String(fmt.Sprintf("command '%s' time: %0.5f", commandName, float64(d) / float64(time.Second)))
}

func (c *Connection) SetServerAuthState () {
	c.IsServerAuth = true
}

func (c *Connection) GetServerAuthState () bool {
	return c.IsServerAuth
}

func (c *Connection) GetSession () *model_auth_session.Session {

	if c.SessionExists {
		return c.Session
	}

	return nil
}

func (c *Connection) SetSession (session *model_auth_session.Session) {

	c.Session = session
	c.SessionExists = true
}

func (c *Connection) Reading() {

	c.ws.SetReadLimit(MaxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(PongWait))	

	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			if err != io.EOF && err.Error() != "websocket: close 1005 " {
				logger.Error(errors.New(err))
			}
			break
		}
		smessage := string(message)

		if smessage == `{"command":"ping"}` {
			c.Send(`{"command":"pong"}`)
			continue
		} else {
			logger.String(fmt.Sprintf("message: %v", smessage))
		}

		commandDetector := CommandDetector{}
		err = json.Unmarshal(message, &commandDetector)
		if err != nil {
			logger.Error(errors.New(err))
			continue
		}

		generator, ok := c.commands[commandDetector.Command]
		if ok {
			command := generator(c, c.commandContext)
			go timeWrapper(commandDetector.Command, command, message)
		}
	}

	if c.SessionExists {
		if c.Session.IsAuth {
			user := model_user.Get(c.Session.UserUUID.String())
			user.Unlock()
		}
		c.Session.Unlock()

	}

}

func (c *Connection) UnAuth() {

	if c.SessionExists && c.Session.IsAuth {

		user := model_user.Get(c.Session.UserUUID.String())
		if user != nil {
			user.Unlock()
		}

		
		c.Session.Update(model.Fields{
			"IsAuth": false,
			"UserUUID": nil,
			"AuthMethod": nil,
		})

		c.Send(`{"command":"reload"}`)
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

func (c *Connection) GetRemoteAddr() string {
	return c.remoteAddr
}

func (c *Connection) GetUserAgent() string {
	return c.userAgent
}
