package connections

import "fmt"
import "github.com/gorilla/websocket"
import "net/http"

var Upgrader = websocket.Upgrader {
  ReadBufferSize: 1024,
  WriteBufferSize: 1024,
}

type Player struct {
  Conn *websocket.Conn
  Name string
  Points int
  game *Game
}

func (player *Player) Buzz() {
  player.game.Buzz(player)
}

func (player *Player) Wager(amount int) {
  player.game.Wager(amount, player.Name)
}

func AcceptPlayer(w http.ResponseWriter, r *http.Request) {
  fmt.Println("Client initiating connection...")
  conn, err := Upgrader.Upgrade(w, r, nil)
  if (err != nil) {
    fmt.Println(err)
    return
  }

  resp := make(map[string]interface{})

  err = conn.ReadJSON(&resp)
  if err != nil {
    fmt.Println(err)
    return
  }

  var p *Player

  if resp["request"] == "register" {
    Mu.Lock()
    p = CurrentGame.AddPlayer(resp["name"].(string), conn)
    Mu.Unlock()
    fmt.Printf("Player %s registered\n", p.Name)
  }

  for {
    err = conn.ReadJSON(&resp)
    if err != nil {
      fmt.Println(err)
	  p.Conn = nil
      return
    }

    Mu.Lock()
    switch resp["request"] {
    case "buzz":
      p.Buzz()
    case "wager":
      p.Wager(int(resp["amount"].(float64)))
    }
    Mu.Unlock()
  }

}
