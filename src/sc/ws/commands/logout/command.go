
package auth

import (
	"sc/ws/command"
	"sc/ws/connection"
	"sc/model"
)

type Command struct {
	connection		command.Connection
	ctx				*command.Context
}

func (c *Command) Execute(message []byte) {

	c.connection.(*connection.Connection).Session.Update(model.Fields{
		"UserUUID": nil,
		"IsAuth": false,
		"AuthMethod": nil,
	})
		
	c.connection.Send(`{"command":"reload"}`)
}

func Generator(con command.Connection, ctx *command.Context) command.Command {

	c := Command{ connection: con, ctx: ctx }

	return &c
} 
