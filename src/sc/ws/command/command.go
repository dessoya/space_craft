
package command

import (
	"github.com/gocql/gocql"
)

type Command interface {
	Execute([]byte)
}

type Connection interface {
	Send(string)
}

type Context struct {
	CQLSession *gocql.Session
}

type Generator func(Connection, *Context) Command