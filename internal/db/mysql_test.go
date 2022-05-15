package db

import (
	"bank_spike_backend/internal/util/config"
	"testing"
)

func TestGetUserById(t *testing.T) {
	config.InitViper()
	defer Close()

}
