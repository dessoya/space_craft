
package auth_session

import (
	"github.com/gocql/gocql"
	"fmt"
	"strings"

	"sc/errors"
	"sc/logger"

	// model_user "sc/models/user"
)

var CQLSession *gocql.Session

func Init(session_ *gocql.Session) {
	CQLSession = session_
}

type Session struct {
	UUID		gocql.UUID
	Exists		bool
	IsAuth		bool
	UserUUID	gocql.UUID
}

func (s *Session) Load() {

	var row = map[string]interface{}{}
	var uuid = s.UUID.String()
	s.Exists = false

	if err := CQLSession.Query(fmt.Sprintf(`SELECT * FROM auth_sessions where session_uuid = %s`, uuid)).MapScan(row); err != nil {
		if err != gocql.ErrNotFound {
			logger.Error(errors.New(err))
		}		
		return
	}

	s.Exists = true

}

func (s *Session) Create() {

	s.Exists = false

	for {
		s.UUID = gocql.TimeUUID()
		var row = map[string]interface{}{}

		var apply bool
		var err error

		if apply, err = CQLSession.Query(fmt.Sprintf(`insert into auth_sessions (session_uuid,last_access,create_time) values (%s,now(),now()) if not exists`, s.UUID)).MapScanCAS(row); err != nil {	
			logger.Error(errors.New(err))
			return
		}

		if apply {
			break
		}
	}

	s.Load()
	
}

func LoadOrCreateSession(uuid string) *Session {

	s := &Session{ }

	var err error
	s.UUID, err = gocql.ParseUUID(uuid)
	if err == nil {
		s.Load()
	}

	if !s.Exists {
		s.Create()
	}

	return s
}

func (s *Session) Update(fields map[string]interface{}) error {

	pairs := []string{}

	for key, value := range fields {
		var pair string = key + " = "
		switch t := value.(type) {
		case bool:
			pair += fmt.Sprintf("%v", t)
		case string:
			pair += "'" + t + "'"
		case gocql.UUID:
			pair += t.String()
		default:
			logger.Error(errors.New(fmt.Sprintf("unknown type: %+v",t)))
		}
		pairs = append(pairs, pair)
	}

	q := "update auth_sessions set " + strings.Join(pairs, ",") + " where session_uuid = " + s.UUID.String()

	if err := CQLSession.Query(q).Exec(); err != nil {	
		logger.Error(errors.New(err))
		return err
	}

	return nil
}