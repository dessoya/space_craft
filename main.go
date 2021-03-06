
package main

import (
	"syscall"
	"os"
	"github.com/sevlyar/go-daemon"
	"time"

//	"log"

	"expvar"
	"runtime"

	"strings"
	"net"
	"encoding/json"
	"io"	
	"io/ioutil"	

    "fmt"
	"net/http"
	"log"
	"flag"

	"sc/logger"
	"sc/errors"
	"sc/star"

	"sc/buildings"

	"sc/ws/command"
	"sc/ws/connection"
	"sc/ws/connection/factory"
	"github.com/gocql/gocql"
	module_config "sc/config"

	cmd_auth "sc/ws/commands/auth"
	cmd_logout "sc/ws/commands/logout"
	cmd_set_section "sc/ws/commands/set_section"
	cmd_start "sc/ws/commands/start"
	cmd_get_planet "sc/ws/commands/get_planet"
	cmd_get_planet_buildings_for_construct "sc/ws/commands/get_planet_buildings_for_construct"
	cmd_build  "sc/ws/commands/build"

	cmd_session_lock_state "sc/ws/star_commands/session_lock_state"
	cmd_user_lock_state "sc/ws/star_commands/get_user_lock_state"
	cmd_user_logout "sc/ws/star_commands/star_user_logout"

	"sc/model"
	"sc/model2"

	model_server "sc/models/server"

	model2_player			"sc/models2/player"
	model2_auth_session		"sc/models2/auth_session"
	model2_user				"sc/models2/user"
	model2_live_planet		"sc/models2/live_planet"
	model2_building			"sc/models2/building"

)

var (
	signal = flag.String("s", "", `send signal to the daemon
		quit — graceful shutdown
		stop — fast shutdown
		reload — reloading the configuration file`)
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

	log.SetFlags(log.Llongfile)

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

    logger.Init(config.Logger.Path)

    if config.Daemonize {
    
		flag.Parse()
		daemon.AddCommand(daemon.StringFlag(signal, "quit"), syscall.SIGQUIT, termHandler)
		daemon.AddCommand(daemon.StringFlag(signal, "stop"), syscall.SIGTERM, termHandler)
		daemon.AddCommand(daemon.StringFlag(signal, "reload"), syscall.SIGHUP, reloadHandler)

		cntxt := &daemon.Context{
			PidFileName: config.PidFilepath,
			PidFilePerm: 0644,
			LogFileName: config.Logger.Path + "/stdout.log",
			LogFilePerm: 0640,
			WorkDir:     "./",
			Umask:       027,
			Args:        []string{"[spacecraft-online]"},
		}

		if len(daemon.ActiveFlags()) > 0 {
			d, err := cntxt.Search()
			if err != nil {
				log.Fatalln("Unable send signal to the daemon:", err)
			}
			daemon.SendCommands(d)
			return
		}


		d, err := cntxt.Reborn()
		if err != nil {
			log.Fatalln(err)
		}
		if d != nil {
			return
		}
		defer cntxt.Release()

    }

    logger.String(fmt.Sprintf("started"))

	cluster := gocql.NewCluster(config.Cassandra.IP)
    cluster.Keyspace = "sc_2"
    cluster.Consistency = 1

    session, berr := cluster.CreateSession()



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

    bdispatcher := buildings.NewDispatcher(config.Buildings.PoolSize)


    model2.InstallModels(session,
    	server.UUID,
    	model2_auth_session.InstallInfo,
    	model2_user.InstallInfo,
    	model2_live_planet.InstallInfo,
    	model2_building.InstallInfo,
    	model2_player.InstallInfo)


	var connectionFactory = factory.New()

	// clients commands
	connectionFactory.InstallCommand("auth", cmd_auth.Generator)
	connectionFactory.InstallCommand("logout", cmd_logout.Generator)
	connectionFactory.InstallCommand("set_section", cmd_set_section.Generator)
	connectionFactory.InstallCommand("start", cmd_start.Generator)	
	connectionFactory.InstallCommand("get_planet", cmd_get_planet.Generator)	
	connectionFactory.InstallCommand("get_planet_buildings_for_construct", cmd_get_planet_buildings_for_construct.Generator)
	connectionFactory.InstallCommand("build", cmd_build.Generator)
	


	// star commands
	connectionFactory.InstallCommand("get_session_lock_state", cmd_session_lock_state.Generator)
	connectionFactory.InstallCommand("get_user_lock_state", cmd_user_lock_state.Generator)
	connectionFactory.InstallCommand("star_user_logout", cmd_user_logout.Generator)

	commandContext := &command.Context{ Factory: connectionFactory, CQLSession: session, Config: config, ServerUUID: server.UUID, BDispatcher: bdispatcher }

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
 			logger.String(string(body))
			http.Redirect(w, r, "/", http.StatusMovedPermanently)			
			// logger.Error(errors.New(err))
			return
 		}

		// logger.String(string(body))		

		sessionUUID, err := gocql.ParseUUID(r.URL.Query().Get("session_uuid"))
		if err != nil {
			http.Redirect(w, r, "/", http.StatusMovedPermanently)
			return
		}


		session, err := model2_auth_session.Get(sessionUUID)
		if err != nil || session == nil {
			http.Redirect(w, r, "/", http.StatusMovedPermanently)
			return
		}

		// session := model_auth_session.LoadOrCreateSession(session_uuid, "", r.Header["User-Agent"][0])

		method := r.URL.Query().Get("method")
		unique := r.URL.Query().Get("unique")
		user, _ := model2_user.GetByMethod(method, unique)

		if session.IsAuth {

			// check for another user and relogin
			if user.Exists && user.UUID.String() != session.UserUUID.String() {

				user.AddMethod(method, unique)

			} else {

			}

		} else {

			// loging
			if !user.Exists {
				user, _ = model2_user.Create()
				user.Update(model2.Fields{
					"Name": r.URL.Query().Get("username"),
				})
				user.AddMethod(method, unique)
			}

		}

		session.Update(model2.Fields{
			"IsAuth": true,
			"UserUUID": user.UUID,
			"AuthMethod": method,
		})

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

	go worker()

	derr := daemon.ServeSignals()
	if derr != nil {
		log.Println("Error:", derr)
	}
	// log.Println("daemon terminated")
    logger.String(fmt.Sprintf("daemon terminated"))

}

var (
	stop = make(chan struct{})
	done = make(chan struct{})
)

func worker() {
	for {
		time.Sleep(time.Second)
		if _, ok := <-stop; ok {
			break
		}
	}
	done <- struct{}{}
}

func termHandler(sig os.Signal) error {
    logger.String(fmt.Sprintf("terminating..."))
	time.Sleep(time.Second)
	stop <- struct{}{}
	if sig == syscall.SIGQUIT {
		<-done
	}
	return daemon.ErrStop
}

func reloadHandler(sig os.Signal) error {
    logger.String(fmt.Sprintf("configuration reloaded"))
	return nil
}
