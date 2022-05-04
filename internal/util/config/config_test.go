package config

import (
	"log"
	"testing"
)

func TestConfig(t *testing.T) {
	cfg, err := GetConfig()
	if err != nil {
		t.Fatal(err.Error())
	}
	log.Println(cfg.JWT.Secret)
}
