package common

import (
	"encoding/json"
	"io/ioutil"
)

func LoadConf(filename string, conf interface{}) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, conf)
	if err != nil {
		return err
	}
	return nil
}
