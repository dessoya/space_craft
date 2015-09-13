
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
		
		answer["buildings"] = []map[string]interface{}{
			map[string]interface{}{
				"type": "energy_station",
			},
			map[string]interface{}{
				"type": "mineral_mine",
			},
		}

	}()


	b, _ := json.Marshal(answer)

	c.connection.Send(string(b))
}

func Generator(con command.Connection, ctx *command.Context) command.Command {

	c := Command{ connection: con, ctx: ctx }

	return &c
} 
