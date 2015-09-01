
package model2

import(
	// "github.com/gocql/gocql"
)

type Fields map[string]interface{}

/*
type IFields interface {

}

type Model struct {
	
	TableName		string
	UUIDFieldName	string

	UUID			gocql.UUID
	Exists			bool

	IsLock			bool
	LockServerUUID	gocql.UUID

	Fields			IFields
}

func (m *Model) Get() interface{} {

	return m.Fields
}

func (m *Model) Create() *Model {

	return nil
}

func (m *Model) Load() {

}

func (m *Model) Lock() {

}

func (m *Model) Unlock() {
	
}

type ModelTreator interface {
	New() *Model
}

type ModelInfo struct {
	Name		string
	Treator		ModelTreator
}

var Models = map[string]ModelTreator{}
var CQLSession *gocql.Session

func InstallModels(models []ModelInfo, session *gocql.Session) {
	CQLSession = session	
	for _, model := range models {
		Models[model.Name] = model.Treator
	}
}


func New(name string) *Model {

	t, ok := Models[name]
	if !ok {
		return nil
	}

	return t.New()
}

func Create(name string) *Model {

	t, ok := Models[name]
	if !ok {
		return nil
	}

	m := t.New()
	m.Create()

	return m
}

func Load(name string, uuid gocql.UUID) *Model {

	t, ok := Models[name]
	if !ok {
		return nil
	}

	m := t.New()
	m.UUID = uuid
	m.Load()

	return m
}

*/
