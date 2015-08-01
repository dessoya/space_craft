
package main

import (
	"io/ioutil"
	"encoding/json"

	// "fmt"

	"sc/errors"
)

type ConfigHTTP struct {
	Port uint16 `json:"port"`
}

type ConfigLogger struct {
	Path string `json:"path"`
}

type Config struct {
	Http ConfigHTTP			`json:"http"`
	Logger ConfigLogger		`json:"logger"`
}

var defaultConfig Config = Config{
	Logger: ConfigLogger {
		Path: "",
	},	
	Http: ConfigHTTP {
		Port: 9400,
	},
}

func readConfig() (*Config, *errors.Error) {

	data, err := ioutil.ReadFile("config.json")

	// fmt.Printf("%+v\n", data)

	if err != nil {
		return &defaultConfig, errors.New(err)
	}

	err = json.Unmarshal(data, &defaultConfig)
	// fmt.Printf("%+v\n", defaultConfig)

	if err != nil {
		return &defaultConfig, errors.New(err)
	}

	data, err = ioutil.ReadFile("local.config.json")
	// fmt.Printf("%+v\n", data)

	if err == nil {

		err = json.Unmarshal(data, &defaultConfig)
		// fmt.Printf("%+v\n", defaultConfig)

		if err != nil {
			return &defaultConfig, errors.New(err)
		}
	}

	return &defaultConfig, nil
}