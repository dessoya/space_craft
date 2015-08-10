
package session_lock_state 

import (	
	"sc/ws/command"
	"encoding/json"	
)

type Command struct {
	connection		command.Connection
	ctx				*command.Context
}

type CommandDetector struct {
	CommandId		int `json:"command_id"`
}

func (c *Command) Execute(message []byte) {

	if !c.connection.GetServerAuthState() {
		return
	}

	var commandDetector CommandDetector
	json.Unmarshal(message, &commandDetector)

	b, _ := json.Marshal(map[string]interface{}{
		"command": "answer",
		"command_id": commandDetector.CommandId,
	})

	c.connection.Send(string(b))
}

func Generator(con command.Connection, ctx *command.Context) command.Command {
	c := Command{ connection: con, ctx: ctx }
	return &c
}