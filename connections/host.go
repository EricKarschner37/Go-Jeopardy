package connections

import "fmt"
import "net/http"

func (game *Game) AcceptHost(w http.ResponseWriter, r *http.Request) {
  fmt.Println("Host initiating connection")
  game.Mu.Lock()
  if game.Host != nil {
    return
  }

  conn, err := Upgrader.Upgrade(w, r, nil)
  if err != nil {
	fmt.Println(err)
	return
  }

  game.Host = conn

  game.Mu.Unlock()

  resp := make(map[string]interface{})

  for {
    err = conn.ReadJSON(&resp)
    if err != nil {
	  game.Host = nil
      fmt.Println(err)
      return
    }

    game.Mu.Lock()

    switch (resp["request"]) {
    case "open":
      game.OpenBuzzers()
    case "close":
      game.CloseBuzzers()
    case "correct":
      game.ResponseCorrect(resp["correct"].(bool))
    case "player":
      game.ChoosePlayer(resp["player"].(string))
    }

    game.Mu.Unlock()
  }
}
