
package auth_session

import (
	"github.com/gocql/gocql"
	"fmt"
)

var CQLSession *gocql.Session

func Init(session_ *gocql.Session) {
	CQLSession = session_
}

type Session struct {
	UUID		gocql.UUID
	Exists		bool
}

func (s *Session) Load() {

	var row = map[string]interface{}{}
	var uuid = s.UUID.String()

	if err = CQLSession.Query(fmt.Sprintf(`SELECT * FROM auth_sessions where session_uuid = %s`, session_uuid)).MapScan(row); err != nil {
		logger.Error(errors.New(err))
	}

}

func (s *Session) Create() {
	
}

func LoadOrCreateSession(uuid gocql.UUID) *Session {

	s := &Session{ UUID: uuid }
	s.Load()
	if !s.Exists {
		s.Create()
	}

	return s
}