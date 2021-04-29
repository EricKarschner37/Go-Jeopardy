package main

import "net/http"
import "encoding/json"
import "os/exec"
import "os"
import "io"
import "fmt"
import "strconv"
import "github.com/EricKarschner37/Go-Jeopardy/connections"

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

    num := int(req["num"].(float64))
    if !gameExists(num) {
	  fmt.Printf("Game %d does not exist, fetching...\n", num)
	  if e := exec.Command("./get_game.py", strconv.Itoa(num)).Run(); e != nil {
	    fmt.Println(e)
	  }
    }

	  var game connections.Game
	  game.StartGame(num)

	  http.HandleFunc("/ws/buzzer", game.AcceptPlayer)
	  http.HandleFunc("/ws/host", game.AcceptHost)
	  http.HandleFunc("/ws/board", game.AcceptBoard)
  })

  err := http.ListenAndServe("0.0.0.0:10001", nil)
  if err != nil {
    fmt.Println(err)
  }
}

func gameExists(num int) bool {
  if _, err := os.Stat(fmt.Sprintf("games/%d", num)); os.IsNotExist(err) {
    return false
  }
  if _, err := os.Stat(fmt.Sprintf("games/%d/single_clues.csv", num)); os.IsNotExist(err) {
    return false
  }
  if _, err := os.Stat(fmt.Sprintf("games/%d/single_responses.csv", num)); os.IsNotExist(err) {
    return false
  }
  if _, err := os.Stat(fmt.Sprintf("games/%d/double_clues.csv", num)); os.IsNotExist(err) {
    return false
  }
  if _, err := os.Stat(fmt.Sprintf("games/%d/double_responses.csv", num)); os.IsNotExist(err) {
    return false
  }
  return true
}
