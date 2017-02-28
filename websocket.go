package main

import (
    "golang.org/x/net/websocket"
    "log"
    "net/http"
    "encoding/json"
)

// for sending messages to data clients:
// {
// 	"source": "<last digit of teensy ip address>",
// 	"flavor": "start_touch" | "end_touch" | "new_word",
// 	"choice": "left" | "right"
//  "word": "<new word as string>"
// }
type DataMsg struct {
  Source  int     `json:"source"`
  Flavor  string  `json:"flavor"`
  Choice  string  `json:"choice"`
  Word    string  `json:"word"`
}

type DataClient struct {
  MsgCh chan DataMsg
  Dest  string // ws.Request().RemoteAddr
}

func DataSocket(dch chan DataClient, tch chan TeensyMsg) {

  // spin off channel to accept datamsgs for each client and wait on it
  http.Handle("/", websocket.Handler(func (ws *websocket.Conn) {

    // pass dataclient with message chan back to server over dataclient chan
    dc := DataClient{make(chan DataMsg, 64), ws.Request().RemoteAddr}
    dch <- dc

    // start goroutine to forward teensy messages from dataclient
    go func (iws *websocket.Conn) { // unique inner ws is passed with each call
      for {
        var reply string

        if err := websocket.Message.Receive(iws, &reply); err != nil {
            log.Println("ERROR: failed to receive reply from client " + dc.Dest)
            break
        }

        // unmarshal reply into a teensy message
        var msg TeensyMsg
        err := json.Unmarshal([]byte(reply), &msg)
        if err != nil {
          log.Printf("ERROR unmarshalling %v: %v", reply, err)
        } else {

          // send message down teensy message channel
          tch <- msg
        }
      }
    }(ws)

    // loop and forward messages from datamsg channel to remote client
    for dm := range dc.MsgCh {
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
    }
  }))

  log.Println("listening for websocket data clients at ws://localhost:8888")

  if err := http.ListenAndServe(":8888", nil); err != nil {
      log.Fatal("ListenAndServe:", err)
  }
}
