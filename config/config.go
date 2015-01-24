package config

import (
	"os"

	log "github.com/Sirupsen/logrus"
)

const (
	DATA_DIR = "/var/lib/docker-bastion"
)

func init() {
	// If path is already a directory, MkdirAll does nothing and returns nil.
	err := os.MkdirAll(DATA_DIR, 0755)
	if err != nil {
		log.Fatal(err)
	}
}
