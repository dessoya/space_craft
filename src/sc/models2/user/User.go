
package user

import(
	"crypto/md5"
	"io"
	"github.com/gocql/gocql"
	"sync"
	"fmt"
	"strings"
	"strconv"
	model "sc/model2"
	"sc/logger"
	"sc/errors"
)

var TableName		= "users"
var UUIDFieldName	= "user_uuid"

type Fields struct {
	UUID			gocql.UUID	`cql:"user_uuid"`
	Exists			bool
	IsLock			bool		`cql:"lock"`
	LockServerUUID	gocql.UUID	`cql:"lock_server_uuid"`
	Name			string		`cql:"username"`
	SectionName		string		`cql:"section"`
	PlayerUUID		*gocql.UUID	`cql:"player_uuid"`
}


func GetMethodUUID(method string, unique string) gocql.UUID {

	h := md5.New()
	io.WriteString(h, method)
	io.WriteString(h, unique)
	methodUUID, _ := gocql.UUIDFromBytes(h.Sum(nil))

	return methodUUID
}


func (u *Fields) AddMethod(method string, unique string) error {

	methodUUID := GetMethodUUID(method, unique)

	if err := CQLSession.Query(`insert into idx_users_method_uuid (method, method_uuid, user_uuid) values (?,?,?)`, method, methodUUID, u.UUID).Exec(); err != nil {	
		logger.Error(errors.New(err))
		return err
	}

	return nil
}

func GetByMethod(method string, unique string) (*Fields, error) {

	methodUUID := GetMethodUUID(method, unique)
	u := &Fields{ }

	var row = model.Fields{}
	if err := CQLSession.Query(`SELECT * FROM idx_users_method_uuid where method = ? and method_uuid = ?`, method, methodUUID).MapScan(row); err != nil {
		if err != gocql.ErrNotFound {
			logger.Error(errors.New(err))
		}

		return u, nil
	}

	u.UUID = row["user_uuid"].(gocql.UUID)
	err := u.Load()

	return u, err
}
var Field2CQL = map[string]string{
	"UUID": "user_uuid",
	"IsLock": "lock",
	"LockServerUUID": "lock_server_uuid",
	"Name": "username",
	"SectionName": "section",
	"PlayerUUID": "player_uuid",
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

	if err := CQLSession.Query(`SELECT * FROM users where user_uuid = ?`, m.UUID).MapScan(row); err != nil {
		return err
	}
	m.Exists = true

	m.UUID = row["user_uuid"].(gocql.UUID)
	m.IsLock = row["lock"].(bool)
	m.LockServerUUID = row["lock_server_uuid"].(gocql.UUID)
	m.Name = row["username"].(string)
	m.SectionName = row["section"].(string)
	v6 := row["player_uuid"].(gocql.UUID)
	logger.String(fmt.Sprintf("PlayerUUID: %+v", v6))
	if v6.String() == "00000000-0000-0000-0000-000000000000" { m.PlayerUUID = nil
	} else { m.PlayerUUID = &v6}

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
		if apply, err = CQLSession.Query(`insert into users (user_uuid,create_time) values (?,now()) if not exists`, m.UUID).MapScanCAS(row); err != nil {
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

func Access(uuid string) (*Fields) {

    mutex.RLock()
	m, ok := Models[uuid]
    mutex.RUnlock()
    if ok {
    	return m
    }
    return nil
}

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
		case "Name":
			switch value.(type) {
			case nil:
			m.Name = ""
			default:
			m.Name = value.(string)
			}
		case "SectionName":
			switch value.(type) {
			case nil:
			m.SectionName = ""
			default:
			m.SectionName = value.(string)
			}
		case "PlayerUUID":
			switch t := value.(type) {
			case nil:
			m.PlayerUUID = nil
			case gocql.UUID:
			m.PlayerUUID = &t
			default:
			m.PlayerUUID = value.(*gocql.UUID)
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
		case float64:
			pair += fmt.Sprintf("%v", t)
		case int64:
			pair += fmt.Sprintf("%v", t)
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
		case []string:
			a := []string{}
			for _, s := range t {
				a = append(a, `'` + s + `'`)
			}
			pair += "[" + strings.Join(a, ",") + "]"
		case []int:
			a := []string{}
			for _, i := range t {
				a = append(a, strconv.Itoa(i))
			}
			pair += "[" + strings.Join(a, ",") + "]"
		case gocql.UUID:
			pair += t.String()
		default:
			logger.Error(errors.New(fmt.Sprintf("unknown type: %+v",t)))
		}
		pairs = append(pairs, pair)
	}
	q := "update users set " + strings.Join(pairs, ",") + " where user_uuid = " + m.UUID.String()
	logger.String(q)

	if err := CQLSession.Query(q).Exec(); err != nil {
		logger.Error(errors.New(err))
		return err
	}

	return nil

}

func GetLockedModels() ([]string) {
	keys := []string{}
    mutex.RLock()
    for key, _ := range Models {
    	keys = append(keys, key)
    }
    mutex.RUnlock()
	return keys
}
