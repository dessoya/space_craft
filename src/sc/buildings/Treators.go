
package buildings

import (
	model_live_planet "sc/models2/live_planet"
	model_building "sc/models2/building"	
)

type TreatHint struct {
	UpdateResource		bool
}

type Treator interface {
	TurnOn(*model_building.Fields, *model_live_planet.Fields)
	TreatSecond(*model_building.Fields, *model_live_planet.Fields, *TreatHint)
}

var Treators = make(map[string]Treator)

func init() {
	Treators["energy_station"] = TreatorEnergyStation{}
	Treators["mineral_mine"] = TreatorMineralMine{}
}
