
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
	UUID			gocql.UUID
	Exists			bool
	IsAuth			bool
	UserUUID		gocql.UUID
	RemoteAddr		string
	UserAgent		string
	IsLock			bool
	LockServerUUID	gocql.UUID
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

	s.IsAuth			= row["is_auth"].(bool)
	s.UserUUID			= row["user_uuid"].(gocql.UUID)
	s.RemoteAddr		= row["remote_addr"].(string)
	s.UserAgent			= row["user_agent"].(string)
	s.IsLock			= row["lock"].(bool)
	s.LockServerUUID	= row["lock_server_uuid"].(gocql.UUID)

	s.Exists = true

}

func (s *Session) Create(remoteAddr string, userAgent string) {

	s.Exists = false

	for {
		s.UUID = gocql.TimeUUID()
		var row = map[string]interface{}{}

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

	s := &Session{ }

	var err error
	s.UUID, err = gocql.ParseUUID(uuid)
	if err == nil {
		s.Load()
	}

	if !s.Exists || remoteAddr != s.RemoteAddr || userAgent != s.UserAgent {
		s.Create(remoteAddr, userAgent)
	}

	return s
}

func (s *Session) Update(fields map[string]interface{}) error {

	pairs := []string{}

	for key, value := range fields {
		var pair string = key + " = "
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

	q := "update auth_sessions set " + strings.Join(pairs, ",") + " where session_uuid = " + s.UUID.String()

	if err := CQLSession.Query(q).Exec(); err != nil {	
		logger.Error(errors.New(err))
		return err
	}

	return nil
}