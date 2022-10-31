package main

import (
	"encoding/json"
	"os"
)

type ConfigFile struct {
	MaxConn uint8  `json:"max_conn"`
	Port    string `json:"port"`
	Host    string `json:"host"`
}

func GetConfigFileData(fileName string) (ConfigFile, error) {
	_data := ConfigFile{}
	confFile, err := os.Open(fileName)
	if err != nil {
		return _data, err
	}

	confBuff := make([]byte, 256)
	n, err := confFile.Read(confBuff)
	if err != nil {
		return _data, err
	}

	if err = confFile.Close(); err != nil {
		return _data, err
	}

	if err = json.Unmarshal(confBuff[:n], &_data); err != nil {
		return _data, err
	}

	return _data, nil
}
