package main

import (
  "fmt"
  "log"
  "net"
)

// sends an example json message over udp to localhost:3333 and exits
func main() {

  msg := "{\"source\": 101, \"flavor\": \"start_touch\", \"choice\": \"right\"}"

  // connect to local udp server on port 3333
  conn, err := net.Dial("udp", "127.0.0.1:3333")
  if err != nil {
  	log.Fatal(err)
  }
  defer conn.Close()

  // write message to server
  _, err = conn.Write([]byte(msg))
  if err != nil {
    log.Fatal(err)
  }

  fmt.Printf("wrote %v to local server on port 3333\n", msg)
}
