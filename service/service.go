package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	)
//
type S1apPacket struct{
	SourceIP		string
	DestinationIP	string
	TAC				byte				//in real life - OCTET STRING (SIZE (2))
	ENB				byte	`json:"eNB"`//in real life - 28bit
	MME				byte				//in real life - 0..255
	PacketSize		byte				//in real life - uint16
}
//DB obj ptr
var pDB				*sql.DB
//queue
var qToDB 			[]S1apPacket
var qToDBSync 		sync.Mutex
//
func main() {
	log.Println("Connecting to ClickHouse DB...\r\n")
	pDB = connectToCH()
	log.Println("Starting HTTP service, waiting on port 8082...\r\n")
	//req mux
	httpMux := http.NewServeMux()
	// /TAC/ req by -> all eNB
	httpMux.Handle("/TAC/", http.HandlerFunc(requestByTAC))
	// /MME/ req by MME -> all eNB
	httpMux.Handle("/MME/", http.HandlerFunc(requestByMME))
	// /D3/ req all eNB
	httpMux.Handle("/D3/", http.HandlerFunc(requestForDraw))
	// /simplex/ req simplex streams -> all src dst IP , eNB
	httpMux.Handle("/simplex/", http.HandlerFunc(requestSimplex))
	// - POST req
	httpMux.Handle("/", http.HandlerFunc(defaultHandler))
	//
	log.Fatal(http.ListenAndServe(":8082", httpMux))
}
//
func requestByTAC(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		if(len(r.RequestURI) > 0){
			slice := strings.Split(r.RequestURI, "/") //from /TAC/...
			value := slice[len(slice)-1:][0]
			TAC, err := strconv.Atoi(value)
			if err != nil {
				log.Printf("Err get TAC value! Value - %s, err - %d\r\n", value, err)
				http.Error(w, "Please send correct request, like /TAC/10", 400)
				return
			}
			//
			slice = selectbyTACFromDB(pDB, uint8(TAC))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(slice)
			//
			log.Printf("TAC %s requested\r\n", value)
		}
	} else{
		log.Println("Err processing TAC request. HTTP method - %s\r\n", r.Method)
	}
}
//
func requestByMME(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		if(len(r.RequestURI) > 0){
			slice := strings.Split(r.RequestURI, "/") //from /MME/...
			value := slice[len(slice)-1:][0]
			TAC, err := strconv.Atoi(value)
			if err != nil {
				log.Printf("Err get MME value! Value - %s, err - %d\r\n", value, err)
				http.Error(w, "Please send correct request, like /MME/1", 400)
				return
			}
			//
			slice = selectbyMMEFromDB(pDB, uint8(TAC))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(slice)
			//
			log.Printf("MME %s requested\r\n", value)
		}
	} else{
		log.Println("Err processing MME request. HTTP method - %s\r\n", r.Method)
	}
}
//
func requestForDraw(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		type D3 struct {
			Nodes   []string `json:"nodes"`
			Links	[]string `json:"links"`
		}
		tmp := D3{}
		tmp.Nodes, tmp.Links = selectForDrawFromDB(pDB)
		w.Header().Set("Content-Type", "application/json")
		//res, _ := json.Marshal(&tmp)
		json.NewEncoder(w).Encode(&tmp)
		log.Println("Data for D3.js requested\r\n",)
	} else{
		log.Printf("Err processing  D3.js request. HTTP method - %s\r\n", r.Method)
	}
}
//
func requestSimplex(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		slice := selectSimplexFromDB(pDB)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(slice)
		//
		log.Println("Simplex streams requested\r\n",)
	} else{
		log.Printf("Err processing Simplex request. HTTP method - %s\r\n", r.Method)
	}
}
//
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var pack S1apPacket
		if r.Body == nil {
			log.Println("There are no body in POST request\r\n")
			http.Error(w, "No request body", 400)
			return
		}
		err := json.NewDecoder(r.Body).Decode(&pack)
		if err != nil {
			log.Println("Err while parsing JSON in POST request\r\n")
			http.Error(w, err.Error(), 400)
			return
		}
		//to queue
		qToDBSync.Lock()
		defer qToDBSync.Unlock()
		qToDB = append(qToDB, pack)
	} else{
		http.Error(w, "using:\r\n 1. Request by TAC: .../TAC/{value}\r\n 2. Request by MME: .../MME/{value}\r\n 3. Request simplex streams: .../simplex/\r\n 4. Request data for D3.js: .../D3/\r\n", 400)
	}
}