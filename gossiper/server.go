package gossiper

import (
	"net/http"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/simonwicky/Peerster/utils"
	"encoding/json"
	"io/ioutil"
	"encoding/hex"
	"os"
)

func (g *Gossiper) HttpServerHandler() {
	// Simple static webserver:
	r := mux.NewRouter()
	r.HandleFunc("/message", g.messageHandler).Methods("POST","GET")
	r.HandleFunc("/node", g.nodeHandler).Methods("POST","GET")
	r.HandleFunc("/file", g.fileHandler).Methods("POST","GET")
	r.HandleFunc("/id", g.idHandler).Methods("GET")
	r.HandleFunc("/identifier", g.identifierHandler).Methods("GET", "POST")
	r.HandleFunc("/download", g.downloadHandler).Methods("POST")
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
				fmt.Fprintln(os.Stderr,"Unable to marshal rumors")
				fmt.Fprintln(os.Stderr,err)
				return
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
			go g.uiPrivateMessageHandler(&utils.GossipPacket{Private :&mp})
		}
	}
}

func (g *Gossiper) fileHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method {
	case "POST":
		w.WriteHeader(http.StatusOK)
		body, _ := ioutil.ReadAll(r.Body)
		fileName := string(body)
		g.uiFileIndexHandler(fileName)
	}
}

func (g *Gossiper) downloadHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method {
	case "POST":
		w.WriteHeader(http.StatusOK)
		body, _ := ioutil.ReadAll(r.Body)
		var parameters []string
		json.Unmarshal(body, &parameters)
		filename := parameters[0]
		destination := parameters[1]
		hash, err := hex.DecodeString(parameters[2])
		if err != nil {
			fmt.Fprintln(os.Stderr,"Malformed hash")
			return
		}
		if destination == ""{
			fmt.Fprintln(os.Stderr,"Incorrect parameters")
			return
		}
		dr := utils.DataRequest{
			Origin: g.Name,
			Destination: destination,
			HopLimit:10,
			HashValue: hash,
		}
		g.NewDatadownloader(&dr, filename)
	}
}

func (g *Gossiper) idHandler(w http.ResponseWriter, r *http.Request){
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s",g.getName())
}