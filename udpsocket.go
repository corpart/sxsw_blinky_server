package main

import (
  "fmt"
  "log"
  "net"
  "encoding/json"
)

// for decoding json messages from teensys:
// {
// 	"source": "<last digit of teensy ip address>",
// 	"flavor": "start_touch" | "end_touch" | "touch_beat",
// 	"choice": "left" | "right"
// }
type TeensyMsg struct {
  Source  int
  Flavor  string
  Choice  string
}

// listens for incoming json teensy messages over udp on port 3333
// converts to teensymsg struct & sends up channel
func TeensySocket(ch chan TeensyMsg) {

  // create packetconn to listen for incoming udp packets on port 3333
  pc, err := net.ListenPacket("udp", ":3333")
  if err != nil {
  	log.Fatal(err)
  }
  defer pc.Close()

  fmt.Println("listening for incoming teensy udp packets on port 3333")

  // loop and handle packets
  buffer := make([]byte, 1024)
  for {

    // try to read a new udp packet into buffer (blocks until success)
    msg_size, src_addr, err := pc.ReadFrom(buffer)
    if err != nil {
      log.Error(err)
    } else {

      // unmarshal buffer into a teensy message
      var msg TeensyMsg
      err := json.Unmarshal(buffer[:msg_size], &msg)
      if err != nil {
        log.Error(err)
      } else {

        // send message up channel
        ch <- msg
      }
    }
  }
}
