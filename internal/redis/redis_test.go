package redisx

import (
	"bank_spike_backend/internal/util/config"
	"context"
	"log"
	"testing"
)

func TestGet(t *testing.T) {
	config.InitViper()
	get, err := Get(context.Background(), "123")
	log.Println(err)
	log.Println(get)
}
