
package main

import (
	"io/ioutil"
	"encoding/json"

	// "fmt"

	"sc/errors"
	module_config "sc/config"
)

var defaultConfig = module_config.Config{
	Logger: module_config.ConfigLogger {
		Path: "",
	},	
	Http: module_config.ConfigHTTP {
		Port: 9400,
	},
	Auth: module_config.ConfigAuth {
		Methods: []string{"fake"},
	},
}

func readConfig() (*module_config.Config, *errors.Error) {

	data, err := ioutil.ReadFile("config.json")

	if err != nil {
		return &defaultConfig, errors.New(err)
	}

	err = json.Unmarshal(data, &defaultConfig)

	if err != nil {
		return &defaultConfig, errors.New(err)
	}

	data, err = ioutil.ReadFile("local.config.json")

	if err == nil {

		err = json.Unmarshal(data, &defaultConfig)

		if err != nil {
			return &defaultConfig, errors.New(err)
		}
	}

	return &defaultConfig, nil
}