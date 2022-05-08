package main

import (
	"bank_spike_backend/internal/access"
	"bank_spike_backend/internal/db"
	redisx "bank_spike_backend/internal/redis"
	"context"
	"log"
	"testing"
)

func TestIsAccessible(t *testing.T) {
	defer db.Close()
	defer redisx.Close()
	var a accessServer
	req := &access.AccessReq{
		UserId:  "1",
		SpikeId: "1",
	}
	accessible, err := a.IsAccessible(context.Background(), req)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(accessible.Result)
	log.Println(accessible.Reason)
}
