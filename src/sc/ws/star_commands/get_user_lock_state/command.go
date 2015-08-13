
package get_user_lock_state

import (	
	"sc/ws/command"
	// "sc/ws/connection"
	"encoding/json"	
	model_user "sc/models/user"

	"fmt"
	"sc/logger"
)

type Command struct {
	connection		command.Connection
	ctx				*command.Context
}

type CommandDetector struct {
	CommandId		int `json:"command_id"`
	UserUUID		string `json:"user_uuid"`
}

func (c *Command) Execute(message []byte) {

	if !c.connection.GetServerAuthState() {
		return
	}

	var commandDetector CommandDetector
	json.Unmarshal(message, &commandDetector)

	var isLock bool = false

	user := model_user.Get(commandDetector.UserUUID)
	logger.String(fmt.Sprintf("commandDetector.UserUUID %+v, user %+v", commandDetector.UserUUID, user))
	if user != nil && user.IsLock {
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