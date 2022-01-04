package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	schedulerapi "k8s.io/kube-scheduler/extender/v1"
)

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome to the green K8s scheduler!\n")
}

// route to priority function
func Prioritize(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var buf bytes.Buffer
	body := io.TeeReader(r.Body, &buf)
	var extenderArgs schedulerapi.ExtenderArgs
	var hostPriorityList *schedulerapi.HostPriorityList
	if err := json.NewDecoder(body).Decode(&extenderArgs); err != nil {
		log.Println(err)
		hostPriorityList = &schedulerapi.HostPriorityList{}
	} else {
		hostPriorityList = prioritize(extenderArgs)
	}

	if response, err := json.Marshal(hostPriorityList); err != nil {
		log.Fatalln(err)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}
}

// health check
func Version(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, fmt.Sprint(version))
}
