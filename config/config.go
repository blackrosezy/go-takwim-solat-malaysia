package config

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	BaseURL  string   `json:"baseURL"`
	Zones    []string `json:"zones"`
	PoolSize int      `json:"poolSize"`
}

func GetConfig(fileName string) (Config, error) {
	var c Config
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(file, &c)
	if err != nil {
		return c, err
	}
	return c, nil
}
