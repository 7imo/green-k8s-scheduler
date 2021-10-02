package main

import (
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

var (
	version string
)

//func init() {
//	rand.Seed(time.Now().UTC().UnixNano())
//}

func main() {
	router := httprouter.New()
	router.GET("/", Index)
	router.GET("/version", Version)
	router.POST("/filter", Filter)
	router.POST("/prioritize", Prioritize)

	log.Fatal(http.ListenAndServe(":80", router))
}
