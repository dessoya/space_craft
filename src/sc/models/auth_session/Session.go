
package auth_session

import (
	"github.com/gocql/gocql"
	"fmt"

	"sc/errors"
	"sc/logger"

	"sc/model"

	// model_user "sc/models/user"
)

var Sessions = map[string]*Session{}
// todo: mutex
func Get(UUID string) *Session {
	session, ok := Sessions[UUID]
	if ok {
		return session
	}

	return nil
}


var CQLSession *gocql.Session

func Init(session_ *gocql.Session) {
	CQLSession = session_
}

type Session struct {

	model.Model

	IsAuth			bool			`cql:"is_auth"`
	AuthMethod		string			`cql:"auth_method"`
	UserUUID		gocql.UUID		`cql:"user_uuid"`
	RemoteAddr		string			`cql:"remote_addr"`
	UserAgent		string			`cql:"user_agent"`
}


func (s *Session) PlaceModel() {
	Sessions[s.UUID.String()] = s
}

func (s *Session) RemoveModel() {
	delete(Sessions, s.UUID.String())
}

func New() *Session {

	s := &Session{ Model: model.Model{ TableName: "auth_sessions", UUIDFieldName: "session_uuid" } }
	s.Child = s

	return s
}


func (s *Session) Create(remoteAddr string, userAgent string) {

	s.Exists = false

	for {
		s.UUID = gocql.TimeUUID()
		var row = model.Fields{}

		var apply bool
		var err error

		if apply, err = CQLSession.Query(`insert into auth_sessions (session_uuid,last_access,create_time, remote_addr, user_agent) values (?,now(),now(),?,?) if not exists`, s.UUID, remoteAddr, userAgent).MapScanCAS(row); err != nil {	
			logger.Error(errors.New(err))
			return
		}

		if apply {
			break
		}
	}

	s.Load()
	
}

func LoadOrCreateSession(uuid string, remoteAddr string, userAgent string) *Session {

	s := New()

	var err error
	s.UUID, err = gocql.ParseUUID(uuid)
	if err == nil {
		s.Load()
	}

	if !s.Exists || (len(remoteAddr) > 0 && remoteAddr != s.RemoteAddr) || userAgent != s.UserAgent {
		logger.String(fmt.Sprintf("remoteAddr %s %s, userAgent %s %s", remoteAddr, s.RemoteAddr, userAgent, s.UserAgent))
		s.Create(remoteAddr, userAgent)
	}

	return s
}
