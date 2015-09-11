
package command

import (
	"github.com/gocql/gocql"
	model2_auth_session "sc/models2/auth_session"
	"sc/config"
	"sc/buildings"
	// "sc/ws/connection/factory"
)

type Command interface {
	Execute([]byte)
}

type Connection interface {
	Send(string)

	SetSession (session *model2_auth_session.Fields)
	GetSession () *model2_auth_session.Fields
	
	SetServerAuthState ()
	GetServerAuthState () bool

	GetRemoteAddr() string
	GetUserAgent() string
}

type Context struct {
	CQLSession		*gocql.Session
	Config			*config.Config
	ServerUUID		gocql.UUID
	Factory			interface{}
	BDispatcher		*buildings.Dispatcher
}

type Generator func(Connection, *Context) Command