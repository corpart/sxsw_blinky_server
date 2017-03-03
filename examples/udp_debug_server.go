package main

import (
	"fmt"
	"log"
	"net"
)

// listens for incoming udp packets on port 3333 and prints them to stdout
func main() {

	// listen for incoming udp packets on port 3333
	pc, err := net.ListenPacket("udp", ":3333")
	if err != nil {
		log.Fatal(err)
	}
	defer pc.Close()

	fmt.Println("listening for incoming udp packets on port 3333")

	// loop and print packets to standard out
	buffer := make([]byte, 1024)
	for {
		msg_size, src_addr, err := pc.ReadFrom(buffer)
		if err != nil {
			log.Fatal(err)
		}
		msg := string(buffer[:msg_size])

		// print contents of packet
		fmt.Printf("%v: % x\n", src_addr, msg)
	}
}
