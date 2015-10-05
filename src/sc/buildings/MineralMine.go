
package buildings

import (

	model_live_planet "sc/models2/live_planet"
	model_building "sc/models2/building"
	"sc/model2"
)

type TreatorMineralMine struct {
}


func (t TreatorMineralMine) TurnOn(b *model_building.Fields, p *model_live_planet.Fields) {

	// levelInfo := GetBuildingLevelInfo(b.Type, b.Level)	
/*
	p.Update(model2.Fields{
		"Energy": p.Energy + float64(levelInfo["energyProduced"].(int)),
		"EnergyAvail": p.EnergyAvail + float64(levelInfo["energyProduced"].(int)),
	})
	*/
}

func (t TreatorMineralMine) TreatSecond(b *model_building.Fields, p *model_live_planet.Fields, th *TreatHint) {

	levelInfo := GetBuildingLevelInfo(b.Type, b.Level)

	p.Update(model2.Fields{
		"Minerals": p.Minerals + levelInfo["mineralsInSec"].(float64),
	})

	th.UpdateResource = true

}