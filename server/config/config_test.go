package config

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	filename := "../conf/server.json"

	err := LoadConfig(filename)
	if err != nil {
		t.Fatalf("LoadConfig failed,err:%s", err.Error())
	}
	PrintServerConf(GServerConf)
}
