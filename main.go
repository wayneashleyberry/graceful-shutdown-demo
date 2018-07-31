// Package main is a demo of the graceful shutdown added in Go 1.8.
// The command will run an http server with a catch-all route that will
// perform a slow sql query, sending an interrupt signal should wait for
// any requests to finish and then shut down the server.
// You can read more here https://golang.org/pkg/net/http/#Server.Shutdown
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-sql-driver/mysql"
)

const port = ":3000"
const username = "root"
const password = "root"
const database = "test"

func main() {
	conf := mysql.NewConfig()
	conf.Net = "tcp"
	conf.User = username
	conf.Passwd = password
	conf.DBName = database

	db, err := sql.Open("mysql", conf.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		if _, err := db.Exec("select sleep(30);"); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, err.Error())
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}

	server := http.Server{
		Addr:    port,
		Handler: http.HandlerFunc(handler),
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// We received an interrupt signal, shut down.
		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("listening on %s\n", server.Addr)

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed
}
