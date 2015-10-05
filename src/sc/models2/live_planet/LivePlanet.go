
package live_planet

import(
	"github.com/gocql/gocql"
	"sync"
	"fmt"
	"strings"
	"strconv"
	model "sc/model2"
	"sc/logger"
	"sc/errors"
	model_building "sc/models2/building"
	// "sc/ws/command"
)

var TableName		= "live_planets"
var UUIDFieldName	= "planet_uuid"

type Connection interface {
	Send(string)	
}

type Fields struct {
	UUID			gocql.UUID		`cql:"planet_uuid"`
	Exists			bool
	IsLock			bool			`cql:"lock"`
	LockServerUUID	gocql.UUID		`cql:"lock_server_uuid"`
	PlayerUUID		gocql.UUID		`cql:"owner_player_uuid"`
	Buildings		[]gocql.UUID	`cql:"buildings_list"`
	Population		float64			`cql:"population"`
	PopulationSInc	float64			`cql:"population_sinc"`
	PopulationUsage	float64			`cql:"population_usage"`
	PopulationAvail	float64			`cql:"population_avail"`
	Energy			float64			`cql:"energy"`
	EnergyAvail		float64			`cql:"energy_avail"`
	Crystals		float64			`cql:"crystals"`
	CrystalsSInc	float64			`cql:"crystals_sinc"`
	Minerals		float64			`cql:"minerals"`
	MineralsSInc	float64			`cql:"minerals_sinc"`
	TreatTime		int64			`cql:"treat_time"`
	QueueBuildType	[]string		`cql:"queue_build_type"`
	QueueBuildX		[]int			`cql:"queue_build_x"`
	QueueBuildY		[]int			`cql:"queue_build_y"`
	BuildInProgress	[]gocql.UUID	`cql:"build_in_progress"`
	TurnOnBuildings	[]gocql.UUID	`cql:"turn_on_buildings"`
	Connection		Connection
	DMutex			sync.Mutex
}


func (lp *Fields) MakeClientInfo() (info model.Fields) {

    info = model.Fields{}
    var buildings []interface{}

	for _, uuid := range lp.Buildings {
		b, _ := model_building.Get(uuid)
		if b != nil {
			buildings = append(buildings, b.MakeClientInfo())
		}
	}
	
	info["population"] = lp.Population
	info["population_avail"] = lp.PopulationAvail
	info["minerals"] = lp.Minerals
	info["crystals"] = lp.Crystals
	info["energy"] = lp.Energy
	info["energy_avail"] = lp.EnergyAvail

	info["buildings"] = buildings

	return
}

func (lp *Fields) GetConnection() (Connection) {
	return lp.Connection
}

func (lp *Fields) NCUpdatePlanetResources() (string) {
	return fmt.Sprintf(`{"command":"nc_update_planet_resource","planet_uuid":"%s","resources":{"minerals":%d,"crystals":%d,"population_avail":%d,"energy":%d,"energy_avail":%d}}`,
		lp.UUID.String(),
		int(lp.Minerals),
		int(lp.Crystals),
		int(lp.PopulationAvail),
		int(lp.Energy),
		int(lp.EnergyAvail))
}
var Field2CQL = map[string]string{
	"UUID": "planet_uuid",
	"IsLock": "lock",
	"LockServerUUID": "lock_server_uuid",
	"PlayerUUID": "owner_player_uuid",
	"Buildings": "buildings_list",
	"Population": "population",
	"PopulationSInc": "population_sinc",
	"PopulationUsage": "population_usage",
	"PopulationAvail": "population_avail",
	"Energy": "energy",
	"EnergyAvail": "energy_avail",
	"Crystals": "crystals",
	"CrystalsSInc": "crystals_sinc",
	"Minerals": "minerals",
	"MineralsSInc": "minerals_sinc",
	"TreatTime": "treat_time",
	"QueueBuildType": "queue_build_type",
	"QueueBuildX": "queue_build_x",
	"QueueBuildY": "queue_build_y",
	"BuildInProgress": "build_in_progress",
	"TurnOnBuildings": "turn_on_buildings",
}

var InstallInfo = model.InstallInfo{ Init: Init }
var LockServerUUID gocql.UUID
var CQLSession *gocql.Session

func Init(session *gocql.Session, serverUUID gocql.UUID) {
	LockServerUUID = serverUUID
	CQLSession = session
}

func Load(UUID gocql.UUID) (*Fields, error) {
	m := &Fields{ UUID: UUID }
	err := m.Load()
	return m, err
}

func (m *Fields) Load() (error) {

	m.Exists = false
	var row = map[string]interface{}{}

	if err := CQLSession.Query(`SELECT * FROM live_planets where planet_uuid = ?`, m.UUID).MapScan(row); err != nil {
		return err
	}
	m.Exists = true

	m.UUID = row["planet_uuid"].(gocql.UUID)
	m.IsLock = row["lock"].(bool)
	m.LockServerUUID = row["lock_server_uuid"].(gocql.UUID)
	m.PlayerUUID = row["owner_player_uuid"].(gocql.UUID)
	m.Buildings = row["buildings_list"].([]gocql.UUID)
	v6 := row["population"]
	if v6 == nil { m.Population = 0
	} else { m.Population = v6.(float64) }
	v7 := row["population_sinc"]
	if v7 == nil { m.PopulationSInc = 0
	} else { m.PopulationSInc = v7.(float64) }
	v8 := row["population_usage"]
	if v8 == nil { m.PopulationUsage = 0
	} else { m.PopulationUsage = v8.(float64) }
	v9 := row["population_avail"]
	if v9 == nil { m.PopulationAvail = 0
	} else { m.PopulationAvail = v9.(float64) }
	v10 := row["energy"]
	if v10 == nil { m.Energy = 0
	} else { m.Energy = v10.(float64) }
	v11 := row["energy_avail"]
	if v11 == nil { m.EnergyAvail = 0
	} else { m.EnergyAvail = v11.(float64) }
	v12 := row["crystals"]
	if v12 == nil { m.Crystals = 0
	} else { m.Crystals = v12.(float64) }
	v13 := row["crystals_sinc"]
	if v13 == nil { m.CrystalsSInc = 0
	} else { m.CrystalsSInc = v13.(float64) }
	v14 := row["minerals"]
	if v14 == nil { m.Minerals = 0
	} else { m.Minerals = v14.(float64) }
	v15 := row["minerals_sinc"]
	if v15 == nil { m.MineralsSInc = 0
	} else { m.MineralsSInc = v15.(float64) }
	m.TreatTime = row["treat_time"].(int64)
	m.QueueBuildType = row["queue_build_type"].([]string)
	m.QueueBuildX = row["queue_build_x"].([]int)
	m.QueueBuildY = row["queue_build_y"].([]int)
	m.BuildInProgress = row["build_in_progress"].([]gocql.UUID)
	m.TurnOnBuildings = row["turn_on_buildings"].([]gocql.UUID)

	return nil
}

func Create() (*Fields, error) {
    var err error
	m := &Fields{ Exists: false }
	for {
		m.UUID, err = gocql.RandomUUID()
		if err != nil {
			return nil, err
		}
		var row = map[string]interface{}{}
		var apply bool
		if apply, err = CQLSession.Query(`insert into live_planets (planet_uuid,create_time) values (?,now()) if not exists`, m.UUID).MapScanCAS(row); err != nil {
			logger.Error(errors.New(err))
			return nil, err
		}
		if apply {
			break
		}
	}
	return m, nil
}

var mutex sync.RWMutex
var Models = map[string]*Fields{}

func Access(uuid string) (*Fields) {

    mutex.RLock()
	m, ok := Models[uuid]
    mutex.RUnlock()
    if ok {
    	return m
    }
    return nil
}

func Get(UUID gocql.UUID) (*Fields, error) {

    var err error
    uuid := UUID.String()
    mutex.RLock()
	m, ok := Models[uuid]
    mutex.RUnlock()
	if !ok {
		m, err = Load(UUID)
		if err != nil {
			return nil, err
		}
		if m == nil {
			return nil, nil
		}
	    mutex.Lock()
		m2, ok := Models[uuid]
	    if ok {
		    mutex.Unlock()
	    	return m2, nil
	    }
		Models[uuid] = m
	    mutex.Unlock()
	    m.Update(model.Fields{
	    	"IsLock": true,
	    	"LockServerUUID": LockServerUUID,
	    })
	}

	return m, nil

}

func (m *Fields) Lock() (error) {

    uuid := m.UUID.String()
    mutex.Lock()
	m, ok := Models[uuid]
	if ok {
	    mutex.Unlock()
		return m.Load()
	} else {
		Models[uuid] = m
	    mutex.Unlock()
	    m.Update(model.Fields{
	    	"IsLock": true,
	    	"LockServerUUID": LockServerUUID,
	    })
	}

	return nil

}

func (m *Fields) Unlock() error {

    uuid := m.UUID.String()
    mutex.Lock()
	_, ok := Models[uuid]

    if ok {
    	delete(Models, uuid)
    }
    mutex.Unlock()

    err := m.Update(model.Fields{
    	"IsLock": nil,
    	"LockServerUUID": nil,
    })

    return err
}

func (m *Fields) Update(fields model.Fields) error {

	pairs := []string{}

	for key, value := range fields {
	    switch key {
		case "UUID":
			switch value.(type) {
			case nil:
			m.UUID = gocql.UUID{}
			default:
			m.UUID = value.(gocql.UUID)
			}
		case "IsLock":
			switch value.(type) {
			case nil:
			m.IsLock = false
			default:
			m.IsLock = value.(bool)
			}
		case "LockServerUUID":
			switch value.(type) {
			case nil:
			m.LockServerUUID = gocql.UUID{}
			default:
			m.LockServerUUID = value.(gocql.UUID)
			}
		case "PlayerUUID":
			switch value.(type) {
			case nil:
			m.PlayerUUID = gocql.UUID{}
			default:
			m.PlayerUUID = value.(gocql.UUID)
			}
		case "Buildings":
			m.Buildings = value.([]gocql.UUID)
		case "Population":
			switch t := value.(type) {
			case int:
			m.Population = float64(t)
			default:
			m.Population = value.(float64)
			}
		case "PopulationSInc":
			switch t := value.(type) {
			case int:
			m.PopulationSInc = float64(t)
			default:
			m.PopulationSInc = value.(float64)
			}
		case "PopulationUsage":
			switch t := value.(type) {
			case int:
			m.PopulationUsage = float64(t)
			default:
			m.PopulationUsage = value.(float64)
			}
		case "PopulationAvail":
			switch t := value.(type) {
			case int:
			m.PopulationAvail = float64(t)
			default:
			m.PopulationAvail = value.(float64)
			}
		case "Energy":
			switch t := value.(type) {
			case int:
			m.Energy = float64(t)
			default:
			m.Energy = value.(float64)
			}
		case "EnergyAvail":
			switch t := value.(type) {
			case int:
			m.EnergyAvail = float64(t)
			default:
			m.EnergyAvail = value.(float64)
			}
		case "Crystals":
			switch t := value.(type) {
			case int:
			m.Crystals = float64(t)
			default:
			m.Crystals = value.(float64)
			}
		case "CrystalsSInc":
			switch t := value.(type) {
			case int:
			m.CrystalsSInc = float64(t)
			default:
			m.CrystalsSInc = value.(float64)
			}
		case "Minerals":
			switch t := value.(type) {
			case int:
			m.Minerals = float64(t)
			default:
			m.Minerals = value.(float64)
			}
		case "MineralsSInc":
			switch t := value.(type) {
			case int:
			m.MineralsSInc = float64(t)
			default:
			m.MineralsSInc = value.(float64)
			}
		case "TreatTime":
			switch t := value.(type) {
			case int:
			m.TreatTime = int64(t)
			default:
			m.TreatTime = value.(int64)
			}
		case "QueueBuildType":
			m.QueueBuildType = value.([]string)
		case "QueueBuildX":
			m.QueueBuildX = value.([]int)
		case "QueueBuildY":
			m.QueueBuildY = value.([]int)
		case "BuildInProgress":
			m.BuildInProgress = value.([]gocql.UUID)
		case "TurnOnBuildings":
			m.TurnOnBuildings = value.([]gocql.UUID)
		}
		var pair = Field2CQL[key] + "="
		switch t := value.(type) {
		case nil:
			pair += "null"
		case bool:
			pair += fmt.Sprintf("%v", t)
		case int:
			pair += fmt.Sprintf("%v", t)
		case string:
			pair += "'" + t + "'"
		case float64:
			pair += fmt.Sprintf("%v", t)
		case int64:
			pair += fmt.Sprintf("%v", t)
		case *gocql.UUID:
			pair += t.String()
		case []*gocql.UUID:
			a := []string{}
			for _, uuid := range t {
				a = append(a, uuid.String())
			}
			pair += "[" + strings.Join(a, ",") + "]"
		case []gocql.UUID:
			a := []string{}
			for _, uuid := range t {
				a = append(a, uuid.String())
			}
			pair += "[" + strings.Join(a, ",") + "]"
		case []string:
			a := []string{}
			for _, s := range t {
				a = append(a, `'` + s + `'`)
			}
			pair += "[" + strings.Join(a, ",") + "]"
		case []int:
			a := []string{}
			for _, i := range t {
				a = append(a, strconv.Itoa(i))
			}
			pair += "[" + strings.Join(a, ",") + "]"
		case gocql.UUID:
			pair += t.String()
		default:
			logger.Error(errors.New(fmt.Sprintf("unknown type: %+v",t)))
		}
		pairs = append(pairs, pair)
	}
	q := "update live_planets set " + strings.Join(pairs, ",") + " where planet_uuid = " + m.UUID.String()
	logger.String(q)

	if err := CQLSession.Query(q).Exec(); err != nil {
		logger.Error(errors.New(err))
		return err
	}

	return nil

}

func GetLockedModels() ([]string) {
	keys := []string{}
    mutex.RLock()
    for key, _ := range Models {
    	keys = append(keys, key)
    }
    mutex.RUnlock()
	return keys
}
