package db

import (
	"bank_spike_backend/internal/orm"
	"bank_spike_backend/internal/util/config"
	"log"
	"testing"
	"time"
)

func TestGetUserById(t *testing.T) {
	config.InitViper()
	defer Close()

}

func TestInsertOrder(t *testing.T) {
	config.InitViper()
	defer Close()
	o := &orm.Order{
		ID:         "",
		UserID:     "1",
		SpikeID:    "1",
		Quantity:   1,
		State:      "s",
		CreateTime: time.Now(),
	}
	err := InsertOrder(o)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(o.ID)
}
