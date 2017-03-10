package main

import (
	"log"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"net"
)

// Led - type for reading led data from json file
type Led struct {
	IP    string  `json:"ip"`
	Index int     `json:"index"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Z     float64 `json:"z"`
}

func main() {
	// read led position file and create bllnkr with data
	leddata, err := ioutil.ReadFile("../led_locations.json")
	// log.Println(len(leddata))
	if err != nil {
		log.Fatal(err)
	}

	// unmarshal json data into a slice of Leds
	var leds []Led
	err = json.Unmarshal(leddata, &leds)
	if err != nil {
		log.Fatal(err)
	}

	blank := [3]int16{0, 0, 0}

	for _, led := range leds {

		ip := led.IP

		// write color values for lamps leds to buffer
		buf := new(bytes.Buffer)
		for i := 0; i < 16; i++ {
			err := binary.Write(buf, binary.BigEndian, blank)
			if err != nil {
				log.Println("ERROR: binary write of color to buffer failed:", err)
			}
		}

		// connect to local udp server on port 3333
		dst := ip + ":" + "3333"
		conn, err := net.Dial("udp", dst)
		if err != nil {
			log.Printf("ERROR: failed get udp conn for lamp at %v: %v", dst, err)
		}
		defer conn.Close()

		// write buffer to server
		_, err = conn.Write(buf.Bytes())
		if err != nil {
			// log.Printf("ERROR: failed to send udp message to lamp at %v: %v", dst, err)
		}

		log.Printf("sent udp packet to %v: %v", dst, buf.Bytes())
	}
	log.Println("blanked leds!")
}