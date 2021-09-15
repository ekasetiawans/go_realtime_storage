package main

import (
	"log"

	"github.com/ekasetiawans/realtimestorage/realtimedb"
)

func main() {
	server := &realtimedb.RealtimeStorage{}
	log.Fatal(server.Run(":8888"))
}
