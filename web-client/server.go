package main

import (
	"log"
	"net/http"
	"fmt"
	"github.com/gorilla/mux"
)

func main() {
	// Simple static webserver:
	r := mux.NewRouter()
	r.HandleFunc("/message", messagePost).Methods("POST")
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("root"))))
	log.Fatal(http.ListenAndServe(":8080",r))
}

func messagePost(w http.ResponseWriter, r *http.Request){
	r.ParseForm()
	log.Println(r.Form)
	fmt.Fprintf(w,"Here");
	switch r.Method {
		case "POST":
			fmt.Fprintf(w,"Here");
	}
}