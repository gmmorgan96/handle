// Command visualizer serves the multi-service project viewer.
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/gmmorgan96/handle/internal/api"
	"github.com/gmmorgan96/handle/internal/ui"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	root := flag.String("root", ".", "project root containing .project/services.yaml")
	flag.Parse()

	mux := http.NewServeMux()
	mux.Handle("/api/", api.NewHandler(*root))
	mux.Handle("/", ui.NewHandler(*root))

	server := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("serving project %q at http://%s", *root, *addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
