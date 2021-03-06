
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

func GetBuildingLevelInfo(btype string, level int) (map[string]interface{}) {

	var binfo map[string]interface{}
	switch btype {
	case "energy_station":
		binfo = EnergyStation
	case "mineral_mine":
		binfo = MineralMine
	}

	i1 := binfo["levelParams"].([]interface{})
	i2 := i1[level].(map[string]interface{})

	return i2
}

func GetBuildingUsage(levelInfo map[string]interface{}) (float64, float64) {
	var eu float64
	energyUsage, ok := levelInfo["energyUsage"]
	if ok {
		eu = float64(energyUsage.(int))
	} else {
		eu = 0
	}
	return float64(levelInfo["popUsage"].(int)), eu
}

/*
	1442085845				- s  / 1 000 000 000
	1442085845446			- ms / 1 000 000
	1442085845446983700		- ns
*/

func (w *Worker) Loop() {

	for {

		mo := <- w.C
		m := &mo

		// todo: lock planet
		planet, _ := model_live_planet.Get(*m.PlanetUUID)
		lockPlanet(planet)

		// logger.String(fmt.Sprintf("%+v", m))

		switch m.Type {

		case MT_Build:
			w.MT_BuildProc(planet, m)

		case MT_TurnOn:

			logger.String(fmt.Sprintf("TurnOnTime: %v", time.Now().UnixNano()))


			buildingUUID := m.Params["*building_uuid"].(*gocql.UUID)
			building, _ := model_building.Get(*buildingUUID)
			building.Update(model2.Fields{
				"TurnOnTime": time.Now().UnixNano(),
			})

		case MT_Treat:

			ptt := planet.TreatTime / 1000000000
			ctt := time.Now().UnixNano() / 1000000000

			for ptt < ctt {

				conn := planet.GetConnection()

				att := planet.TreatTime + 1000000000

				// check for build queue
				for len(planet.QueueBuildType) > 0 {

					btype := planet.QueueBuildType[0]
					x := planet.QueueBuildX[0]
					y := planet.QueueBuildY[0]

					i1 := EnergyStation["levelParams"].([]interface{})
					i2 := i1[0].(map[string]interface{})
					i3 := i2["level_up"]
					i4 := i3.(map[string]interface{})

					costMinerals := i4["minerals"].(int)
					costCrystals := i4["crystals"].(int)
					duration := i4["time"].(int)
					population := i4["pop"].(int)

					building, _ := model_building.Create()

					building.Update(model2.Fields{
						"Type":					btype,
						"Level":				0,
						"TurnOn":				false,
						"TurnOnTime":			0,
						"X":					x,
						"Y":					y,
						"UpgradeInProgress":	true,
						"UpgradeDuration":		duration,
						"UpgradeElapsed":		0,
						"UpgradePopulation":	population,
					})

					planet.Update(model2.Fields{
						"Minerals":			planet.Minerals - float64(costMinerals),
						"Crystals":			planet.Crystals - float64(costCrystals),
						"PopulationAvail":	planet.PopulationAvail - float64(population),						
						"Buildings":		append(planet.Buildings, building.UUID),
						"QueueBuildType":	planet.QueueBuildType[1:],
						"QueueBuildX":		planet.QueueBuildX[1:],
						"QueueBuildY":		planet.QueueBuildY[1:],
						"BuildInProgress":	append(planet.BuildInProgress, building.UUID),
					})

					conn := planet.GetConnection()
					if conn != nil {

						conn.Send(building.NCBuildingUpdate(m.PlanetUUID))
						conn.Send(planet.NCUpdatePlanetResources())

					}					

				}


				again := true
				for len(planet.BuildInProgress) > 0 && again {
					again = false
					for i, UUID := range planet.BuildInProgress {
						building, _ := model_building.Get(UUID)
						if building == nil { continue }

						building.Update(model2.Fields{
							"UpgradeElapsed": building.UpgradeElapsed + 1,
						})
	
						if building.UpgradeElapsed >= building.UpgradeDuration {

							population := building.UpgradePopulation
							if building.Level == 0 {
								building.Update(model2.Fields{
									"Level": 1,
									"UpgradeInProgress": false,
									"UpgradeElapsed": 0,
									"UpgradeDuration": 0,
									"UpgradePopulation": 0,
									"TurnOnTime": time.Now().UnixNano(),
								})
							}

							planet.Update(model2.Fields{
								"BuildInProgress": append(planet.BuildInProgress[:i], planet.BuildInProgress[i+1:]...),
								"PopulationAvail": planet.PopulationAvail + float64(population),
								"TurnOnBuildings": append(planet.TurnOnBuildings, building.UUID),
							})

							if conn != nil {

								conn.Send(building.NCBuildingUpdate(m.PlanetUUID))
								conn.Send(planet.NCUpdatePlanetResources())

							}							

							again = true
							break

						} else {

							if conn != nil {
								conn.Send(building.NCBuildingUpdate(m.PlanetUUID))
							}

						}
					}
				}

				// 1. check turn on buildings
				again = true
				for len(planet.TurnOnBuildings) > 0 && again {
					again = false
					for i, UUID := range planet.TurnOnBuildings {
						building, _ := model_building.Get(UUID)

						if building.TurnOnTime < att {

							levelInfo := GetBuildingLevelInfo(building.Type, building.Level)
							popUsage, energyUsage := GetBuildingUsage(levelInfo)

							avail := false
							if popUsage <= planet.PopulationAvail && energyUsage <= planet.EnergyAvail {
								avail = true
							}

							if avail {

								building.Update(model2.Fields{
									"TurnOn": true,
									"TurnOnTime": 0,
								})

								planet.Update(model2.Fields{
									"TurnOnBuildings": append(planet.TurnOnBuildings[:i], planet.TurnOnBuildings[i+1:]...),
									"PopulationAvail": planet.PopulationAvail - float64(popUsage),
									"EnergyAvail": planet.EnergyAvail - float64(energyUsage),
								})

								// custom building logic
								Treators[building.Type].TurnOn(building, planet)

								if conn != nil {
									conn.Send(building.NCBuildingUpdate(m.PlanetUUID))
									conn.Send(planet.NCUpdatePlanetResources())
								}

								again = true
								break
							} else {
								// send problem turn on

							}

						}

					}
				}

				// custom work

				th := &TreatHint{ UpdateResource: false }
				for _, UUID := range planet.Buildings {
					building, _ := model_building.Get(UUID)
					if building == nil || !building.TurnOn { continue }

					t, ok := Treators[building.Type]
					if ok {
						t.TreatSecond(building, planet, th)
					}
				}

				if th.UpdateResource && conn != nil {
					conn.Send(planet.NCUpdatePlanetResources())
				}

				planet.Update(model2.Fields{
					"TreatTime": planet.TreatTime + 1000000000,
				})

				ptt = planet.TreatTime / 1000000000
			}

			// logger.String(fmt.Sprintf("treat planet %s", m.PlanetUUID.String()))

		}



		// todo: unlock planet
		unlockPlanet(planet)


	}
}

func (w *Worker) MT_BuildProc(planet *model_live_planet.Fields, m *WorkerMessage) {

	planet.Update(model2.Fields{
		"QueueBuildType": append(planet.QueueBuildType, m.Params["type"].(string)),
		"QueueBuildX": append(planet.QueueBuildX, m.Params["x"].(int)),
		"QueueBuildY": append(planet.QueueBuildY, m.Params["y"].(int)),
	})

}
