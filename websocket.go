package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/net/websocket"
)

// DataMsg - for sending messages to data clients:
// {
// 	"source": "<last digit of teensy ip address>",
// 	"flavor": "start_touch" | "end_touch" | "new_word",
// 	"choice": "left" | "right"
//  "word": "<new word as string>"
// }
type DataMsg struct {
	Source int    `json:"source"`
	Flavor string `json:"flavor"`
	Choice string `json:"choice"`
	Word   string `json:"word"`
	Color  []int  `json:"color"`
}

// DataClient - holds channel to goroutine with websocket connection to client
type DataClient struct {
	MsgCh chan DataMsg
	Dest  string // ws.Request().RemoteAddr
}

// DataSocket - server routine that listens for incoming websocket clients
func DataSocket(dch chan DataClient, tch chan TeensyMsg) {

	// spin off channel to accept datamsgs for each client and wait on it
	http.Handle("/", websocket.Handler(func(ws *websocket.Conn) {

		// pass dataclient with message chan back to server over dataclient chan
		dc := DataClient{make(chan DataMsg, 64), ws.Request().RemoteAddr}
		select {
		case dch <- dc:
		default:
			log.Printf("ERROR: dch full!")
		}

		// start goroutine to forward teensy messages from dataclient
		go func(iws *websocket.Conn) { // unique inner ws is passed with each call
			for {
				fmt.Printf("{-%v-", iws.Request().RemoteAddr)
				var reply string

				if err := websocket.Message.Receive(iws, &reply); err != nil {
					log.Println("ERROR: failed to receive reply from client " + dc.Dest)
					return
				}

				// unmarshal reply into a teensy message
				var msg TeensyMsg
				err := json.Unmarshal([]byte(reply), &msg)
				if err != nil {
					log.Printf("ERROR unmarshalling %v: %v", reply, err)
				} else {

					// send message down teensy message channel
					select {
					case tch <- msg:
					default:
						log.Printf("ERROR: tch full! discarding message %v", msg)
					}
				}
				fmt.Printf("}\n")
			}
		}(ws)

		// loop and forward messages from datamsg channel to remote client
		for dm := range dc.MsgCh {
			fmt.Printf("{+%v+", dc.Dest)
			msg, err := json.Marshal(dm)
			if err != nil {
				log.Println("ERROR: failed to marshal data message!")
			} else {
				log.Println("forwarding msg to: " + dc.Dest + ": " + string(msg))
				if err = websocket.Message.Send(ws, string(msg)); err != nil {
					log.Println("ERROR: failed to send msg '" + string(msg) + "' to client " + dc.Dest)
					return
				}
			}
			fmt.Printf("}\n")
		}
	}))

	log.Println("listening for websocket data clients at ws://localhost:8888")

	if err := http.ListenAndServe(":8888", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
