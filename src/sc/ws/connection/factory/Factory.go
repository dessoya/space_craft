
package factory

import (
	"github.com/gorilla/websocket"
	"time"
	"fmt"
	"sync"

	"sc/ws/connection"
	"sc/ws/command"
	"sc/logger"
	"sc/errors"

	// model2_user "sc/models2/user"
)

type DebugInfo struct {
	Connections map[string]interface{}
	WriteChannelSize int
}

type Factory struct {
	mutex					*sync.Mutex
	connectionIdGenerator	uint32
	connections				map[uint32]*connection.Connection
	closeChannel			chan *connection.Connection
	closeFactory			chan bool
	writeChannel			connection.WriteChannel
	commands				map[string]command.Generator
}

func New() *Factory {

	f := Factory {
		closeChannel:			make(chan *connection.Connection),
		closeFactory:			make(chan bool),
		connectionIdGenerator:	1,
		mutex:					&sync.Mutex{},
		connections:			make(map[uint32]*connection.Connection),
		writeChannel:			make(connection.WriteChannel, 1024),
		commands:				make(map[string]command.Generator),
	}

	go f.connectionsTreator()

	return &f
}

func (f *Factory) GetConnections() []*connection.Connection {

	f.mutex.Lock()
	a := make([]*connection.Connection, 0)
	for _, connection := range f.connections {
		a = append(a, connection)
	}
	f.mutex.Unlock()

	return a
}

func (f *Factory) GetCommands() map[string]command.Generator {
	return f.commands
}

func (f *Factory) connectionsTreator() {

	pingTicker := time.NewTicker(connection.PingPeriod)
	defer func() {
		pingTicker.Stop()		
	}()

	for {
		select {

		case message := <-f.writeChannel:
			logger.String("send message: " + message.Message)
			go func() {
				err := message.Connection.Write(websocket.TextMessage, []byte(message.Message))
				if err != nil {
					logger.Error(errors.New(err))
					message.Connection.Close()					
				}			
			}()

		case c := <-f.closeChannel:
			delete(f.connections, c.Id)

		case <-pingTicker.C:
			var cs map[uint32]*connection.Connection = make(map[uint32]*connection.Connection)

			logger.String("pingTicker.lock")
			f.mutex.Lock()
			for index, connection := range f.connections {
				cs[index] = connection
			}
			f.mutex.Unlock()
			logger.String("pingTicker.unlock")

			for _, connection := range f.connections {
				go func() {
					err := connection.Ping()
					if err != nil {
						connection.Close()
					}
				}()
			}

		// case <-f.closeFactory:
		}
	}
}

func (f *Factory) CreateConnection(ws *websocket.Conn, ctx *command.Context, remoteAddr string, userAgent string) *connection.Connection {

	f.mutex.Lock()
	id := f.connectionIdGenerator
	f.connectionIdGenerator ++
	f.mutex.Unlock()

	c := connection.New(ws, id, f.closeChannel, &f.writeChannel, f.commands, ctx, remoteAddr, userAgent)
	f.connections[id] = c

	return c
}

func (f *Factory) InstallCommand (commandName string, commandGenerator command.Generator) {
	f.commands[commandName] = commandGenerator
}


func (f *Factory) MakeDebugInfo () map[string]interface{} {

	di := map[string]interface{}{
		"Connections": map[string]interface{}{},
		"WriteChannelSize": len(f.writeChannel),
		"Models": map[string]interface{}{
			"Sessions": map[string]interface{}{},
			"Users": map[string]interface{}{},
		},
	}

	f.mutex.Lock()
	for index, conn := range f.connections {
		c := map[string]interface{}{}
		c["SessionUUID"] = conn.Session.UUID.String()
		i := fmt.Sprintf("%d", index)
		item := di["Connections"].(map[string]interface{})
		item[i] = c
	}
	f.mutex.Unlock()

/*
	for _, user := range model_user.Users {

		c := map[string]interface{}{}
		c["IsLock"] = user.IsLock
		
		item := di["Models"].(map[string]interface{})
		item = item["Users"].(map[string]interface{})
		item[user.UUID.String()] = c

	}
*/
	return di
}

