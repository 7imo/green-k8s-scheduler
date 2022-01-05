package main

import (
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

var (
	version string
)

func main() {
	// http server that is called by the kube-scheduler
	router := httprouter.New()
	router.GET("/", Index)
	router.GET("/version", Version)
	router.POST("/prioritize", Prioritize)
	// extender port
	log.Fatal(http.ListenAndServe(":80", router))
}
