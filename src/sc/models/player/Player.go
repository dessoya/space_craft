
package player

import (
	// "crypto/md5"
	// "io"

	"sc/logger"
	"sc/errors"

	"sc/model"
	"sc/model2"

	"github.com/gocql/gocql"

	// model_live_planet "sc/models/live_planet"
)

var Players = map[string]*Player{}
// todo: mutex
func Get(UUID string) *Player {
	player, ok := Players[UUID]
	if ok {
		return player
	}

	return nil
}


var CQLSession *gocql.Session

type modelTreater struct { }

func (mt *modelTreater) Get(uuid string) interface{} {
	return Get(uuid)
}

func (mt *modelTreater) New() interface{} {
	return New()
}

func Init(session_ *gocql.Session) {
	CQLSession = session_

	model.Models["player"] = &modelTreater{}
}

type Player struct {
	model.Model

	UserUUID			gocql.UUID		`cql:"user_uuid"`
	CapitalPlanetUUID	gocql.UUID		`cql:"capital_planet_uuid"`
	Planets				[]*gocql.UUID	`cql:"planet_uuid_list"`

}

func (p *Player) PlaceModel() {
	Players[p.UUID.String()] = p
}

func (p *Player) RemoveModel() {
	delete(Players, p.UUID.String())
}


func New() *Player {
	p := &Player{ Model: model.Model{ TableName: "players", UUIDFieldName: "player_uuid" } }
	p.Child = p
	return p
}

func (p *Player) Create() (error) {

	p.Exists = false

	for {
		p.UUID = gocql.TimeUUID()
		var row = map[string]interface{}{}

		var apply bool
		var err error

		if apply, err = CQLSession.Query(`insert into players (player_uuid,create_time) values (?,now()) if not exists`, p.UUID).MapScanCAS(row); err != nil {	
			logger.Error(errors.New(err))
			return err
		}

		if apply {
			break
		}
	}

	return nil
}

/*
func GetPlanet(UUID gocql.UUID) *model_live_planet.LivePlanet {

	lp := model_live_planet.Get(UUID.String())
	if lp == nil {
		lp = model_live_planet.New()
		lp.UUID = UUID
		lp.Load()
		lp.Lock()
		if !lp.Exists {
			return nil
		}
	}

	return lp
}
*/


// -------------------------------------------------------------------------

type Fields struct {
	UserUUID			gocql.UUID		`cql:"user_uuid"`
	CapitalPlanetUUID	gocql.UUID		`cql:"capital_planet_uuid"`
	Planets				[]*gocql.UUID	`cql:"planet_uuid_list"`
}

type ModelTreator struct {

}

func (mt *ModelTreator) New() *model2.Model {
	m := &model2.Model{
		TableName:		"players",
		UUIDFieldName:	"player_uuid",
		Exists:			false,
		Fields:			&Fields{},
	}
	return m
}

var ModelInfo = model2.ModelInfo{ "player", &ModelTreator{} }
