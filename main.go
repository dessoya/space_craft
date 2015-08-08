
package main

import (
	"expvar"
	"runtime"

	"encoding/json"
	"io"	
	"io/ioutil"	

    "fmt"
	"net/http"
	"os"
	"log"

	"sc/logger"
	"sc/errors"

	"sc/ws/command"
	"sc/ws/connection"
	"sc/ws/connection/factory"
	"github.com/gocql/gocql"
	module_config "sc/config"

	model_auth_session "sc/models/auth_session"
	model_user "sc/models/user"

	cmd_auth "sc/ws/commands/auth"
	cmd_logout "sc/ws/commands/logout"
)

func goroutines() interface{} {
    return runtime.NumGoroutine()
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



	var connectionFactory = factory.New()

	connectionFactory.InstallCommand("auth", cmd_auth.Generator)
	connectionFactory.InstallCommand("logout", cmd_logout.Generator)


	commandContext := &command.Context{ CQLSession: session, Config: config }

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {

		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}

		ws, err := connection.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error(errors.New(err))
			return
		}

		c := connectionFactory.CreateConnection(ws, commandContext)
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
		session := model_auth_session.LoadOrCreateSession(session_uuid)

		methodUUID := model_user.GetMethodUUID(r.URL.Query().Get("method"), r.URL.Query().Get("unique"))
		user, _ := model_user.GetByMethodUUID(r.URL.Query().Get("method"), methodUUID)

		if session.IsAuth {

			// check for another user and relogin
			if user.Exists && user.UUID.String() != session.UserUUID.String() {

				m := r.URL.Query().Get("method") + "_uuid"
				user.Update(map[string]interface{}{
					m: methodUUID,
				})

				session.Update(map[string]interface{}{
					"user_uuid": user.UUID,
					"auth_method": r.URL.Query().Get("method"),
				})

			} else {

			}

			// update auth method



		} else {

			// loging
			if !user.Exists {
				user.Create()
				m := r.URL.Query().Get("method") + "_uuid"
				user.Update(map[string]interface{}{
					"username": r.URL.Query().Get("username"),
					m: methodUUID,
				})
			}

			session.Update(map[string]interface{}{
				"is_auth": true,
				"user_uuid": user.UUID,
				"auth_method": r.URL.Query().Get("method"),
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