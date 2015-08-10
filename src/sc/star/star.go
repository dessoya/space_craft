
package star

import (	
	"github.com/gorilla/websocket"
	"github.com/gocql/gocql"
	"fmt"
	"sync"
	"net/url"
	"net"
	"net/http"
	"time"
	"encoding/json"
	"io"

	"sc/logger"
	"sc/errors"
	"sc/ws/command"

	model_auth_session "sc/models/auth_session"
	model_server "sc/models/server"
)

const (
	WriteWait		= 10 * time.Second
	PongWait		= 20 * time.Second
	PingPeriod		= (PongWait * 9) / 10
	MaxMessageSize	= 4096
)

type Server struct {
	isLocal			bool
	ws				*websocket.Conn
}

type Info struct {
	ReturnChan		chan []byte
	CommandId		int
}

type LocalConnection struct {

}

type CommandDetector struct {
	Command string `json:"command"`
	CommandId float64 `json:"command_id"`
}



var servers map[string]*Server = make(map[string]*Server)
var localServer *model_server.Server



var readChannel chan *Info = make(chan *Info, 1024)
var localConnection = &LocalConnection{}



var answers map[int]chan []byte = make(map[int]chan []byte)



var localAnswers map[int][]byte = make(map[int][]byte)
var commands map[string]command.Generator
var commandContext *command.Context



var mutexCommandId = &sync.Mutex{}
var commandIdIterator = 1




func (l *LocalConnection) Send(message string) {

	logger.String(fmt.Sprintf("local send: %+v", message))

	commandDetector := CommandDetector{}
	err := json.Unmarshal([]byte(message), &commandDetector)
	if err != nil {
		logger.Error(errors.New(err))
		return
	}

	logger.String(fmt.Sprintf("local send 1 %+v", commandDetector))

	if commandDetector.Command == "answer" {
		logger.String(fmt.Sprintf("local send 2 %+v", commandDetector.CommandId))

		localAnswers[int(commandDetector.CommandId)] = []byte(message)
	}

}

func (l *LocalConnection) SetSession (session *model_auth_session.Session) {

}

func (l *LocalConnection) GetSession () *model_auth_session.Session {
	return nil
}

func (l *LocalConnection) SetServerAuthState () {
}

func (l *LocalConnection) GetServerAuthState () bool {
	return true
}


func (l *LocalConnection) GetRemoteAddr() string {
	// return localServer.RemoteAddr
	return ""
}

func (l *LocalConnection) GetUserAgent() string {
	// return localServer.UserAgent
	return ""
}

func NewServer(serverUUID gocql.UUID) *Server {
	s := &Server{ isLocal: false }

	e := model_server.Get(serverUUID)
	if !e.Exists {
		logger.String(fmt.Sprintf("server %s not found in storage", serverUUID.String()))
		return nil
	}

	path := fmt.Sprintf("http://%s:%d/ws", e.IP, e.Port)
	logger.String(fmt.Sprintf("connect to: %s", path))
	u, err := url.Parse(path)
	if err != nil {

		logger.String(fmt.Sprintf("%v", err))

	    // return err
	    return nil
	}

	rawConn, err := net.Dial("tcp", u.Host)
	if err != nil {
		logger.String(fmt.Sprintf("%v", err))
	    // return err
	    return nil
	}

	wsHeaders := http.Header{
	    "Origin":                   {"http://local.host:80"},
	    // your milage may differ
	    "Sec-WebSocket-Extensions": {"permessage-deflate; client_max_window_bits, x-webkit-deflate-frame"},
	    "User-Agent": {"spacecrfat-agent"},
	}

	wsConn, _, err := websocket.NewClient(rawConn, u, wsHeaders, 4096, 4096)
	if err != nil {
		logger.String(fmt.Sprintf("%v", err))
	    // return fmt.Errorf("websocket.NewClient Error: %s\nResp:%+v", err, resp)
	    return nil
	}

	s.ws = wsConn

	s.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err = s.ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"command":"auth","server_uuid":"%s"}`, localServer.UUID.String())))
	if err != nil {
		logger.String(fmt.Sprintf("%v", err))
		return nil
	}

	_, message, err := s.ws.ReadMessage()
	// s.ws.ReadMessage()

	
	if err != nil {
		if err != io.EOF {
			logger.Error(errors.New(err))
		}
	}
	
	smessage := string(message)
	logger.String(fmt.Sprintf("star auth answer: %v", smessage))

	go s.Reading()
	go s.Pinger()

	return s
}

func GetServer(serverUUID gocql.UUID) *Server {

	// todo: add mutex

	server, ok := servers[serverUUID.String()]
	if !ok {
		server = NewServer(serverUUID)
		if server == nil {
			return nil
		}

		servers[serverUUID.String()] = server
	}

	return server
}

func (s *Server) Pinger() {

	pingTicker := time.NewTicker(PingPeriod)
	defer func() {
		pingTicker.Stop()		
	}()

	for {

		select {
		case <-pingTicker.C:
			s.ws.SetWriteDeadline(time.Now().Add(WriteWait))

			if err := s.ws.WriteMessage(websocket.TextMessage, []byte(`{"command":"ping"}`)); err != nil {
				logger.Error(errors.New(err))
				break
			}
		}
	}
}

func (s *Server) Send(message map[string]interface{}) ([]byte, error) {

	b, err := json.Marshal(message)
	if err != nil {
		logger.Error(errors.New(err))
		return nil, err
	}

	logger.String(fmt.Sprintf("star send: %+v", string(b)))

	if s.isLocal {

		command := message["command"].(string)
		logger.String(fmt.Sprintf("isLocal: %s", command))

		generator, ok := commands[command]
		if ok {

			command := generator(localConnection, commandContext)
			command.Execute(b)

			a := localAnswers[message["command_id"].(int)]
			delete(localAnswers, message["command_id"].(int))
			return a, nil
		}
		
		return nil, nil
	}

	s.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))

	rc := make(chan []byte)
	answers[message["command_id"].(int)] = rc

	err = s.ws.WriteMessage(websocket.TextMessage, b)
	if err != nil {
		return nil, err
	}

	ri := <-rc
	return ri, nil
}

func (s *Server) Reading() {

	s.ws.SetReadLimit(MaxMessageSize)
	s.ws.SetReadDeadline(time.Now().Add(PongWait))	
	// s.ws.SetPongHandler(func(string) error { s.ws.SetReadDeadline(time.Now().Add(PongWait)); return nil })

	for {
		_, message, err := s.ws.ReadMessage()
		if err != nil {
			if err != io.EOF {
				logger.Error(errors.New(err))
			}
			break
		}
		// logger.String(fmt.Sprintf("star message type %d", mt))
		smessage := string(message)
		if smessage != `{"command":"pong"}` {
			logger.String(fmt.Sprintf("star message: %v", smessage))
		}

		commandDetector := CommandDetector{}
		err = json.Unmarshal(message, &commandDetector)
		if err != nil {
			logger.Error(errors.New(err))
			continue
		}

		if commandDetector.Command == "answer" {
			answer, ok := answers[int(commandDetector.CommandId)]
			if ok {
				answer <- message
				delete(answers, int(commandDetector.CommandId))
			}
		}

		s.ws.SetReadDeadline(time.Now().Add(PongWait))
	}
}

func Send (serverUUID gocql.UUID, message map[string]interface{}) ([]byte, error) {

 	mutexCommandId.Lock()
	commandId := commandIdIterator
	commandIdIterator = commandIdIterator + 1
    mutexCommandId.Unlock()

    message["command_id"] = commandId

    server := GetServer(serverUUID)

    if server != nil {
    	a, e := server.Send(message)
    	return a, e
    }

    return nil, nil
}

func SetLocalServer(server *model_server.Server) {
	localServer = server
	servers[server.UUID.String()] = &Server{ isLocal: true }
}

func SetCommands(commands_ map[string]command.Generator, commandContext_ *command.Context) {
	commands = commands_
	commandContext = commandContext_
}

