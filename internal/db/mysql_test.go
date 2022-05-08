package db

import (
	"log"
	"testing"
)

func TestGetUserById(t *testing.T) {
	defer Close()
	u, err := GetSpikeById("1")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(u)
}
