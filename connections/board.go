package connections

import "fmt"
import "net/http"

func AcceptBoard(w http.ResponseWriter, r *http.Request) {
  if (CurrentGame.Board != nil) {
    return
  }

  fmt.Println("Board attempting to connect...")
  conn, err := Upgrader.Upgrade(w, r, nil)
  if (err != nil) {
    fmt.Println(err)
    return
  }


  resp := make(map[string]interface{})
  CurrentGame.Board = conn

  if (CurrentGame.SingleJeopardy == nil) {
    err = conn.ReadJSON(&resp)
    if err != nil {
      fmt.Println(err)
      return
    }

    if resp["request"] == "start_game" {
      StartGame(int(resp["game_num"].(float64)))
    }
  }

  CurrentGame.SendCategories()

  for {
    err = conn.ReadJSON(&resp)
    if err != nil {
      fmt.Println(err)
      return
    }

    Mu.Lock()
    switch resp["request"] {
    case "reveal":
      row := int(resp["row"].(float64))
      col := int(resp["col"].(float64))
      fmt.Printf("Board asking to reveal: %d, %d\n", row, col)
      CurrentGame.Reveal(row, col)
    case "start_double":
      CurrentGame.StartDouble()
    case "response":
      CurrentGame.ShowResponse()
    case "board":
      CurrentGame.ShowBoard()
    }
    Mu.Unlock()
  }
}
