package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sing3demons/go-api/logger"
	"github.com/sing3demons/go-api/middleware"
	"github.com/sing3demons/go-api/xhttp"
)

func main() {
	logg := logger.NewLogger()
	r := http.DefaultServeMux

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s := xhttp.ServiceConfig{
			Method: "GET",
			Name:   "Hello",
			Url:    "https://jsonplaceholder.typicode.com/posts",
			// Url:        "https://jsonplaceholder.typicode.com/posts/{id}",
			System:     "Hello",
			Timeout:    10,
			StatusCode: "200",
		}

		b, err := s.Call(r.Context(), xhttp.Option{
			Query: map[string]string{
				// ?_start=10&_limit=10
				"_start": "1",
				"_limit": "5",
			},
			// Param: map[string]string{
			// 	"{id}": "1",
			// },
		})
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		// fmt.Println(string(b))
		w.Write(b)
	})

	Port := "8080"

	svc := http.Server{
		Addr:    fmt.Sprintf(":%s", Port),
		Handler: middleware.Logger(r),
	}

	stopServer := make(chan os.Signal, 1)
	signal.Notify(stopServer, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stopServer)
	// channel to listen for errors coming from the listener.
	serverErrors := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()

		logg.Info(fmt.Sprintf("Server started on port: %s", Port))
		serverErrors <- svc.ListenAndServe()
	}(&wg)

	// blocking run and waiting for shutdown.
	select {
	case err := <-serverErrors:
		log.Fatalf("error: starting rest api server: %v\n", err)
	case <-stopServer:
		fmt.Println("server received stop signal")
		// asking listener to shutdown
		err := svc.Shutdown(context.Background())
		if err != nil {
			log.Fatalf("graceful shutdown did not complete: %v\n", err)
		}
		wg.Wait()
		fmt.Println("server was shut down gracefully")
	}
}
