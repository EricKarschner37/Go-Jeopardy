package connections

import "fmt"
import "net/http"

func AcceptBoard(w http.ResponseWriter, r *http.Request) {
  fmt.Println("Board attempting to connect...")
  conn, err := Upgrader.Upgrade(w, r, nil)
  if (err != nil) {
    fmt.Println(err)
    return
  }

  if (CurrentGame == nil) {
    return
  }

  resp := make(map[string]interface{})

  for {
    err = conn.ReadJSON(&resp)
    if err != nil {
      fmt.Println(err)
      return
    }

    Mu.Lock()
    switch resp["request"] {
    case "reveal":
      CurrentGame.Reveal(resp["row"].(int), resp["col"].(int))
    case "start_double":
      CurrentGame.StartDouble()
    case "response":
      CurrentGame.ShowResponse()
    }
    Mu.Unlock()
  }
}
