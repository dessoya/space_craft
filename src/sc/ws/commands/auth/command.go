
package auth

import (
	
	// "sc/logger"
	// "sc/error"

	"github.com/gocql/gocql"
	"encoding/json"

	"sc/ws/command"
	"sc/logger"
	"sc/errors"
	"fmt"
)

type Command struct {
	connection command.Connection
	ctx *command.Context
}

type CommandDetector struct {
	SessionUUID string `json:"session_uuid"`
}

type SendCommandAuth struct {
	Command string `json:"command"`
	SessionUUID string `json:"session_uuid"`
}

func (c *Command) Execute(message []byte) {

	var commandDetector CommandDetector

	json.Unmarshal(message, &commandDetector)

	logger.String(fmt.Sprintf("session_uuid: %+v", commandDetector.SessionUUID))

	sessionUUID, err := gocql.ParseUUID(commandDetector.SessionUUID)
	if err != nil {
		// generate new session
		logger.String("generate new session")
		sessionUUID = gocql.TimeUUID()
	}

	logger.String(fmt.Sprintf("session_uuid: %+v", sessionUUID.String()))

	var row map[string]interface{} = make(map[string]interface{})
	var session_uuid = sessionUUID.String()

	logger.String(fmt.Sprintf(`SELECT * FROM auth_sessions where session_uuid = %s`, session_uuid))
	if err = c.ctx.CQLSession.Query(fmt.Sprintf(`SELECT * FROM auth_sessions where session_uuid = %s`, session_uuid)).MapScan(row); err != nil {
		logger.Error(errors.New(err))
	}

	logger.String(fmt.Sprintf("%+v", row))

	if row["session_uuid"] != sessionUUID {

		logger.String("inserting session")
		if err = c.ctx.CQLSession.Query(fmt.Sprintf(`insert into auth_sessions (session_uuid) values (%s)`, session_uuid)).Exec(); err != nil {
			logger.Error(errors.New(err))
		}
	}


	sendCommandAuth := SendCommandAuth{ Command: "auth", SessionUUID: session_uuid }
	b, err := json.Marshal(sendCommandAuth)
	if err != nil {
		logger.Error(errors.New(err))
	}
	
	c.connection.Send(string(b))
}

func Generator(con command.Connection, ctx *command.Context) command.Command {

	c := Command{ connection: con, ctx: ctx }

	return &c
} 