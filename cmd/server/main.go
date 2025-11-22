package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/yourusername/always-at-morg/internal/server"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP service address")
	flag.Parse()

	srv := server.NewServer()

	http.HandleFunc("/ws", srv.HandleWebSocket)

	log.Printf("Starting server on %s", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
