
package model2

import(
	"github.com/gocql/gocql"
)

type Fields map[string]interface{}

type InstallInfo struct {
	Init func(*gocql.Session, gocql.UUID)
}

func InstallModels(args ...interface{}) {

	var session *gocql.Session
	var serverUUID gocql.UUID

	for index, arg := range args {

	    switch index {
		case 0:
			session = arg.(*gocql.Session)

		case 1:
			serverUUID = arg.(gocql.UUID)

		default:
			ii := arg.(InstallInfo)
			ii.Init(session, serverUUID)

		}

	}

}