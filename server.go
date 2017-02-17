package main

import (
  "fmt"
  "log"
  "net"
  "encoding/json"
)



// listens for incoming udp packets on port 3333 and prints them to stdout
func main() {

  // buffered channel to receive teensymsgs
  tch := make(chan TeensyMsg, 64)

  // buffered channel to receive websocket requests
  // TODO -> implement subroutine to do this
  wch := make(chan )

  // listen for teensy messages on udp port 3333 and pass them up channel
  go TeensySocket(tch)

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
    fmt.Printf("%v: %v\n", src_addr, msg)
  }
}
