package main

import "fmt"
import "net/http"
import "github.com/EricKarschner37/Go-Jeopardy/connections"
//import "github.com/gorilla/websocket"
import "sync"

var mu = &sync.Mutex{}

func main() {
  connections.Upgrader.CheckOrigin = func(r *http.Request) bool {
    return true
  }

  http.HandleFunc("/buzzer", connections.AcceptPlayer)
  http.HandleFunc("/buzzer/host", connections.AcceptHost)
  connections.StartGame()
  err := http.ListenAndServe("0.0.0.0:8080", nil)
  if err != nil {
    fmt.Println(err)
  }
}
