
package build

import (
	"sc/ws/command"
	// "sc/ws/connection"
	// "sc/model2"

	// "github.com/gocql/gocql"
	// "fmt"

	"encoding/json"	

	model_user "sc/models2/user"
	model_player "sc/models2/player"
	model_live_planet "sc/models2/live_planet"
	// model_building "sc/models2/building"

	// model_live_planet "sc/models/live_planet"
	// model_building "sc/models/building"
	// "sc/logger"

)

type Command struct {
	connection		command.Connection
	ctx				*command.Context
}

type CommandDetector struct {
	CommandId		int			`json:"command_id"`
	Building		string		`json:"building"`
	X				int			`json:"x"`
	Y				int			`json:"y"`
}


func (c *Command) Execute(message []byte) {

	session := c.connection.GetSession()

	if session == nil || !session.IsAuth {
		return
	}

	var commandDetector CommandDetector
	json.Unmarshal(message, &commandDetector)

	/*
	answer := model2.Fields{
		"command_id": commandDetector.CommandId,
	}
	*/

	func () {

		user, _ := model_user.Get(session.UserUUID)
		if user == nil {
			return
		}

		player, _ := model_player.Get(*user.PlayerUUID)
		if player == nil {
			return
		}

		planet, _ := model_live_planet.Get(player.CapitalPlanetUUID)
		if planet == nil {
			return
		}

		c.ctx.BDispatcher.Build(&planet.UUID, commandDetector.Building, int(commandDetector.X), int(commandDetector.Y))
		
	}()

}

func Generator(con command.Connection, ctx *command.Context) command.Command {

	c := Command{ connection: con, ctx: ctx }

	return &c
} 
