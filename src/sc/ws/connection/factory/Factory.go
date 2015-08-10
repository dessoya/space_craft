
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
			err := message.Connection.Write(websocket.TextMessage, []byte(message.Message))
			if err != nil {
				logger.Error(errors.New(err))
				message.Connection.Close()
			}			

		case c := <-f.closeChannel:
			delete(f.connections, c.Id)

		case <-pingTicker.C:
			var cs map[uint32]*connection.Connection = make(map[uint32]*connection.Connection)

			f.mutex.Lock()
			for index, connection := range f.connections {
				cs[index] = connection
			}
			f.mutex.Unlock()

			for _, connection := range f.connections {
				err := connection.Ping()
				if err != nil {
					connection.Close()
				}
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


func (f *Factory) MakeDebugInfo () *DebugInfo {

	di := DebugInfo{ Connections: make(map[string]interface{}), WriteChannelSize: len(f.writeChannel) }

	f.mutex.Lock()
	for index, _ := range f.connections {
		di.Connections[fmt.Sprintf("%d", index)] = make(map[string]interface{})
	}
	f.mutex.Unlock()

	return &di
}

