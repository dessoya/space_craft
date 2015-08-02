
package command

import (
	"github.com/gocql/gocql"
	model_auth_session "sc/models/auth_session"
)

type Command interface {
	Execute([]byte)
}

type Connection interface {
	Send(string)
	SetSession (session *model_auth_session.Session)
}

type Context struct {
	CQLSession *gocql.Session
}

type Generator func(Connection, *Context) Command