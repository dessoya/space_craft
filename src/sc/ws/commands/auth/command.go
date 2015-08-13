
package auth

import (
	
	// "sc/logger"
	// "sc/error"

	"github.com/gocql/gocql"
	"encoding/json"

	"sc/ws/command"
	"sc/logger"
	"sc/errors"

	model_auth_session "sc/models/auth_session"
	model_user "sc/models/user"
	model_server "sc/models/server"
	"sc/model"

	"sc/star"
)

type Command struct {
	connection		command.Connection
	ctx				*command.Context
}

type CommandDetector struct {
	SessionUUID		string `json:"session_uuid"`
	ServerUUID		*string `json:"server_uuid,omitempty"`	
}

type CommandCheckSessionDetector struct {
	IsLock			bool  `json:"is_lock"`
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


	// inner connection auth
	if commandDetector.ServerUUID != nil {
		ServerUUID, _ := gocql.ParseUUID(*commandDetector.ServerUUID)
		// todo: check err
		server := model_server.Get(ServerUUID)
		answer := "error"
		if server.Exists && c.connection.GetRemoteAddr() == server.IP {
			c.connection.SetServerAuthState()
			answer = "ok"
		}

		c.connection.Send(`{"command":"` + answer + `"}`)
		return
	}

	
	// ---------------
	session := model_auth_session.LoadOrCreateSession(commandDetector.SessionUUID, c.connection.GetRemoteAddr(), c.connection.GetUserAgent())

	if session.IsLock {
		b, err := star.Send(session.LockServerUUID, model.Fields{
			"command": "get_session_lock_state",
			"session_uuid": session.UUID,
		})

		var commandCheckSessionDetector CommandCheckSessionDetector
		if err == nil {
			json.Unmarshal(b, &commandCheckSessionDetector)
		}

		if err != nil || commandCheckSessionDetector.IsLock {
			session.Create(c.connection.GetRemoteAddr(), c.connection.GetUserAgent())
		}

		logger.String(string(b))
	}

	session.Lock()
	c.connection.SetSession(session)

	sendCommandAuth := SendCommandAuth{
		Command:			"auth",
		SessionUUID:		session.UUID.String(),
		AuthMethods:		c.ctx.Config.Auth.Methods,
		IsAuth:				session.IsAuth,
	}

	if session.IsAuth {

		user := model_user.New()
		user.UUID = session.UserUUID
		user.Load()

		// check for user lock
		if user.IsLock {

			b, err := star.Send(user.LockServerUUID, model.Fields{
				"command": "get_user_lock_state",
				"user_uuid": user.UUID.String(),
			})

			if err != nil {

			} else {

				type CommandCheckUserDetector struct {
					IsLock			bool  `json:"is_lock"`
				}

				var commandCheckUserDetector CommandCheckUserDetector
				json.Unmarshal(b, &commandCheckUserDetector)

				if commandCheckUserDetector.IsLock {
					
					_, _ = star.Send(user.LockServerUUID, model.Fields{
						"command": "star_user_logout",
						"user_uuid": user.UUID,
						"session_uuid": session.UUID.String(),
					})

					user.Load()
				}

			}
		}

		user.Lock()

		// if user.Exists {
			sendCommandAuth.User = SendCommandAuthUser{
				Name: user.Name,
			}
		// }
	}

	b, err := json.Marshal(sendCommandAuth)
	if err != nil {
		logger.Error(errors.New(err))
		return
	}
	
	c.connection.Send(string(b))
}

func Generator(con command.Connection, ctx *command.Context) command.Command {

	c := Command{ connection: con, ctx: ctx }

	return &c
} 