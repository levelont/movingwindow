package main

import (
	"context"
	"movingwindow/api"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	server := api.NewServer(api.ParseEnvironment())
	server.Logger.Println("Server is starting...")
	server.Routes()

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	signal.Notify(quit, os.Kill)

	go func() {
		signal := <-quit
		server.Logger.Printf("Server received signal '%v'.", signal)

		if err := server.PersistState(); err != nil {
			server.Logger.Fatalf("Could not save state to disk: %v\n", err)
		}

		server.Logger.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			server.Logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		server.CloseChannels()
		close(done)
	}()

	server.Logger.Println("Server is ready to handle requests at", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		server.Logger.Fatalf("Could not listen on %s: %v\n", server.Addr, err)
	}

	<-done
	server.Logger.Println("Server stopped")
}
