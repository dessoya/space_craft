
package server

import (
	"fmt"
	"crypto/md5"
	"io"

	"sc/logger"
	"sc/errors"
	"time"

	"github.com/gocql/gocql"
)

var CQLSession *gocql.Session

func Init(session_ *gocql.Session) {
	CQLSession = session_
}

type Server struct {
	Exists		bool
	UUID		gocql.UUID
	IP			string
	Port		uint16
}

func Get(UUID gocql.UUID) *Server {

	s := &Server{ UUID: UUID, Exists: false }

	var row = map[string]interface{}{}
	var uuid = UUID.String()

	if err := CQLSession.Query(`SELECT * FROM servers where server_uuid = ?`, uuid).MapScan(row); err != nil {
		if err != gocql.ErrNotFound {
			logger.Error(errors.New(err))
		}		
		return s
	}

	s.Exists		= true
	s.IP			= row["ip"].(string)
	s.Port			= uint16(row["port"].(int))

	t := time.Now().Unix()
	if t - row["live_ts"].(int64) > 10 {
		s.Exists = false
	}

	return s	
}

func New(ip string, port uint16) *Server {

	h := md5.New()
	io.WriteString(h, ip)
	io.WriteString(h, fmt.Sprintf("%d", port))
	uuid, _ := gocql.UUIDFromBytes(h.Sum(nil))

	s := &Server{ IP: ip, Port: port, UUID: uuid }

	return s
}

func (s *Server) Update() {
	// CQLSession.Query(`delete from servers where server_uuid = ?`, s.UUID.String()).Exec()
	CQLSession.Query("insert into servers (server_uuid, ip, port, live_ts) values (?, ?, ?, ?)", s.UUID.String(), s.IP, s.Port, time.Now().Unix()).Exec()
}

var SavePeriod = time.Second * 3

func (s *Server) updater() {

	saveTicker := time.NewTicker(SavePeriod)
	defer func() {
		saveTicker.Stop()		
	}()

	for {
		select {
		case <-saveTicker.C:
			s.Update()


		}
	}
}	

func (s *Server) StartUpdater() {
	go s.updater()
}