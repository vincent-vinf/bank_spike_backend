package db

import (
	"bank_spike_backend/internal/util/config"
	"log"
	"testing"
)

func TestGetUserById(t *testing.T) {
	config.InitViper()
	defer Close()
	u, err := GetSleepSpike()
	if err != nil {
		log.Fatal(err)
	}
	log.Println(u[0].StartTime)
}
