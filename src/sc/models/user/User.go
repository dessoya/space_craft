
package user

import (
	"crypto/md5"
	"io"
	"fmt"
	"strings"

	"sc/logger"
	"sc/errors"

	"github.com/gocql/gocql"
)

var CQLSession *gocql.Session

func Init(session_ *gocql.Session) {
	CQLSession = session_
}

type User struct {
	UUID		gocql.UUID
	Exists		bool
	Name		string
}

func New(UUID gocql.UUID) *User {
	u := &User{ UUID: UUID, Exists: false }
	u.Load()
	return u
}

func (u *User) Create() (error) {

	u.Exists = false

	for {
		u.UUID = gocql.TimeUUID()
		var row = map[string]interface{}{}

		var apply bool
		var err error

		if apply, err = CQLSession.Query(fmt.Sprintf(`insert into users (user_uuid,create_time) values (%s,now()) if not exists`, u.UUID)).MapScanCAS(row); err != nil {	
			logger.Error(errors.New(err))
			return err
		}

		if apply {
			break
		}
	}

	return nil
}

func (u *User) Update(fields map[string]interface{}) error {

	pairs := []string{}

	for key, value := range fields {

		var pair string = key + " = "
		switch t := value.(type) {
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

	for key, value := range fields {
		switch key {
		case "fake_uuid", "vk_uuid", "facebook_uuid", "google_uuid":
			method := key[:len(key) - 5]
			q := fmt.Sprintf("insert into idx_users_method_uuid (method, method_uuid, user_uuid) values ('%s', %s, %s)", method, value, u.UUID.String())
			logger.String(q)
			if err := CQLSession.Query(q).Exec(); err != nil {	
				logger.Error(errors.New(err))
				return err
			}
		}
	}	

	q := "update users set " + strings.Join(pairs, ",") + " where user_uuid = " + u.UUID.String()
	logger.String(q)

	if err := CQLSession.Query(q).Exec(); err != nil {	
		logger.Error(errors.New(err))
		return err
	}

	return nil
}

func (u *User) Load() (err error) {

	var row = map[string]interface{}{}

	query := fmt.Sprintf(`select * from users where user_uuid = %s`, u.UUID.String())
	logger.String(query)

	if err = CQLSession.Query(query).MapScan(row); err != nil {

		if err != gocql.ErrNotFound {
			logger.Error(errors.New(err))
		}

		return
	}

	logger.String(fmt.Sprintf("%+v", row))

	u.Exists = true

	u.Name = row["username"].(string)

	return
}

func GetMethodUUID(method string, unique string) *gocql.UUID {

	h := md5.New()
	io.WriteString(h, method)
	io.WriteString(h, unique)
	methodUUID, _ := gocql.UUIDFromBytes(h.Sum(nil))

	return &methodUUID
}

func GetByMethodUUID(method string, methodUUID *gocql.UUID) (*User, error) {

	u := &User{ Exists: false }

	var row = map[string]interface{}{}
	query := fmt.Sprintf(`SELECT * FROM idx_users_method_uuid where method = '%s' and method_uuid = %s`, method, methodUUID.String())
	logger.String(query)
	
	if err := CQLSession.Query(query).MapScan(row); err != nil {
		if err != gocql.ErrNotFound {
			logger.Error(errors.New(err))
		}

		return u, nil
	}

	u.UUID = row["user_uuid"].(gocql.UUID)
	err := u.Load()

	return u, err
}