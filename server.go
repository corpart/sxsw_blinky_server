package main

import (
  "log"
  "os"
)

var CYCLE_DELAY = int64(20000)  // delay between cycling words in ms

// listens for incoming udp packets on port 3333 and prints them to stdout
func main() {

  // ordered list of vote station addresses & last beats
  vote_stns := []int{101, 102, 103}
  vote_stn_beats := make([]int64, len(vote_stns))
  last_cycle := NowMs() // timestamp of last word cycle event

  f, err := os.OpenFile("wordlog.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
  if err != nil {
  	log.Fatal(err)
  }
  wrdr := NewWrdr(vote_stns, f)
  defer f.Close() // close word log file on exit

  // buffered channel to receive udp teensymsgs
  tch := make(chan TeensyMsg, 64)

  // buffered channel to receive websocket data clients
  dch := make(chan DataClient, 16)

  // index of connected clients
  dcdx := make(map[string]DataClient)

  // listen for teensy messages on udp port 3333 and pass them up channel
  go TeensySocket(tch)

  // listen for data clients on ws port 8888 and pass them up channel
  go DataSocket(dch, tch)

  // loop over
  for {
    select {

    // incoming teensy message channel
    case tm := <-tch:

      // log message
      log.Printf("received: %+v", tm)

      switch tm.Flavor {

      case "touch_beat": // log heartbeat
        var sdx int = -1
        for i, src := range vote_stns {
          if src == tm.Source {
            sdx = i
          }
        }
        if sdx > -1 {
          log.Printf("ERROR: unrecognized touch beat source '%v'\n", tm.Source)
        } else {
          vote_stn_beats[sdx] = NowMs() // set last beat time to now
        }

      case "start_touch", "end_touch": // broadcast to data clients
        wrdr.LogTouch(tm.Source, tm.Flavor, tm.Choice)

        dm := DataMsg{Source: tm.Source, Flavor: tm.Flavor, Choice: tm.Choice}
        bcastMsg(dm, dcdx)
      }

    // incoming data client channel
    case dc := <-dch:
      log.Printf("new data client at %v", dc.Dest)
      dcdx[dc.Dest] = dc // append data client to index

    // cycle words at intervals
    default:
      nw := NowMs()
      if last_cycle + CYCLE_DELAY < nw {
        last_cycle = nw // reset last cycle timestamp
        dm := wrdr.CycleWrd() // pick a new word & gen message
        bcastMsg(dm, dcdx) // broadcast message to data clients
      }
    }
  }
}

func bcastMsg(dm DataMsg, dcdx map[string]DataClient) {
  for _, dc := range dcdx {
    dc.MsgCh <- dm
  }
}
