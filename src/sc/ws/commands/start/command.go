
package start

import (
	"sc/ws/command"
	// "sc/ws/connection"
	"sc/model"
	"sc/model2"
	"github.com/gocql/gocql"

	"fmt"

	"encoding/json"	

	model_user "sc/models2/user"
	model_player "sc/models2/player"
	model_live_planet "sc/models/live_planet"
	model_building "sc/models/building"
	// "sc/logger"

)

type Command struct {
	connection		command.Connection
	ctx				*command.Context
}

type CommandDetector struct {
	CommandId		int `json:"command_id"`
}


func (c *Command) Execute(message []byte) {

	session := c.connection.GetSession()

	if session == nil || !session.IsAuth {
		return
	}

	var commandDetector CommandDetector
	json.Unmarshal(message, &commandDetector)

	user := model_user.Get(session.UserUUID.String())
	if user == nil {
		return
	}

	/*
	player := model_player.New()
	player.Create()
	player.Lock()
	*/

	player, _ := model_player.Create()

	player.Update(model2.Fields{
		"UserUUID": user.UUID,
	})

	user.Update(model.Fields{
		"PlayerUUID": player.UUID,
	})

	// create live planet

	livePlanet := model_live_planet.New()
	livePlanet.Create()

	player.Update(model2.Fields{
		"CapitalPlanetUUID": livePlanet.UUID,
		"Planets": []gocql.UUID{ livePlanet.UUID },
	})

	building := model_building.New()
	building.Create()

	building.Update(model.Fields{
		"Type": "capital",
		"Level": 1,
		"TurnOn": true,
		"X": 0,
		"Y": 0,
	})

	livePlanet.Update(model.Fields{
		"PlayerUUID": player.UUID,
		"Buildings": []*gocql.UUID{ &building.UUID },
	})




	// logger.String(fmt.Sprintf("%+v", user))




	c.connection.Send(fmt.Sprintf(`{"command_id":%d}`, commandDetector.CommandId))
}

func Generator(con command.Connection, ctx *command.Context) command.Command {

	c := Command{ connection: con, ctx: ctx }

	return &c
} 
