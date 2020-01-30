package gossiper

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/simonwicky/Peerster/utils"
)

func (g *Gossiper) HttpServerHandler(port string) {
	// Simple static webserver:
	r := mux.NewRouter()
	r.HandleFunc("/proxies", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		//utils.LogObj.Warn("proxies ", g.proxyPool)
		paths := [][]string{}
		var proxiesJson []map[string]string = make([]map[string]string, 0)
		var proxies []*Proxy
		if g.settings.ProxyMocking {
			if g.Name == "Alice" {
				mockProxies := []*Proxy{&Proxy{
					IP:    "127.0.0.1:5008",
					Paths: [2]string{"127.0.0.1:5001", "127.0.0.1:5001"},
				}, &Proxy{
					IP:    "127.0.0.1:5009",
					Paths: [2]string{"127.0.0.1:5003", "127.0.0.1:5004"},
				}}
				proxies = mockProxies
			} else {
				proxies = []*Proxy{}
			}
		} else {
			proxies = g.proxyPool.proxies
		}
		for _, proxy := range proxies {
			paths = append(paths, proxy.Paths[:])
			proxiesJson = append(proxiesJson, map[string]string{
				"IP": proxy.IP,
				"_1": proxy.Paths[0],
				"_2": proxy.Paths[1],
			})
		}
		data, err := json.Marshal(map[string][]map[string]string{
			"proxies": proxiesJson,
		})
		if err != nil {
			utils.LogObj.Fatal(err.Error())
		} else {
			w.Write(data)
		}

	})
	r.HandleFunc("/message", g.messageHandler).Methods("POST", "GET")
	r.HandleFunc("/node", g.nodeHandler).Methods("POST", "GET")
	r.HandleFunc("/file", g.fileHandler).Methods("POST", "GET")
	r.HandleFunc("/id", g.idHandler).Methods("GET")
	r.HandleFunc("/identifier", g.identifierHandler).Methods("GET", "POST")
	r.HandleFunc("/download", g.downloadHandler).Methods("POST")
	r.HandleFunc("/keywords", g.keywordsHandler).Methods("POST")
	r.HandleFunc("/downloadsearch", g.downloadSearchedHandler).Methods("POST")
	r.HandleFunc("/TLCNames", g.TLCNamesHandler).Methods("GET")
	r.HandleFunc("/round", g.roundHandler).Methods("GET")
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("webclient"))))
	http.ListenAndServe(":"+port, r)
}

func (g *Gossiper) messageHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		w.WriteHeader(http.StatusOK)
		var rumor utils.RumorMessage
		body, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(body, &rumor)
		if rumor.Text != "" {
			go g.uiRumorHandler(&utils.GossipPacket{Rumor: &rumor})
		}
	case "GET":
		w.WriteHeader(http.StatusOK)
		var rumors = g.getLatestRumors()
		rumorsJson, err := json.Marshal(rumors)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Unable to marshal rumors")
			fmt.Fprintln(os.Stderr, err)
			return
		}
		w.Write(rumorsJson)
	}
}

func (g *Gossiper) nodeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		w.WriteHeader(http.StatusOK)
		peer, _ := ioutil.ReadAll(r.Body)
		go g.uiAddPeer(string(peer))
	case "GET":
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, g.getKnownPeers())
	}

}

func (g *Gossiper) identifierHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, g.getIdentifiers())
	case "POST":
		w.WriteHeader(http.StatusOK)
		var mp utils.PrivateMessage
		body, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(body, &mp)
		if mp.Text != "" {
			go g.uiPrivateMessageHandler(&utils.GossipPacket{Private: &mp})
		}
	}
}

func (g *Gossiper) fileHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		w.WriteHeader(http.StatusOK)
		body, _ := ioutil.ReadAll(r.Body)
		fileName := string(body)
		g.uiFileIndexHandler(fileName)
	}
}

func (g *Gossiper) downloadHandler(w http.ResponseWriter, r *http.Request) {
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
			fmt.Fprintln(os.Stderr, "Malformed hash")
			return
		}
		if destination == "" || filename == "" {
			fmt.Fprintln(os.Stderr, "Incorrect parameters")
			return
		}
		g.uiFileDownloadHandler(hash, destination, filename)
	}
}

func (g *Gossiper) downloadSearchedHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		w.WriteHeader(http.StatusOK)
		body, _ := ioutil.ReadAll(r.Body)
		index, _ := strconv.Atoi(string(body))
		match := g.getFileSearcher().getMatches()[index]
		fmt.Fprintf(os.Stderr, "Request to download %s\n", match.name)
		g.uiFileDownloadHandler(match.fileData.metafileHash, "", "")
	}
}

func (g *Gossiper) keywordsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		w.WriteHeader(http.StatusOK)
		body, _ := ioutil.ReadAll(r.Body)
		keywords := string(body)
		matches := g.uiFileSearchHandler(keywords)
		fmt.Fprintf(w, matches)
	}
}
func (g *Gossiper) TLCNamesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s", g.uiGetTLCMessages())
	}
}

func (g *Gossiper) idHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", g.getName())
}

func (g *Gossiper) roundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if !g.hw3ex3 {
		fmt.Fprintf(w, "No Round")
	}
}
