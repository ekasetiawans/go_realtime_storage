package main

import "github.com/ekasetiawans/realtimestorage/realtimedb"

func main() {
	server := &realtimedb.RealtimeStorage{}
	server.Run(":8888")
}
