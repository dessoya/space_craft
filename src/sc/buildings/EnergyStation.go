
package buildings

import (
	model_live_planet "sc/models2/live_planet"
	model_building "sc/models2/building"
	"sc/model2"
)

type TreatorEnergyStation struct {
}


func (t TreatorEnergyStation) TurnOn(b *model_building.Fields, p *model_live_planet.Fields) {
	levelInfo := GetBuildingLevelInfo(b.Type, b.Level)	

	p.Update(model2.Fields{
		"Energy": p.Energy + float64(levelInfo["energyProduced"].(int)),
		"EnergyAvail": p.EnergyAvail + float64(levelInfo["energyProduced"].(int)),
	})
}

func (t TreatorEnergyStation) TreatSecond(b *model_building.Fields, p *model_live_planet.Fields, th *TreatHint) {
}
