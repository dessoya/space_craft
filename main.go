
package main

import (
	"expvar"
	"runtime"

	"encoding/json"
	"io"	

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

	cmd_auth "sc/ws/commands/auth"
)

func goroutines() interface{} {
    return runtime.NumGoroutine()
}


var config *Config

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



	var connectionFactory = factory.New()

	connectionFactory.InstallCommand("auth", cmd_auth.Generator)


	commandContext := &command.Context{ CQLSession: session }

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