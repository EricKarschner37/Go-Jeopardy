package connections

import "fmt"
import "net/http"

func (game *Game) AcceptBoard(w http.ResponseWriter, r *http.Request) {
  if (game.Board != nil) {
    return
  }

  fmt.Println("Board attempting to connect...")
  conn, err := Upgrader.Upgrade(w, r, nil)
  if (err != nil) {
    fmt.Println(err)
    return
  }

  resp := make(map[string]interface{})
  game.Board = conn
  game.SendCategories()

  for {
    err = conn.ReadJSON(&resp)
    if err != nil {
      fmt.Println(err)
      return
    }

    game.Mu.Lock()
    switch resp["request"] {
    case "reveal":
      row := int(resp["row"].(float64))
      col := int(resp["col"].(float64))
      fmt.Printf("Board asking to reveal: %d, %d\n", row, col)
      game.Reveal(row, col)
    case "start_double":
      game.StartDouble()
    case "start_final":
      fmt.Printf("Starting final... State: %s\n", game.state.Name)
      game.StartFinal()
    case "response":
      game.ShowResponse()
    case "board":
      game.ShowBoard()
    case "remove":
      game.RemovePlayer(resp["name"].(string))
    }
    game.Mu.Unlock()
  }
}
