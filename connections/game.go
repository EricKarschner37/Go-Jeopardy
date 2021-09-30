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
  Mu *sync.Mutex
  OnEnd func()
  state State
}


func (game *Game) sendState() {
  stateJson, err := json.Marshal(&(game.state))
  if err != nil {
	fmt.Println(err)
  }
  fmt.Println("Sending: ", string(stateJson))
  fmt.Println(game.state.Clue)
  for _, p := range game.state.Players {
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

func (game *Game) Wager(amount int, player string) {
  var max int
  bal := game.state.Players[player].Points
  if game.state.Double {
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
  game.state.Cost = amount
  game.state.Name = "clue"
  game.state.Buzzers_open = false
  game.sendState()

}

func (game *Game) Buzz(player *Player) {
  if (game.state.Buzzers_open) {
    game.state.Buzzers_open = false
    game.state.Selected_player = player.Name

    fmt.Println("Player buzzed:", player.Name)
	game.sendState()
  }
}

func (game *Game) AddPlayer(name string, conn *websocket.Conn) *Player {
  for n, p := range game.state.Players {
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
  game.state.Players[name] = &p
  game.sendState()
  return &p
}

func (game *Game) Reveal(row int, col int) {
  fmt.Println("Revealing clue")
  var round *JeopardyRound
  if (game.state.Double) {
    round = game.DoubleJeopardy
    game.state.Cost = (row + 1) * 400
  } else {
    round = game.SingleJeopardy
    game.state.Cost = (row + 1) * 200
  }

  if (strings.Contains(round.Clues[row][col], "Daily Double: ")) {
    game.state.Name = "daily_double"
  } else {
    game.state.Name = "clue"
  }

  game.state.Buzzers_open = false
  game.state.Response = round.Responses[row][col]
  game.state.Clue = round.Clues[row][col]
  game.sendState()
}

func (game *Game) OpenBuzzers() {
  game.state.Buzzers_open = true
  game.sendState()
}

func (game *Game) CloseBuzzers() {
  game.state.Buzzers_open = false
  game.sendState()
}

func (game *Game) ResponseCorrect(correct bool) {
  if player, ok := game.state.Players[game.state.Selected_player]; ok {
    if (correct) {
      player.Points += game.state.Cost
      game.state.Name = "response"
    } else {
      player.Points -= game.state.Cost
      game.state.Buzzers_open = true
    }

    game.state.Selected_player = ""
  }
  game.sendState()
}

func (game *Game) ChoosePlayer(name string) {
  game.state.Selected_player = name
  game.state.Name = "daily_double"
  game.sendState()
}

func (game *Game) StartDouble() {
  game.state.Double = true
  game.state.Name = "board"
  game.SendCategories()
  game.sendState()
}

func (game *Game) ShowResponse() {
  if (!game.state.Buzzers_open) {
    game.state.Name = "response"
  }
  game.sendState()
}

func (game *Game) ShowBoard() {
  game.state.Name = "board"
  game.sendState()
}

func (game *Game) SendCategories() {
  categoriesMap := map[string]interface{} {
    "message": "categories",
  }

  if game.state.Double {
    categoriesMap["categories"] = game.DoubleJeopardy.Categories
  } else {
    categoriesMap["categories"] = game.SingleJeopardy.Categories
  }

  categoriesMsg, _ := json.Marshal(categoriesMap)

  game.Board.WriteMessage(websocket.TextMessage, []byte(categoriesMsg))
}

func readRound(cluesFile string, responsesFile string, round *JeopardyRound) {
  file, err := os.Open(cluesFile)
  if err != nil {
    fmt.Println(err)
  }

  r := csv.NewReader(file)


  record, err := r.Read()
  copy(round.Categories[:], record)

  for i := 0; i < 5; i++ {
    record, err = r.Read()
    copy(round.Clues[i][:], record)
  }


  file, err = os.Open(responsesFile)
  if err != nil {
	fmt.Println(err)
  }

  r = csv.NewReader(file)
  r.Read()

  for i := 0; i < 5; i++ {
    record, err = r.Read()
    copy(round.Responses[i][:], record)
  }
}

// Valid state names:
//  - clue
//  - response
//  - daily_double
//  - board

func (game *Game) StartGame(num int) {
  dir := fmt.Sprintf("games/%d", num)

  game.SingleJeopardy = &JeopardyRound{}
  game.DoubleJeopardy = &JeopardyRound{}
  game.Mu = &sync.Mutex{}

  readRound(fmt.Sprintf("%s/single_clues.csv", dir), fmt.Sprintf("%s/single_responses.csv", dir), game.SingleJeopardy)

  readRound(fmt.Sprintf("%s/double_clues.csv", dir), fmt.Sprintf("%s/double_responses.csv", dir), game.DoubleJeopardy)

  game.state = State {
	"state",					//message
    false,						//buzzers_open
    "",							//selected_player
    0,							//cost
    "",							//clue
    "",							//response
    make(map[string]*Player),	//players
    "board",					//name
    false,						//double
  }
}

func (game *Game) endGame() {
  if (game.Board != nil) {
    game.Board.Close()
  }
  if (game.Host != nil) {
	game.Host.Close()
  }
  for _, p := range game.state.Players {
	if (p.Conn != nil) {
      p.Conn.Close()
	}
  }
  game.OnEnd()
}

type State struct {
  Message string				`json:"message"`
  Buzzers_open bool				`json:"buzzers_open"`
  Selected_player string		`json:"selected_player"`
  Cost int						`json:"cost"`
  Clue string					`json:"clue"`
  Response string				`json:"response"`
  Players map[string]*Player	`json:"players"`
  Name string					`json:"name"`
  Double bool					`json:"double"`
}
