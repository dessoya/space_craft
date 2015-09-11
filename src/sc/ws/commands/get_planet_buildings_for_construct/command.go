
package get_planet_buildings_for_construct

import (
	"sc/ws/command"
	"sc/model"

	"encoding/json"	
/*
	model_user "sc/models2/user"
	model_player "sc/models2/player"
	model_live_planet "sc/models2/live_planet"
*/
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

	answer := model.Fields{
		"command_id": commandDetector.CommandId,
	}

	func () {

		/*
		user, _ := model_user.Get(session.UserUUID)
		if user == nil {
			return
		}

		// answer["user"] = true
		
		// player := user.GetPlayer()
		// player := model.Get("player", *user.PlayerUUID).(*model_player.Player)
		player, _ := model_player.Get(*user.PlayerUUID)

		if player == nil {
			return
		}

		// answer["player"] = true

		planet, _ := model_live_planet.Get(player.CapitalPlanetUUID)
		if planet == nil {
			return
		}

		// answer["planet"] = true

		answer["planet_info"] = planet.MakeClientInfo()
		*/
		answer["buildings"] = []string{
			"energy_station",
		}
	}()


	b, _ := json.Marshal(answer)

	c.connection.Send(string(b))
}

func Generator(con command.Connection, ctx *command.Context) command.Command {

	c := Command{ connection: con, ctx: ctx }

	return &c
} 
