package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
)

/*
10.0.11.10(TAC=10, eNB=100) -- s1ap --> mme((MME 1))
10.0.11.12(TAC=10, eNB 101) -- s1ap --> mme((MME 1))
10.0.11.13(TAC=20, eNB 100) -- s1ap --> mme((MME 1))
10.0.11.14(TAC=20, eNB 103) -- s1ap --> mme((MME 1))
10.0.11.15(TAC=30, eNB 104) -- s1ap --> mme((MME 1))
*/

type S1apPacket struct{
	SourceIP		string
	DestinationIP	string	//"10.255.100.178"
	TAC				byte	//OCTET STRING (SIZE (2))
	ENB				byte	`json:"eNB"`
	MME				byte	//0..255
	PacketSize		byte	//
}

var  S1apVariants = []S1apPacket{
	{	SourceIP: "10.0.11.10",	DestinationIP: "10.255.100.178", TAC: 10,	ENB: 100,	MME: 1,	PacketSize:	146	},
    {	SourceIP: "10.0.11.12",	DestinationIP: "10.255.100.178", TAC: 10,	ENB: 101,	MME: 1,	PacketSize:	146	},
    {	SourceIP: "10.0.11.13",	DestinationIP: "10.255.100.178", TAC: 20,	ENB: 100,	MME: 1,	PacketSize:	146	},
    {	SourceIP: "10.0.11.14",	DestinationIP: "10.255.100.178", TAC: 20,	ENB: 103,	MME: 1,	PacketSize:	146	},
    {	SourceIP: "10.0.11.15",	DestinationIP: "10.255.100.178", TAC: 30,	ENB: 104,	MME: 1,	PacketSize:	146	},
	{	SourceIP: "10.255.100.178",	DestinationIP: "10.0.11.10", TAC: 10,	ENB: 100,	MME: 1,	PacketSize:	146	},
	{	SourceIP: "10.255.100.178",	DestinationIP: "10.0.11.12", TAC: 10,	ENB: 101,	MME: 1,	PacketSize:	146	},
}

func main() {
	fmt.Println("URL:>http://127.0.0.1:8082")
	//
	for {
		packet := S1apVariants[rand.Intn(len(S1apVariants))]
		b := new(bytes.Buffer)
		json.NewEncoder(b).Encode(packet)
		//no err processing, just working
		http.Post("http://127.0.0.1:8082", "application/json; charset=utf-8", b)
	}
}