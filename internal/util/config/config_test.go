package config

import (
	"log"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	InitViper()
	c := GetConfig()
	log.Println(c)
	time.Sleep(time.Second * 10)
}
