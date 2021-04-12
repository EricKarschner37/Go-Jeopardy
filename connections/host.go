package connections

import "fmt"
import "net/http"

func AcceptHost(w http.ResponseWriter, r *http.Request) {
  fmt.Println("Host initiating connection")
  Mu.Lock()
  if CurrentGame.Host != nil {
    return
  }

  conn, err := Upgrader.Upgrade(w, r, nil)
  if err != nil {
    CurrentGame.Host = conn
  }

  Mu.Unlock()

  resp := make(map[string]interface{})

  for {
    err = conn.ReadJSON(&resp)
    if err != nil {
      fmt.Println(err)
      return
    }

    Mu.Lock()

    switch (resp["request"]) {
    case "open":
      CurrentGame.OpenBuzzers()
    case "close":
      CurrentGame.CloseBuzzers()
    case "correct":
      CurrentGame.ResponseCorrect(resp["correct"].(bool))
    case "player":
      CurrentGame.ChoosePlayer(resp["player"].(string))
    }

    Mu.Unlock()
  }
}
