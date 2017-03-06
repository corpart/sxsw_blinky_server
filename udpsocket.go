package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
)

// TeensyMsg - for decoding json messages from teensys:
// {
// 	"source": "<last digit of teensy ip address>",
// 	"flavor": "start_touch" | "end_touch" | "touch_beat",
// 	"choice": "left" | "right"
// }
type TeensyMsg struct {
	Source int    `json:"source"`
	Flavor string `json:"flavor"`
	Choice string `json:"choice"`
}

// TeensySocket - listens for incoming teensy messages over udp on port 3333
// converts json to teensymsg struct & sends up channel
func TeensySocket(ch chan TeensyMsg) {

	// create packetconn to listen for incoming udp packets on port 3333
	pc, err := net.ListenPacket("udp", ":3333")
	if err != nil {
		log.Fatal(err)
	}
	defer pc.Close()

	log.Println("listening for incoming teensy udp packets on port 3333")

	// loop and handle packets
	buffer := make([]byte, 1024)
	for {
		fmt.Printf("#")

		// try to read a new udp packet into buffer (blocks until success)
		msgsize, _, err := pc.ReadFrom(buffer)
		if err != nil {
			log.Printf("ERROR: %v", err)
		} else {

			// unmarshal buffer into a teensy message
			var msg TeensyMsg
			err := json.Unmarshal(buffer[:msgsize], &msg)
			if err != nil {
				log.Printf("ERROR unmarshalling %v: %v", string(buffer[:msgsize]), err)
			} else {

				// send message up channel
				select {
				case ch <- msg:
				default:
					log.Printf("ERROR: teensy message channel full!")
				}
			}
		}
	}
}
