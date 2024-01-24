package main

import (
	"log"
	"net/http"
	"rq/config"
	"rq/files"
	"rq/storage"
)

func main() {
	// TODO: Move profile selection to CLI arg / env var
	if err := config.LoadConfigFile("default"); err != nil {
		log.Fatal("Could not load config file", err)
	}

	addServerExcludedHeaders(&config.Config.Server.ExcludedHeaders)

	mux := http.NewServeMux()

	databaseStore, err := storage.NewSqliteRecordStore(config.Config.Database.Filepath)
	fileStore, err := files.NewDiskFileStore()

	if err != nil {
		log.Fatal("error opening database connection: ", err)
	}
	recordServer := &RecordServer{databaseStore, fileStore}
	//HttpRequestHandler := http.HandlerFunc(QueueHttpHandler)

	mux.Handle("/api/rq/http", RqHttpMiddleware(recordServer))
	log.Fatal(http.ListenAndServe(":8080", mux))

}
