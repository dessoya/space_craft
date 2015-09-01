
package auth

import (
	"sc/ws/command"
	// "sc/ws/connection"
	"sc/model2"
)

type Command struct {
	connection		command.Connection
	ctx				*command.Context
}

func (c *Command) Execute(message []byte) {

	session := c.connection.GetSession()

	if session == nil || !session.IsAuth {
		return
	}

	session.Update(model2.Fields{
		"UserUUID": nil,
		"IsAuth": false,
		"AuthMethod": nil,
	})

	session.Unlock()
		
	c.connection.Send(`{"command":"reload"}`)
}

func Generator(con command.Connection, ctx *command.Context) command.Command {

	c := Command{ connection: con, ctx: ctx }

	return &c
} 
