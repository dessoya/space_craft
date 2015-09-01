
package building

import(
	"github.com/gocql/gocql"
	"sync"
	"fmt"
	"strings"
	model "sc/model2"
	"sc/logger"
	"sc/errors"
)

var TableName		= "buildings"
var UUIDFieldName	= "building_uuid"

type Fields struct {
	UUID			gocql.UUID	`cql:"building_uuid"`
	Exists			bool
	IsLock			bool		`cql:"lock"`
	LockServerUUID	gocql.UUID	`cql:"lock_server_uuid"`
	Type			string		`cql:"type"`
	Level			int			`cql:"level"`
	TurnOn			bool		`cql:"turn_on"`
	X				int			`cql:"x"`
	Y				int			`cql:"y"`
}

var Field2CQL = map[string]string{
	"UUID": "building_uuid",
	"IsLock": "lock",
	"LockServerUUID": "lock_server_uuid",
	"Type": "type",
	"Level": "level",
	"TurnOn": "turn_on",
	"X": "x",
	"Y": "y",
}

var LockServerUUID gocql.UUID
var CQLSession *gocql.Session

func Init(session *gocql.Session) {
	CQLSession = session
}

func Load(UUID gocql.UUID) (*Fields, error) {
	m := &Fields{ UUID: UUID }
	err := m.Load()
	return m, err
}

func (m *Fields) Load() (error) {

	m.Exists = false
	var row = map[string]interface{}{}

	if err := CQLSession.Query(`SELECT * FROM buildings where building_uuid = ?`, m.UUID).MapScan(row); err != nil {
		return err
	}
	m.Exists = true

	m.UUID = row["building_uuid"].(gocql.UUID)
	m.IsLock = row["lock"].(bool)
	m.LockServerUUID = row["lock_server_uuid"].(gocql.UUID)
	m.Type = row["type"].(string)
	m.Level = row["level"].(int)
	m.TurnOn = row["turn_on"].(bool)
	m.X = row["x"].(int)
	m.Y = row["y"].(int)

	return nil
}

func Create() (*Fields, error) {
    var err error
	m := &Fields{ Exists: false }
	for {
		m.UUID, err = gocql.RandomUUID()
		if err != nil {
			return nil, err
		}
		var row = map[string]interface{}{}
		var apply bool
		if apply, err = CQLSession.Query(`insert into buildings (building_uuid,create_time) values (?,now()) if not exists`, m.UUID).MapScanCAS(row); err != nil {
			logger.Error(errors.New(err))
			return nil, err
		}
		if apply {
			break
		}
	}
	return m, nil
}

var mutex sync.RWMutex
var Models = map[string]*Fields{}

func Get(UUID gocql.UUID) (*Fields, error) {

    var err error
    uuid := UUID.String()
    mutex.RLock()
	m, ok := Models[uuid]
    mutex.RUnlock()
	if !ok {
		m, err = Load(UUID)
		if err != nil {
			return nil, err
		}
		if m == nil {
			return nil, nil
		}
	    mutex.Lock()
		m2, ok := Models[uuid]
	    if ok {
		    mutex.Unlock()
	    	return m2, nil
	    }
		Models[uuid] = m
	    mutex.Unlock()
	    m.Update(model.Fields{
	    	"IsLock": true,
	    	"LockServerUUID": LockServerUUID,
	    })
	}

	return m, nil

}

func (m *Fields) Lock() (error) {

    uuid := m.UUID.String()
    mutex.Lock()
	m, ok := Models[uuid]
	if ok {
	    mutex.Unlock()
		return m.Load()
	} else {
		Models[uuid] = m
	    mutex.Unlock()
	    m.Update(model.Fields{
	    	"IsLock": true,
	    	"LockServerUUID": LockServerUUID,
	    })
	}

	return nil

}

func (m *Fields) Unlock() error {

    uuid := m.UUID.String()
    mutex.Lock()
	_, ok := Models[uuid]

    if ok {
    	delete(Models, uuid)
    }
    mutex.Unlock()

    err := m.Update(model.Fields{
    	"IsLock": nil,
    	"LockServerUUID": nil,
    })

    return err
}

func (m *Fields) Update(fields model.Fields) error {

	pairs := []string{}

	for key, value := range fields {
	    switch key {
		case "UUID":
			switch value.(type) {
			case nil:
			m.UUID = gocql.UUID{}
			default:
			m.UUID = value.(gocql.UUID)
			}
		case "IsLock":
			switch value.(type) {
			case nil:
			m.IsLock = false
			default:
			m.IsLock = value.(bool)
			}
		case "LockServerUUID":
			switch value.(type) {
			case nil:
			m.LockServerUUID = gocql.UUID{}
			default:
			m.LockServerUUID = value.(gocql.UUID)
			}
		case "Type":
			switch value.(type) {
			case nil:
			m.Type = ""
			default:
			m.Type = value.(string)
			}
		case "Level":
			m.Level = value.(int)
		case "TurnOn":
			switch value.(type) {
			case nil:
			m.TurnOn = false
			default:
			m.TurnOn = value.(bool)
			}
		case "X":
			m.X = value.(int)
		case "Y":
			m.Y = value.(int)
		}
		var pair = Field2CQL[key] + "="
		switch t := value.(type) {
		case nil:
			pair += "null"
		case bool:
			pair += fmt.Sprintf("%v", t)
		case int:
			pair += fmt.Sprintf("%v", t)
		case string:
			pair += "'" + t + "'"
		case *gocql.UUID:
			pair += t.String()
		case []*gocql.UUID:
			a := []string{}
			for _, uuid := range t {
				a = append(a, uuid.String())
			}
			pair += "[" + strings.Join(a, ",") + "]"
		case []gocql.UUID:
			a := []string{}
			for _, uuid := range t {
				a = append(a, uuid.String())
			}
			pair += "[" + strings.Join(a, ",") + "]"
		case gocql.UUID:
			pair += t.String()
		default:
			logger.Error(errors.New(fmt.Sprintf("unknown type: %+v",t)))
		}
		pairs = append(pairs, pair)
	}
	q := "update buildings set " + strings.Join(pairs, ",") + " where building_uuid = " + m.UUID.String()
	logger.String(q)

	if err := CQLSession.Query(q).Exec(); err != nil {
		logger.Error(errors.New(err))
		return err
	}

	return nil

}
