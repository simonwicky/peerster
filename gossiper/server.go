package gossiper

import (
	"log"
	"net/http"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/simonwicky/Peerster/utils"
	"encoding/json"
	"io/ioutil"
)

func (g *Gossiper) HttpServerHandler() {
	// Simple static webserver:
	r := mux.NewRouter()
	r.HandleFunc("/message", g.messageHandler).Methods("POST","GET")
	r.HandleFunc("/node", g.nodeHandler).Methods("POST","GET")
	r.HandleFunc("/id", g.idHandler).Methods("GET")
	r.HandleFunc("/identifier", g.identifierHandler).Methods("GET", "POST")		
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("webclient"))))
	http.ListenAndServe(":8080",r)
}

func (g *Gossiper) messageHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method {
		case "POST":
			w.WriteHeader(http.StatusOK)
			var rumor utils.RumorMessage
			body, _ := ioutil.ReadAll(r.Body)
			json.Unmarshal(body, &rumor)
			if rumor.Text != "" {
				go g.uiRumorHandler(&utils.GossipPacket{Rumor:&rumor})
			}
		case "GET":
			w.WriteHeader(http.StatusOK)
			var rumors = g.getLatestRumors()
			rumorsJson, err := json.Marshal(rumors)
			if err != nil {
				log.Fatal("Unable to marshal rumors", err)
			}
			w.Write(rumorsJson)
	}
}

func (g *Gossiper) nodeHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method {
	case "POST":
		w.WriteHeader(http.StatusOK)
		peer, _ := ioutil.ReadAll(r.Body)
		go g.uiAddPeer(string(peer))
	case "GET":
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w,g.getKnownPeers()) 
	}

}

func (g *Gossiper) identifierHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method {
	case "GET":
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w,g.getIdentifiers()) 
	case "POST":
		w.WriteHeader(http.StatusOK)
		var mp utils.PrivateMessage
		body, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(body, &mp)
		if mp.Text != "" {
			fmt.Println(mp.Text)
			go g.uiPrivateMessageHandler(&utils.GossipPacket{Private :&mp})
		}
	}
}

func (g *Gossiper) idHandler(w http.ResponseWriter, r *http.Request){
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s",g.getName())
}