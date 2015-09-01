
package get_user_lock_state

import (	
	"github.com/gocql/gocql"
	"sc/ws/command"
	// "sc/ws/connection"
	"encoding/json"	
	model_user "sc/models2/user"

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

	userUUID, err := gocql.ParseUUID(commandDetector.UserUUID)
	if err != nil {


	} else {

		user, err := model_user.Get(userUUID)
		logger.String(fmt.Sprintf("commandDetector.UserUUID %+v, user %+v", commandDetector.UserUUID, user))
		if err == nil && user != nil && user.IsLock {
			isLock = true
		}	
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