
package star_user_logout

import (	
	"sc/ws/command"
	// "sc/ws/connection"
	"sc/ws/connection/factory"
	"encoding/json"	
	// model_user "sc/models/user"
)

type Command struct {
	connection		command.Connection
	ctx				*command.Context
}

type CommandDetector struct {
	CommandId		int `json:"command_id"`
	UserUUID		string `json:"user_uuid"`
	SessionUUID		string `json:"session_uuid"`
}

func (c *Command) Execute(message []byte) {

	if !c.connection.GetServerAuthState() {
		return
	}

	var commandDetector CommandDetector
	json.Unmarshal(message, &commandDetector)


	var f *factory.Factory = c.ctx.Factory.(*factory.Factory)

	conns := f.GetConnections()
	for _, conn := range conns {

		// get connection with user uuid
		s := conn.GetSession()
		if s != nil && s.UUID.String() != commandDetector.SessionUUID && s.IsAuth && s.UserUUID.String() == commandDetector.UserUUID {

			// unauth
			conn.UnAuth()

			// reload
			// conn.Send(`{"command":"reload"}`)
		}
	}

	

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