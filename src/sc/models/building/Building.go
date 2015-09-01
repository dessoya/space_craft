
package building

import (
	// "crypto/md5"
	// "io"

	"sc/logger"
	"sc/errors"

	"sc/model"

	"github.com/gocql/gocql"
)

var Buildings = map[string]*Building{}
// todo: mutex
func Get(UUID string) *Building {
	building, ok := Buildings[UUID]
	if ok {
		return building
	}

	return nil
}


var CQLSession *gocql.Session

func Init(session_ *gocql.Session) {
	CQLSession = session_
}

type Building  struct {
	model.Model

	Type	string	`cql:"type"`
	Level	int		`cql:"level"`
	TurnOn	bool	`cql:"turn_on"`
	X		int		`cql:"x"`
	Y		int		`cql:"y"`


}

func (b *Building) PlaceModel() {
	Buildings[b.UUID.String()] = b
}

func (b *Building) RemoveModel() {
	delete(Buildings, b.UUID.String())
}


func New() *Building {
	b := &Building{ Model: model.Model{ TableName: "buildings", UUIDFieldName: "building_uuid" } }
	b.Child = b
	return b
}

func (b *Building) Create() (error) {

	b.Exists = false

	for {
		b.UUID = gocql.TimeUUID()
		var row = map[string]interface{}{}

		var apply bool
		var err error

		if apply, err = CQLSession.Query(`insert into buildings (building_uuid,create_time) values (?,now()) if not exists`, b.UUID).MapScanCAS(row); err != nil {	
			logger.Error(errors.New(err))
			return err
		}

		if apply {
			break
		}
	}

	return nil
}
