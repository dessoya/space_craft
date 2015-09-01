
package set_section

import (
	"encoding/json"

	"sc/ws/command"
	// "sc/ws/connection"
	"sc/model2"
	model_user "sc/models2/user"
)

type Command struct {
	connection		command.Connection
	ctx				*command.Context
}

type CommandDetector struct {
	SectionName		string `json:"section"`
}

func (c *Command) Execute(message []byte) {

	session := c.connection.GetSession()

	if session == nil || !session.IsAuth {
		return
	}

	var commandDetector CommandDetector
	json.Unmarshal(message, &commandDetector)
	
	user, _ := model_user.Get(session.UserUUID)
	if user == nil {
		user, _ = model_user.Create()
		user.UUID = session.UserUUID
		user.Load()
	}

	user.Update(model2.Fields{
		"SectionName": commandDetector.SectionName,
	})

}

func Generator(con command.Connection, ctx *command.Context) command.Command {

	c := Command{ connection: con, ctx: ctx }

	return &c
} 
