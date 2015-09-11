
package main

import (
	"io/ioutil"
	"encoding/json"
	"os"

	"fmt"

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
	Buildings: module_config.ConfigBuildings {
		PoolSize: 16,
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

	var localConfig string = "local.config.json"
	mode := "search"
	for _, arg := range os.Args {

		if arg[0] == '-' {
			mode = arg
			continue
		}

		switch mode {
		case "-localConfig":
			localConfig = arg
			mode = "search"
		}
	}
	fmt.Printf("use local config: %s\n", localConfig)

	data, err = ioutil.ReadFile(localConfig)

	if err == nil {

		err = json.Unmarshal(data, &defaultConfig)

		if err != nil {
			return &defaultConfig, errors.New(err)
		}
	}

	return &defaultConfig, nil
}