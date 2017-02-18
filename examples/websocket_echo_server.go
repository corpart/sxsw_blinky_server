package main

import (
    "golang.org/x/net/websocket"
    "fmt"
    "log"
    "net/http"
)

func Echo(ws *websocket.Conn) {
    var err error

    clntaddr := ws.Request().RemoteAddr

    fmt.Println("opened socket for client at " + clntaddr)

    for {
        var reply string

        if err = websocket.Message.Receive(ws, &reply); err != nil {
            fmt.Println("Can't receive from client at " + clntaddr)
            break
        }

        fmt.Println("Received back from client: " + reply)

        msg := "Received:  " + reply
        fmt.Println("Sending to client: " + msg)

        if err = websocket.Message.Send(ws, msg); err != nil {
            fmt.Println("Can't send")
            break
        }
    }
}

func main() {
    http.Handle("/", websocket.Handler(Echo))

    fmt.Println("listening for websocket clients at ws://localhost:8888")

    if err := http.ListenAndServe(":8888", nil); err != nil {
        log.Fatal("ListenAndServe:", err)
    }
}
