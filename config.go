
package main

import (
	"io/ioutil"
	"encoding/json"

	"sc/errors"
)

type ConfigHTTP struct {
		Port uint16 `json:"port"`
}

type Config struct {
	Http ConfigHTTP `json:"http"`
}

var defaultConfig Config = Config{
	Http: ConfigHTTP {
		Port: 9400,
	},
}

func readConfig() (*Config, *errors.Error) {

	data, err := ioutil.ReadFile("config.json")

	if err != nil {
		return &defaultConfig, errors.New(err)
	}

	err = json.Unmarshal(data, &defaultConfig)

	if err != nil {
		return &defaultConfig, errors.New(err)
	}

	return &defaultConfig, nil
}