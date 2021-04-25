package main

import "fmt"
import "net/http"
import "github.com/EricKarschner37/Go-Jeopardy/connections"
import "sync"
import "io"
import "encoding/json"

var mu = &sync.Mutex{}

func main() {
  connections.Upgrader.CheckOrigin = func(r *http.Request) bool {
    return true
  }

  http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
	if (r.Method != http.MethodPost) {
	  http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	  return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
	  fmt.Println(err)
	}
	var req map[string]interface{}
	err = json.Unmarshal(body, &req)
	if err != nil {
	  fmt.Println(err)
	}
	var game connections.Game
	game.StartGame(int(req["num"].(float64)))

	http.HandleFunc("/ws/buzzer", game.AcceptPlayer)
	http.HandleFunc("/ws/host", game.AcceptHost)
	http.HandleFunc("/ws/board", game.AcceptBoard)
  })

  err := http.ListenAndServe("0.0.0.0:10001", nil)
  if err != nil {
    fmt.Println(err)
  }
}
