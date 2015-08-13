
package model

import (
	"github.com/gocql/gocql"
	"fmt"
	"strings"
	"sc/logger"
	"sc/errors"
	"sync"

	r "reflect"
)


type Fields map[string]interface{}

var localServerUUID gocql.UUID
var CQLSession *gocql.Session

func Init(UUID gocql.UUID, session *gocql.Session) {
	localServerUUID = UUID
	CQLSession = session
}

type IModel interface {
	PlaceModel()
	RemoveModel()
}

type Model struct {
	
	TableName		string
	UUIDFieldName	string

	UUID			gocql.UUID
	Exists			bool

	IsLock			bool
	LockServerUUID	gocql.UUID

	Child			IModel
}


var modelMapMutex sync.RWMutex
var modelMap = make(map[r.Type]*ModelInfo)

type FieldInfo struct {
	ModelField string
	CQLField string
	Num int
	Zero interface{}
}

type ModelInfo struct {
	ModelFields map[string]FieldInfo
	CQLFields map[string]FieldInfo
}

func getModelInfo(val interface{}) *ModelInfo {
	v := r.Indirect(r.ValueOf(val))
	st := r.Indirect(v).Type()
	modelMapMutex.RLock()
	sinfo, found := modelMap[st]
	modelMapMutex.RUnlock()
	if found {
		return sinfo
	}

	n := st.NumField()
	fieldsMap := make(map[string]FieldInfo)
	cqlMap := make(map[string]FieldInfo)
	for i := 0; i != n; i++ {
		field := st.Field(i)
		var tag = field.Tag.Get("cql")
		if tag == "" {
			continue
		}
		info := FieldInfo{Num: i, ModelField: field.Name, CQLField: tag, Zero: r.Zero(v.Field(i).Type()).Interface() }
		fieldsMap[field.Name] = info
		cqlMap[tag] = info
	}

	sinfo = &ModelInfo{
		fieldsMap,
		cqlMap,
	}

	modelMapMutex.Lock()
	modelMap[st] = sinfo
	modelMapMutex.Unlock()
	return sinfo
}


func (m *Model) Update(fields Fields) error {

	pairs := []string{}

	modelInfo := getModelInfo(m.Child)

	// m.Child.UpdateFields(fields)

	val := r.Indirect(r.ValueOf(m.Child))

	// logger.String(fmt.Sprintf("%+v", modelInfo))
	// logger.String(fmt.Sprintf("%+v", fields))

	for key, value := range fields {
		
		var inner bool
		var innerKey string
		inner = false
		innerKey = "none"

		switch key {
		case "IsLock":
			innerKey = "lock"
			inner = true
			if value == nil {
				m.IsLock = false
			} else {
				m.IsLock = value.(bool)
			}
		case "LockServerUUID":
			innerKey = "lock_server_uuid"
			inner = true
			switch t := value.(type) {
			case *gocql.UUID:
				m.LockServerUUID = *t
			case gocql.UUID:
				m.LockServerUUID = t
			}
		}

		var structField r.Value
		var info FieldInfo
		var ok bool

		if !inner {
			if info, ok = modelInfo.ModelFields[key]; ok {
				structField = val.Field(info.Num)
				switch value.(type) {
				case nil:
					structField.Set(r.ValueOf(info.Zero))
				default:
					structField.Set(r.ValueOf(value))
				}
			} else {
				continue
			}
		}

		var pair string
		if inner {
			pair = innerKey + " = "
		} else {
			pair = info.CQLField + " = "
		}

		switch t := value.(type) {
		case nil:
			pair += "null"
		case bool:
			pair += fmt.Sprintf("%v", t)
		case string:
			pair += "'" + t + "'"
		case *gocql.UUID:
			pair += t.String()
		case gocql.UUID:
			pair += t.String()
		default:
			logger.Error(errors.New(fmt.Sprintf("unknown type: %+v",t)))
		}
		pairs = append(pairs, pair)
	}

	q := "update " + m.TableName + " set " + strings.Join(pairs, ",") + " where " + m.UUIDFieldName + " = " + m.UUID.String()
	logger.String(q)

	if err := CQLSession.Query(q).Exec(); err != nil {	
		logger.Error(errors.New(err))
		return err
	}

	// logger.String(fmt.Sprintf("%+v", m.Child))

	return nil
}

func (m *Model) Load() error {

	var row = Fields{}
	m.Exists = false

	if err := CQLSession.Query(`SELECT * FROM ` + m.TableName + ` where ` + m.UUIDFieldName + ` = ?`, m.UUID).MapScan(row); err != nil {
		if err != gocql.ErrNotFound {
			logger.Error(errors.New(err))
			return err
		}		
		return nil
	}

	m.IsLock			= row["lock"].(bool)
	m.LockServerUUID	= row["lock_server_uuid"].(gocql.UUID)

	// m.Child.LoadFromMap(row)

	modelInfo := getModelInfo(m.Child)
	val := r.Indirect(r.ValueOf(m.Child))

	for key, value := range row {

		var structField r.Value
		if info, ok := modelInfo.CQLFields[key]; ok {
			structField = val.Field(info.Num)
			structField.Set(r.ValueOf(value))
		}

	}

	// logger.String(fmt.Sprintf("%+v", m.Child))

	m.Exists = true
	return nil
}


// todo: lock mutex

func (m *Model) Lock() {
	m.IsLock = true
	m.Child.PlaceModel()
	m.Update(Fields{
		"IsLock": true,
		"LockServerUUID": localServerUUID,
	})
}

func (m *Model) Unlock() {
	m.IsLock = false
	m.Child.RemoveModel()
	m.Update(Fields{
		"IsLock": nil,
		"LockServerUUID": nil,
	})
}
