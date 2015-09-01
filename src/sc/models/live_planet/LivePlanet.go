
package live_planet

import (
	// "crypto/md5"
	// "io"

	"sc/logger"
	"sc/errors"

	"sc/model"

	"github.com/gocql/gocql"

	model_building "sc/models/building"
)

var LivePlanets = map[string]*LivePlanet{}
// todo: mutex
func Get(UUID string) *LivePlanet {
	livePlanet, ok := LivePlanets[UUID]
	if ok {
		return livePlanet
	}

	return nil
}


var CQLSession *gocql.Session

func Init(session_ *gocql.Session) {
	CQLSession = session_
}

type LivePlanet  struct {
	model.Model

	PlayerUUID		gocql.UUID	`cql:"owner_player_uuid"`	
	Buildings		[]*gocql.UUID	`cql:"buildings_list"`	
}

func (lp *LivePlanet) PlaceModel() {
	LivePlanets[lp.UUID.String()] = lp
}

func (lp *LivePlanet) RemoveModel() {
	delete(LivePlanets, lp.UUID.String())
}


func New() *LivePlanet {
	lp := &LivePlanet{ Model: model.Model{ TableName: "live_planets", UUIDFieldName: "planet_uuid" } }
	lp.Child = lp
	return lp
}

func (lp *LivePlanet) Create() (error) {

	lp.Exists = false

	for {
		lp.UUID = gocql.TimeUUID()
		var row = map[string]interface{}{}

		var apply bool
		var err error

		if apply, err = CQLSession.Query(`insert into live_planets (planet_uuid,create_time) values (?,now()) if not exists`, lp.UUID).MapScanCAS(row); err != nil {	
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
func GetBuilding(uuid gocql.UUID) *model_building.Building {

	b := model_building.Get(uuid.String())
	if b == nil {
		b := model_building.New()
		b.UUID = uuid
		b.Load()
		b.Lock()
		if !b.Exists {
			return
		}
	}	

	return b
}
*/

func (lp *LivePlanet) MakeClientInfo() (info model.Fields) {

	info["buildings"] = []interface{}{}

	for _, uuid := range lp.Buildings {
		_ = model.Get("building", *uuid).(*model_building.Building)
	}
	

	return
}