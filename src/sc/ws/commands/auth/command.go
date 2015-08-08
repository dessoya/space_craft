
package auth

import (
	
	// "sc/logger"
	// "sc/error"

	// "github.com/gocql/gocql"
	"encoding/json"

	"sc/ws/command"
	"sc/logger"
	"sc/errors"

	model_auth_session "sc/models/auth_session"
	model_user "sc/models/user"	
)

type Command struct {
	connection		command.Connection
	ctx				*command.Context
}

type CommandDetector struct {
	SessionUUID		string `json:"session_uuid"`
}

type SendCommandAuthUser struct {
	Name		string		`json:"name"`
}

type SendCommandAuth struct {
	Command			string		`json:"command"`
	SessionUUID		string		`json:"session_uuid"`
	IsAuth			bool		`json:"is_auth"`
	AuthMethods		[]string	`json:"auth_methods"`
	User			SendCommandAuthUser `json:"user"`
}

func (c *Command) Execute(message []byte) {

	var commandDetector CommandDetector
	json.Unmarshal(message, &commandDetector)

	session := model_auth_session.LoadOrCreateSession(commandDetector.SessionUUID)

	sendCommandAuth := SendCommandAuth{
		Command:			"auth",
		SessionUUID:		session.UUID.String(),
		AuthMethods:		c.ctx.Config.Auth.Methods,
		IsAuth:				session.IsAuth,
	}

	if session.IsAuth {
		user := model_user.New(session.UserUUID)
		if user.Exists {
			sendCommandAuth.User = SendCommandAuthUser{
				Name: user.Name,
			}
		}
	}

	b, err := json.Marshal(sendCommandAuth)
	if err != nil {
		logger.Error(errors.New(err))
		return
	}
	
	c.connection.SetSession(session)
	c.connection.Send(string(b))
}

func Generator(con command.Connection, ctx *command.Context) command.Command {

	c := Command{ connection: con, ctx: ctx }

	return &c
} 