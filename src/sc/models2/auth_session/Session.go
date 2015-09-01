
package auth_session

import(
	"github.com/gocql/gocql"
	"sync"
	"fmt"
	"strings"
	model "sc/model2"
	"sc/logger"
	"sc/errors"
)

var TableName		= "auth_sessions"
var UUIDFieldName	= "session_uuid"

type Fields struct {
	UUID			gocql.UUID	`cql:"session_uuid"`
	Exists			bool
	IsLock			bool		`cql:"lock"`
	LockServerUUID	gocql.UUID	`cql:"lock_server_uuid"`
	IsAuth			bool		`cql:"is_auth"`
	AuthMethod		string		`cql:"auth_method"`
	UserUUID		gocql.UUID	`cql:"user_uuid"`
	RemoteAddr		string		`cql:"remote_addr"`
	UserAgent		string		`cql:"user_agent"`
}

var Field2CQL = map[string]string{
	"UUID": "session_uuid",
	"IsLock": "lock",
	"LockServerUUID": "lock_server_uuid",
	"IsAuth": "is_auth",
	"AuthMethod": "auth_method",
	"UserUUID": "user_uuid",
	"RemoteAddr": "remote_addr",
	"UserAgent": "user_agent",
}

var InstallInfo = model.InstallInfo{ Init: Init }
var LockServerUUID gocql.UUID
var CQLSession *gocql.Session

func Init(session *gocql.Session, serverUUID gocql.UUID) {
	LockServerUUID = serverUUID
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

	if err := CQLSession.Query(`SELECT * FROM auth_sessions where session_uuid = ?`, m.UUID).MapScan(row); err != nil {
		return err
	}
	m.Exists = true

	m.UUID = row["session_uuid"].(gocql.UUID)
	m.IsLock = row["lock"].(bool)
	m.LockServerUUID = row["lock_server_uuid"].(gocql.UUID)
	m.IsAuth = row["is_auth"].(bool)
	m.AuthMethod = row["auth_method"].(string)
	m.UserUUID = row["user_uuid"].(gocql.UUID)
	m.RemoteAddr = row["remote_addr"].(string)
	m.UserAgent = row["user_agent"].(string)

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
		if apply, err = CQLSession.Query(`insert into auth_sessions (session_uuid,create_time) values (?,now()) if not exists`, m.UUID).MapScanCAS(row); err != nil {
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
		case "IsAuth":
			switch value.(type) {
			case nil:
			m.IsAuth = false
			default:
			m.IsAuth = value.(bool)
			}
		case "AuthMethod":
			switch value.(type) {
			case nil:
			m.AuthMethod = ""
			default:
			m.AuthMethod = value.(string)
			}
		case "UserUUID":
			switch value.(type) {
			case nil:
			m.UserUUID = gocql.UUID{}
			default:
			m.UserUUID = value.(gocql.UUID)
			}
		case "RemoteAddr":
			switch value.(type) {
			case nil:
			m.RemoteAddr = ""
			default:
			m.RemoteAddr = value.(string)
			}
		case "UserAgent":
			switch value.(type) {
			case nil:
			m.UserAgent = ""
			default:
			m.UserAgent = value.(string)
			}
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
	q := "update auth_sessions set " + strings.Join(pairs, ",") + " where session_uuid = " + m.UUID.String()
	logger.String(q)

	if err := CQLSession.Query(q).Exec(); err != nil {
		logger.Error(errors.New(err))
		return err
	}

	return nil

}
