
package session_lock_state 

import (	
	"sc/ws/command"
	// "sc/ws/connection"
	"encoding/json"	

	model_auth_session "sc/models/auth_session"

)

type Command struct {
	connection		command.Connection
	ctx				*command.Context
}

type CommandDetector struct {
	CommandId		int `json:"command_id"`
	SessionUUID		string `json:"session_uuid"`
}

func (c *Command) Execute(message []byte) {

	if !c.connection.GetServerAuthState() {
		return
	}

	var commandDetector CommandDetector
	json.Unmarshal(message, &commandDetector)

	var isLock bool = false

	s := model_auth_session.Get(commandDetector.SessionUUID)
	if s != nil && s.IsLock {
		isLock = true
	}	
	

	b, _ := json.Marshal(map[string]interface{}{
		"command": "answer",
		"command_id": commandDetector.CommandId,
		"is_lock": isLock,
	})

	c.connection.Send(string(b))
}

func Generator(con command.Connection, ctx *command.Context) command.Command {
	c := Command{ connection: con, ctx: ctx }
	return &c
}