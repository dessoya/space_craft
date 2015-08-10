
package main

import (
	"expvar"
	"runtime"

	"strings"
	"net"
	"encoding/json"
	"io"	
	"io/ioutil"	

    "fmt"
	"net/http"
	"os"
	"log"
	// "flag"

	"sc/logger"
	"sc/errors"
	"sc/star"

	"sc/ws/command"
	"sc/ws/connection"
	"sc/ws/connection/factory"
	"github.com/gocql/gocql"
	module_config "sc/config"

	"sc/model"

	model_auth_session "sc/models/auth_session"
	model_user "sc/models/user"
	model_server "sc/models/server"

	cmd_auth "sc/ws/commands/auth"
	cmd_logout "sc/ws/commands/logout"

	cmd_session_lock_state "sc/ws/star_commands/session_lock_state"
)

func goroutines() interface{} {
    return runtime.NumGoroutine()
}

func localIP() (string, error) {

	addrs, tt := net.InterfaceAddrs()
	if tt != nil {
    	return "", tt
	}   
	for _, addr := range addrs {
		s := addr.String()
		if s[0] == '0' {
			continue
		}
		return s, nil
	}  
	return "", nil
} 


var config *module_config.Config

func main() {

	expvar.Publish("Goroutines", expvar.Func(goroutines))
	runtime.GOMAXPROCS(6)	

	var err *errors.Error

    config, err = readConfig()

	if err != nil {
		// logger.Error(fmt.Sprintf("readConfig - %v", err))		
		log.Fatal(err)
		// log.Fatal("ListenAndServe: ", err)
		os.Exit(0)
	}	

    // fmt.Printf("%+v\n", *config)

    logger.Init(config.Logger.Path)
    logger.String(fmt.Sprintf("started"))



	cluster := gocql.NewCluster("192.168.88.102")
    cluster.Keyspace = "sc_2"
    cluster.Consistency = 1

    session, berr := cluster.CreateSession()


    model_auth_session.Init(session)
    model_user.Init(session)
    model_server.Init(session)



    ip, t := localIP()
    if t != nil {
    	logger.Error(errors.New(t))
    	os.Exit(0)
    }

    server := model_server.New(ip, config.Http.Port)
    model.Init(server.UUID, session)

    server.StartUpdater()
    star.SetLocalServer(server)

	var connectionFactory = factory.New()

	// clients commands
	connectionFactory.InstallCommand("auth", cmd_auth.Generator)
	connectionFactory.InstallCommand("logout", cmd_logout.Generator)

	// star commands
	connectionFactory.InstallCommand("get_session_lock_state", cmd_session_lock_state.Generator)

	commandContext := &command.Context{ CQLSession: session, Config: config, ServerUUID: server.UUID }

	star.SetCommands(connectionFactory.GetCommands(), commandContext)



	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {

		logger.String("/ws")
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}

		ws, err := connection.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error(errors.New(err))
			return
		}

		ra := r.RemoteAddr[:strings.Index(r.RemoteAddr, ":")]
		c := connectionFactory.CreateConnection(ws, commandContext, ra, r.Header["User-Agent"][0])
		// logger.String(ra)
		// logger.String(r.Header["User-Agent"][0])
		logger.String(fmt.Sprintf("accept connection %v", c.Id))
		// go c.Writing()
		c.Reading()
		c.Close()
		logger.String(fmt.Sprintf("close connection %v", c.Id))
	})

	http.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		b, err := json.Marshal(connectionFactory.MakeDebugInfo())
		if err != nil {
			logger.Error(errors.New(err))
		}
		io.WriteString(w, string(b))
	})

	http.HandleFunc("/api/auth/success", func(w http.ResponseWriter, r *http.Request) {		
		
		logger.String("/api/auth/success")

		/*
		ra := r.RemoteAddr[:strings.Index(r.RemoteAddr, ":")]
		logger.String("remoteAdder " + ra)
		logger.String(fmt.Sprintf("%+v", r.Header))
		*/

		/*
		logger.String(fmt.Sprintf("session_uuid %s", r.URL.Query().Get("session_uuid")))
		logger.String(fmt.Sprintf("method %s", r.URL.Query().Get("method")))
		logger.String(fmt.Sprintf("username %s", r.URL.Query().Get("username")))
		logger.String(fmt.Sprintf("unique %s", r.URL.Query().Get("unique")))
		logger.String(fmt.Sprintf("token %s", r.URL.Query().Get("token")))
		*/

		client := &http.Client{}
		req, err := http.NewRequest("GET", "http://auth.spacecraft-online.org/api/check_token?token=" + r.URL.Query().Get("token"), nil)
		if err != nil {
			logger.Error(errors.New(err))
			return
		}

		resp, err := client.Do(req)		
		if err != nil {
			logger.Error(errors.New(err))
			return
		}

 		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Error(errors.New(err))
			return
		}

 		if string(body) != "{\"status\":\"ok\",\"result\":true}" {
			logger.Error(errors.New(err))
			return
 		}

		// logger.String(string(body))		

		session_uuid := r.URL.Query().Get("session_uuid")
		logger.String("session_uuid " + session_uuid)

		session := model_auth_session.LoadOrCreateSession(session_uuid, "", r.Header["User-Agent"][0])

		methodUUID := model_user.GetMethodUUID(r.URL.Query().Get("method"), r.URL.Query().Get("unique"))
		user, _ := model_user.GetByMethodUUID(r.URL.Query().Get("method"), methodUUID)

		if session.IsAuth {

			// check for another user and relogin
			if user.Exists && user.UUID.String() != session.UserUUID.String() {

				// m := r.URL.Query().Get("method") + "_uuid"

				// user.AddMethod(r.URL.Query().Get("method"), methodUUID)

				/*
				user.Update(map[string]interface{}{
					m: methodUUID,
				})
				*/

/*
				session.Update(map[string]interface{}{
					"user_uuid": user.UUID,
					"auth_method": r.URL.Query().Get("method"),
				})
				*/

				session.Update(model.Fields{
					"UserUUID":		user.UUID,
					"AuthMethod":	r.URL.Query().Get("method"),
				})

			} else {

			}

			// update auth method



		} else {

			// loging
			if !user.Exists {
				user.Create()
				// m := r.URL.Query().Get("method") + "_uuid"
				user.Update(map[string]interface{}{
					"username": r.URL.Query().Get("username"),
				})
				// user.AddMethod(r.URL.Query().Get("method"), methodUUID)
			}

			session.Update(map[string]interface{}{
				"IsAuth": true,
				"UserUUID": user.UUID,
				"AuthMethod": r.URL.Query().Get("method"),
			})
		}

		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	})

	listen := fmt.Sprintf(":%v", config.Http.Port)
	logger.String(fmt.Sprintf("listen http %v", listen))
	berr = http.ListenAndServe(listen, nil)
	if berr != nil {
		// logger.Error(fmt.Sprintf("ListenAndServe - %v", err))
		logger.Error(errors.New(berr))
		// log.Fatal("ListenAndServe: ", err)
		os.Exit(0)
	}
}