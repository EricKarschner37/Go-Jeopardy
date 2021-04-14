package main

import "fmt"
import "net/http"
import "github.com/EricKarschner37/Go-Jeopardy/connections"
import "sync"

var mu = &sync.Mutex{}

func main() {
  connections.Upgrader.CheckOrigin = func(r *http.Request) bool {
    return true
  }

  http.HandleFunc("/ws/buzzer", connections.AcceptPlayer)
  http.HandleFunc("/ws/host", connections.AcceptHost)
  http.HandleFunc("/ws/board", connections.AcceptBoard)
  err := http.ListenAndServe("0.0.0.0:10001", nil)
  if err != nil {
    fmt.Println(err)
  }
}
