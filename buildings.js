'use strict'

var fs			= require('fs')
  , util		= require('util')

var energy_station = require(__dirname + '\\buildings\\energy_station.js')

var config = { rate: 1 }

function makeGOstruct(data) {

    var text = 'map[string]interface{}{\n'

    text += '	"levelParams": []interface{}{\n'
    for(var i = 0, c = data.levelParams, l = c.length; i < l; i++) {

    var item = c[i]
    text += '		map[string]interface{}{\n'
    for(var k in item) {
    	var v = item[k]
    	text += '			"' + k + '":'
    	if(typeof v === 'object') {
    		text += 'map[string]interface{}{\n'
    		for(var k2 in v) {
    			var v2 = v[k2]
			   	text += '				"' + k2 + '":' + v2 + ',\n'
    		}
    		text += '			},\n'
    	}
    	else {
    		text += v + ',\n'
    	}
    }
    text += '		},\n'


    }
    text += '	},\n'
    
    text += '}\n\n'

	return text
}

var text = 'package buildings\n\n'

text += 'var EnergyStation = ' + makeGOstruct(energy_station.getLevelsInfo(config))

fs.writeFileSync(__dirname + '\\src\\sc\\buildings\\buildings.go', text)

// console.log(makeGOstruct(energy_station.getLevelsInfo(config)))

// console.log(util.inspect(energy_station.getLevelsInfo(config),{depth:null}))