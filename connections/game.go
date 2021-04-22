package connections

import "sync"
import "os"
import "fmt"
import "encoding/csv"
import "encoding/json"
import "github.com/gorilla/websocket"
import "strings"

type JeopardyRound struct {
  Categories [6]string
  Clues [5][6]string
  Responses [5][6]string
}

type Game struct {
  Host *websocket.Conn
  Board *websocket.Conn
  SingleJeopardy *JeopardyRound
  DoubleJeopardy *JeopardyRound
  state map[string]interface{}
}

var Mu = &sync.Mutex{}

var CurrentGame Game

func (game *Game) sendState() {
  stateJson, _ := json.Marshal(game.state)
  for _, p := range game.state["players"].(map[string]*Player) {
	if (p.Conn == nil) {
	  continue
	}
    p.Conn.WriteMessage(websocket.TextMessage, []byte(stateJson))
  }

  if game.Host != nil {
    game.Host.WriteMessage(websocket.TextMessage, []byte(stateJson))
  }

  if game.Board != nil {
    game.Board.WriteMessage(websocket.TextMessage, []byte(stateJson))
  }
}

func (game *Game) setState(key string, value interface{}) {
  game.state[key] = value
  game.sendState()
}

func (game *Game) Wager(amount int, player string) {
  var max int
  bal := game.state["players"].(map[string]*Player)[player].Points
  if game.state["double"].(bool) {
    max = 2000
  } else {
	max = 1000
  }

  if bal > max {
	max = bal
  }

  if amount > max || amount < 5 {
    return
  }
  game.setState("cost", amount)
  game.setState("name", "clue")
  game.setState("buzzers_open", false)
}

func (game *Game) Buzz(player *Player) {
  if (game.state["buzzers_open"].(bool)) {
    game.setState("buzzers_open", false)
    game.setState("selected_player", player.Name)

    fmt.Println("Player buzzed:", player.Name)
  }
}

func (game *Game) AddPlayer(name string, conn *websocket.Conn) *Player {
  for n, p := range game.state["players"].(map[string]*Player) {
	fmt.Println(p.Name)
    if n == name && p.Conn == nil {
      p.Conn = conn
	  game.sendState()
	  return p
	}
  }

  p := Player{
	conn,
	name,
	0,
	game,
  }
  game.state["players"].(map[string]*Player)[name] = &p
  game.sendState()
  return &p
}

func (game *Game) Reveal(row int, col int) {
  fmt.Println("Revealing clue")
  var round *JeopardyRound
  if (game.state["double"].(bool)) {
    round = game.DoubleJeopardy
    game.setState("cost", (row + 1) * 400)
  } else {
    round = game.SingleJeopardy
    game.setState("cost", (row + 1) * 200)
  }

  if (strings.Contains(round.Clues[row][col], "Daily Double: ")) {
    game.setState("name", "daily_double")
  } else {
    game.setState("name", "clue")
  }

  game.setState("buzzers_open", false)
  game.setState("response", round.Responses[row][col])
  game.setState("clue", round.Clues[row][col])
}

func (game *Game) OpenBuzzers() {
  game.setState("buzzers_open", true)
}

func (game *Game) CloseBuzzers() {
  game.setState("buzzers_open", false)
}

func (game *Game) ResponseCorrect(correct bool) {
  if player, ok := game.state["players"].(map[string]*Player)[game.state["selected_player"].(string)]; ok {
    if (correct) {
      player.Points += game.state["cost"].(int)
      game.setState("name", "response")
    } else {
      player.Points -= game.state["cost"].(int)
      game.setState("buzzers_open", true)
    }

    game.setState("selected_player", "")
  }
}

func (game *Game) ChoosePlayer(name string) {
  game.setState("selected_player", name)
  game.setState("name", "daily_double")
}

func (game *Game) StartDouble() {
  game.setState("double", true)
  game.setState("name", "board")
  game.SendCategories()
}

func (game *Game) ShowResponse() {
  if (!game.state["buzzers_open"].(bool)) {
    game.setState("name", "response")
  }
}

func (game *Game) ShowBoard() {
  game.setState("name", "board")
}

func (game *Game) SendCategories() {
  categoriesMap := map[string]interface{} {
    "message": "categories",
  }

  if game.state["double"].(bool) {
    categoriesMap["categories"] = game.DoubleJeopardy.Categories
  } else {
    categoriesMap["categories"] = game.SingleJeopardy.Categories
  }

  categoriesMsg, _ := json.Marshal(categoriesMap)

  game.Board.WriteMessage(websocket.TextMessage, []byte(categoriesMsg))
}

func readCSV(filename string) {
  file, err := os.Open(filename)
  if err != nil {
    fmt.Println(err)
  }

  r := csv.NewReader(file)

  var single JeopardyRound
  record, err := r.Read()
  copy(single.Categories[:], record)
  for i := 0; i < 5; i++ {
    record, err = r.Read()
    copy(single.Clues[i][:], record)
  }
  for i := 0; i < 5; i++ {
    record, err = r.Read()
    copy(single.Responses[i][:], record)
  }

  var double JeopardyRound
  record, err = r.Read()
  copy(double.Categories[:], record)
  for i := 0; i < 5; i++ {
    record, err = r.Read()
    copy(double.Clues[i][:], record)
  }
  for i := 0; i < 5; i++ {
    record, err = r.Read()
    copy(double.Responses[i][:], record)
  }
  CurrentGame.SingleJeopardy = &single
  CurrentGame.DoubleJeopardy = &double
}

// Valid state names:
//  - clue
//  - response
//  - daily_double
//  - board

func StartGame(num int) {
  file := fmt.Sprintf("games/%d.csv", num)
  readCSV(file)
  CurrentGame.state = map[string]interface{}{
    "message": "state",
    "buzzers_open": false,
    "selected_player": "",
    "cost": 0,
    "clue": "",
    "response": "",
    "players": make(map[string]*Player),
    "name": "board",
    "double": false,
  }
}
