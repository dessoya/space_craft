
package buildings

import (
	"github.com/gocql/gocql"
	"fmt"
	"sc/logger"
	"sc/model2"
	"time"
	model_building "sc/models2/building"
	model_live_planet "sc/models2/live_planet"
)

type Dispatcher struct {

	PoolSize	uint16
	Workers		[]*Worker
	RR			int

}

func NewDispatcher(poolSize uint16) (*Dispatcher) {

	dispatcher := &Dispatcher{ PoolSize: poolSize, Workers: make([]*Worker, poolSize), RR: 0 }

	for index, _ := range dispatcher.Workers {
		worker := NewWorker(dispatcher)
		dispatcher.Workers[index] = worker
	}

	go dispatcher.PlanetScanner()

	return dispatcher
}

func (d *Dispatcher) PlanetScanner() (*Worker) {

	for {

		// 1. check for workers channel size > 1/3 of size
		var total = 0
		var used = 0

		for _, w := range d.Workers {
			total += cap(w.C)
			used += len(w.C)
		}

		// logger.String(fmt.Sprintf("total %d used %d", total, used))

		if total / 3 > used {

			// 2. read planet keys
			keys := model_live_planet.GetLockedModels()

			// 3. read each and send to treator
			for _, key := range keys {
				planet := model_live_planet.Access(key)
				if planet == nil { continue }

				ptt := planet.TreatTime / 1000000000
				ctt := time.Now().UnixNano()

				if ctt > ptt {
					worker := d.getFreeWorker()
					m := WorkerMessage{ Type: MT_Treat, PlanetUUID: &planet.UUID }
					worker.C <- m
				}

				// check total + used and sleep if need

			}

		}

		// 4. sleep 1 sec
		time.Sleep(time.Second)

	}

}


func (d *Dispatcher) getFreeWorker() (*Worker) {

	d.RR += 1
	i := d.RR
	if i >= cap(d.Workers) {
		i = 0
		d.RR = 0
	}
	return d.Workers[i]
}

func (d *Dispatcher) Build(PlanetUUID *gocql.UUID, bt string, x int, y int) {

	worker := d.getFreeWorker()
	m := WorkerMessage{ Type: MT_Build, PlanetUUID: PlanetUUID, Params: make(map[string]interface{}) }
	m.Params["type"] = bt
	m.Params["x"] = x
	m.Params["y"] = y
	worker.C <- m

}

func (d *Dispatcher) TurnOn(PlanetUUID *gocql.UUID, BuildingUUID *gocql.UUID) {

	logger.String("Dispatcher.TurnOn")
	worker := d.getFreeWorker()
	m := WorkerMessage{ Type: MT_TurnOn, PlanetUUID: PlanetUUID, Params: make(map[string]interface{}) }
	m.Params["*building_uuid"] = BuildingUUID
	worker.C <- m

}

func lockPlanet(planet *model_live_planet.Fields) {
    planet.DMutex.Lock()
}

func unlockPlanet(planet *model_live_planet.Fields) {
    planet.DMutex.Unlock()	
}

const (
 	MT_Quit int = iota
 	MT_Build
 	MT_TurnOn
 	MT_Treat
)

type WorkerMessage struct {
	Type int
	Params map[string]interface{}
	PlanetUUID *gocql.UUID
}

type Worker struct {

	D		*Dispatcher
	C		chan WorkerMessage

}

func NewWorker(D *Dispatcher) (*Worker) {
	worker := &Worker{ D: D, C: make(chan WorkerMessage, 128) }
	go worker.Loop()
	return worker
}

func getBuildingLevelInfo(btype string, level int) (map[string]interface{}) {

	var binfo map[string]interface{}
	switch btype {
	case "energy_station":
		binfo = EnergyStation
	}

	i1 := binfo["levelParams"].([]interface{})
	i2 := i1[level].(map[string]interface{})

	return i2
}

func getBuildingUsage(levelInfo map[string]interface{}) (float64) {
	return float64(levelInfo["popUsage"].(int))
}

/*

	1442085845				- s  / 1 000 000 000
	1442085845446			- ms / 1 000 000
	1442085845446983700		- ns
*/

func (w *Worker) Loop() {

	for {

		m := <- w.C

		// todo: lock planet
		planet, _ := model_live_planet.Get(*m.PlanetUUID)
		lockPlanet(planet)

		// logger.String(fmt.Sprintf("%+v", m))

		switch m.Type {
		case MT_Build:

			// if planet == nil {
			//	return
			// }

			i1 := EnergyStation["levelParams"].([]interface{})
			i2 := i1[0].(map[string]interface{})
			i3 := i2["level_up"]
			i4 := i3.(map[string]interface{})

			costMinerals := i4["minerals"].(int)
			costCrystals := i4["crystals"].(int)
			planet.Update(model2.Fields{
				"Minerals": planet.Minerals - float64(costMinerals),
				"Crystals": planet.Crystals - float64(costCrystals),
			})

			building, _ := model_building.Create()

			building.Update(model2.Fields{
				"Type":			m.Params["type"],
				"Level":		1,
				"TurnOn":		false,
				"TurnOnTime":	0,
				"X":			m.Params["x"],
				"Y":			m.Params["y"],
			})

			planet.Update(model2.Fields{
				"Buildings": append(planet.Buildings, building.UUID),
			})

			conn := planet.GetConnection()
			if conn != nil {
				conn.Send(fmt.Sprintf(`{"command":"nc_build","planet_uuid":"%s","building":{"type":"%s","x":%d,"y":%d}}`, m.PlanetUUID.String(), m.Params["type"], m.Params["x"], m.Params["y"]))
				conn.Send(fmt.Sprintf(`{"command":"nc_update_planet_resource","planet_uuid":"%s","resources":{"minerals":%d,"crystals":%d}}`, m.PlanetUUID.String(), int(planet.Minerals), int(planet.Crystals)))
			}

			w.D.TurnOn(m.PlanetUUID, &building.UUID)

		case MT_TurnOn:

			logger.String(fmt.Sprintf("TurnOnTime: %v", time.Now().UnixNano()))


			buildingUUID := m.Params["*building_uuid"].(*gocql.UUID)
			building, _ := model_building.Get(*buildingUUID)
			building.Update(model2.Fields{
				"TurnOnTime": time.Now().UnixNano(),
			})

			/*
			buildingUUID := m.Params["*building_uuid"].(*gocql.UUID)
			building, _ := model_building.Get(*buildingUUID)

			// 1. check availability
			levelInfo := getBuildingLevelInfo(building.Type, building.Level)
			popUsage := getBuildingUsage(levelInfo)

			avail := false
			if popUsage <= planet.PopulationAvail {
				avail = true
			}

			if avail {
				// 2. reduce resource

				// 3. turn on



				// 4. send connection notify

			} else {
				// 5. send connection notify fail turn on
			}
			*/

		case MT_Treat:

			ptt := planet.TreatTime / 1000000000
			ctt := time.Now().UnixNano()

			if ptt < ctt {

				att := planet.TreatTime + 1000000000

				// 1. check turn on buildings
				for _, buuid := range planet.Buildings {
					building, _ := model_building.Get(buuid)
					if building == nil {
						// todo: err
						continue
					}

					if building.TurnOn == false && building.TurnOnTime > 0 && building.TurnOnTime < att {

						logger.String("turn on building")

						conn := planet.GetConnection()

						// 1. check availability
						levelInfo := getBuildingLevelInfo(building.Type, building.Level)
						popUsage := getBuildingUsage(levelInfo)

						avail := false
						if popUsage <= planet.PopulationAvail {
							avail = true
						}

						if avail {

							// 2. reduce resource							
							// 3. turn on

							building.Update(model2.Fields{
								"PopulationAvail": planet.PopulationAvail - popUsage,
								"TurnOn": true,
								"TurnOnTime": 0,
							})

							// 4. send connection notify

							if conn != nil {
								conn.Send(fmt.Sprintf(`{"command":"nc_building_turnon","planet_uuid":"%s","building_uuid":"%s"}`, m.PlanetUUID.String(), building.UUID.String()))
							}

						} else {

							// 5. send connection notify fail turn on

							building.Update(model2.Fields{
								"TurnOnTime": 0,
							})

							if conn != nil {
								conn.Send(fmt.Sprintf(`{"command":"nc_building_turnon_problem","planet_uuid":"%s","building_uuid":"%s"}`, m.PlanetUUID.String(), building.UUID.String()))
							}

						}
					}

				}

				planet.Update(model2.Fields{
					"TreatTime": planet.TreatTime + 1000000000,
				})

			}

			// logger.String(fmt.Sprintf("treat planet %s", m.PlanetUUID.String()))

		}



		// todo: unlock planet
		unlockPlanet(planet)


	}
}
