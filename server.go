package main

import (
  "log"
)



// listens for incoming udp packets on port 3333 and prints them to stdout
func main() {

  // buffered channel to receive udp teensymsgs
  tch := make(chan TeensyMsg, 64)

  // buffered channel to receive websocket data clients
  dch := make(chan DataClient, 16)

  // index of connected clients
  dcdex := make(map[string]DataClient)

  // listen for teensy messages on udp port 3333 and pass them up channel
  go TeensySocket(tch)

  // listen for data clients on ws port 8888 and pass them up channel
  go DataSocket(dch)

  //
  for {
    select {
    case tm := <-tch:

      // log message
      log.Printf("received: %+v", tm)

      // broadcast to data clients
      for _, dc := range dcdex {
        dc.MsgCh <- DataMsg{
          Source: tm.Source, Flavor: tm.Flavor, Choice: tm.Choice}
      }

    case dc := <-dch:
      log.Printf("new data client at %v", dc.Dest)
      dcdex[dc.Dest] = dc // append data client to index
    }
  }
}
