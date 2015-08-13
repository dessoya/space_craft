
package user

import (
	"crypto/md5"
	"io"

	"sc/logger"
	"sc/errors"

	"sc/model"

	"github.com/gocql/gocql"
)

var Users = map[string]*User{}
// todo: mutex
func Get(UUID string) *User {
	user, ok := Users[UUID]
	if ok {
		return user
	}

	return nil
}

/*
func Get(UUID goclq.UUID) *User {
	user, ok := Users[UUID.String()]
	if ok {
		return user
	}

	return nil
}
*/

var CQLSession *gocql.Session

func Init(session_ *gocql.Session) {
	CQLSession = session_
}

type User struct {
	model.Model

	Name		string		`cql:"username"`
}

func (u *User) PlaceModel() {
	Users[u.UUID.String()] = u
}

func (u *User) RemoveModel() {
	delete(Users, u.UUID.String())
}


func New() *User {
	u := &User{ Model: model.Model{ TableName: "users", UUIDFieldName: "user_uuid" } }
	u.Child = u
	return u
}

func (u *User) Create() (error) {

	u.Exists = false

	for {
		u.UUID = gocql.TimeUUID()
		var row = map[string]interface{}{}

		var apply bool
		var err error

		if apply, err = CQLSession.Query(`insert into users (user_uuid,create_time) values (?,now()) if not exists`, u.UUID).MapScanCAS(row); err != nil {	
			logger.Error(errors.New(err))
			return err
		}

		if apply {
			break
		}
	}

	return nil
}

func GetMethodUUID(method string, unique string) gocql.UUID {

	h := md5.New()
	io.WriteString(h, method)
	io.WriteString(h, unique)
	methodUUID, _ := gocql.UUIDFromBytes(h.Sum(nil))

	return methodUUID
}


func (u *User) AddMethod(method string, unique string) error {

	methodUUID := GetMethodUUID(method, unique)

	if err := CQLSession.Query(`insert into idx_users_method_uuid (method, method_uuid, user_uuid) values (?,?,?)`, method, methodUUID, u.UUID).Exec(); err != nil {	
		logger.Error(errors.New(err))
		return err
	}

	return nil
}

func GetByMethod(method string, unique string) (*User, error) {

	methodUUID := GetMethodUUID(method, unique)
	u := New()

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