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

type FinalRound struct {
  Category string
  Clue string
  Response string
  Wagers map[string]int
  PlayerResponses map[string]string
}

type Game struct {
  Host *websocket.Conn
  Board *websocket.Conn
  SingleJeopardy *JeopardyRound
  DoubleJeopardy *JeopardyRound
  FinalJeopardy *FinalRound
  Mu *sync.Mutex
  state State
  localState LocalState
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

func (game *Game) FinalResponse(player string, response string) {
  if response == "" || !game.state.Final {
    return
  }
  game.FinalJeopardy.PlayerResponses[player] = response
  all_responses := true 
  for _, r := range game.FinalJeopardy.PlayerResponses {
    if r == "" {
      all_responses = false
      break
    }
  }
  if all_responses {
    game.evaluateFinalResponses()
  }
}

func (game *Game) evaluateFinalResponses() {
  player := ""
  for p, r := range game.FinalJeopardy.PlayerResponses {
    if r != "" {
      player = p
      break
    }
  }

  game.state.Selected_player = player

  if player == "" {
    game.state.Name = "complete"
  } else {
    game.state.Name = "clue"
    game.state.Response = fmt.Sprintf("%s's response: %s\nCorrect response: %s", player, game.FinalJeopardy.PlayerResponses[player], game.FinalJeopardy.Response)
    game.state.Cost = game.FinalJeopardy.Wagers[player]
    game.state.Buzzers_open = true
    game.Buzz(game.state.Players[player])
  }
}

func (game *Game) Wager(amount int, player string) {
  var max int
  bal := game.state.Players[player].Points
  if game.state.Double {
    max = 2000
  } else if game.state.Final {
    max = 3000
  } else {
	  max = 1000
  }

  if bal > max {
	  max = bal
  }

  if amount > max || amount < 5 {
    return
  }

  if game.state.Final {
    game.FinalJeopardy.Wagers[player] = amount
    all_wagered := true
    for _, b := range game.FinalJeopardy.Wagers {
      if b == 0 {
        all_wagered = false
        break
      }
    }
    if all_wagered {
      game.state.Name = "final_clue"
      game.state.Clue = game.FinalJeopardy.Clue
      game.state.Response = game.FinalJeopardy.Response
      game.sendState()
    }
    return
  }

  game.state.Cost = amount
  game.state.Name = "clue"
  game.state.Buzzers_open = false
  game.sendState()
}

func (game *Game) Buzz(player *Player) {
  if (game.state.Buzzers_open && !game.localState.hasPlayerBuzzed[player.Name]) {
    game.state.Buzzers_open = false
    game.state.Selected_player = player.Name
    game.localState.hasPlayerBuzzed[player.Name] = true

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
  game.localState.hasPlayerBuzzed[name] = false;
  game.sendState()
  return &p
}

func (game *Game) Reveal(row int, col int) {
  fmt.Println("Revealing clue")
  bitsetKey := 1 << (row*6 + col)
  if (bitsetKey & game.state.HasClueBeenShownBitset != 0) {
    // Clue has already been shown
    return
  }
  game.state.HasClueBeenShownBitset = game.state.HasClueBeenShownBitset | bitsetKey
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
  if (game.state.Double) {
    game.state.Category = game.DoubleJeopardy.Categories[col]
  } else {
    game.state.Category = game.SingleJeopardy.Categories[col]
  }
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
    if game.state.Final {
      game.FinalJeopardy.PlayerResponses[player.Name] = "";
      game.FinalJeopardy.Wagers[player.Name] = 0;
    }
    if (correct) {
      player.Points += game.state.Cost
      if game.state.Final {
        game.evaluateFinalResponses()
	      return
      } else {
	      game.ShowResponse()
      }
    } else {
      player.Points -= game.state.Cost
      if game.state.Final {
        game.evaluateFinalResponses()
	      return
      } else {
        game.state.Buzzers_open = true
      }
    }
  } else {
    game.state.Buzzers_open = true
  }
  game.state.Selected_player = ""
  game.sendState()
}

func (game *Game) ChoosePlayer(name string) {
  game.state.Selected_player = name
  game.state.Name = "daily_double"
  for n, _ := range game.state.Players {
    game.localState.hasPlayerBuzzed[n] = true
  }
  game.sendState()
}

func (game *Game) StartDouble() {
  game.state.Double = true
  game.state.Name = "board"
  game.state.HasClueBeenShownBitset = 0;
  game.SendCategories()
  game.sendState()
}

func (game *Game) StartFinal() {
  game.state.Double = false
  game.state.Final = true
  game.state.Name = "final"
  game.state.Category = game.FinalJeopardy.Category
  for n, _ := range game.state.Players {
    //TODO - do this on player added/removed
    game.FinalJeopardy.PlayerResponses[n] = ""
    game.FinalJeopardy.Wagers[n] = 0
  }
  game.sendState()
}

func (game *Game) ShowResponse() {
  if (!game.state.Buzzers_open) {
    game.state.Name = "response"
    for n, _ := range game.state.Players {
      game.localState.hasPlayerBuzzed[n] = false;
    }
    game.sendState()
  }
}

func (game *Game) ShowBoard() {
  game.state.Name = "board"
  game.sendState()
}

func (game *Game) RemovePlayer(name string) {
  delete(game.state.Players, name)
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

func (game *Game) SetPlayerBalance(name string, balance int) {
  if player, ok := game.state.Players[name]; ok {
    player.Points = balance;
  }
  game.sendState()
}

func readFinal(finalFile string, final *FinalRound) {
  file, err := os.Open(finalFile)
  if err != nil {
    fmt.Println(err)
  }

  r := csv.NewReader(file)
  record, err := r.Read()
  if err != nil {
    fmt.Println(err)
  }
  final.Category = record[0]

  record, err = r.Read()
  if err != nil {
    fmt.Println(err)
  }
  final.Clue = record[0]

  record, err = r.Read()
  if err != nil {
    fmt.Println(err)
  }
  final.Response= record[0]
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
  game.FinalJeopardy = &FinalRound{}
  game.FinalJeopardy.Wagers = make(map[string]int)
  game.FinalJeopardy.PlayerResponses = make(map[string]string)
  game.localState = LocalState {
    make(map[string]bool),
  }
  game.Mu = &sync.Mutex{}

  readRound(fmt.Sprintf("%s/single_clues.csv", dir), fmt.Sprintf("%s/single_responses.csv", dir), game.SingleJeopardy)

  readRound(fmt.Sprintf("%s/double_clues.csv", dir), fmt.Sprintf("%s/double_responses.csv", dir), game.DoubleJeopardy)

  readFinal(fmt.Sprintf("%s/final.csv", dir), game.FinalJeopardy)

  game.state = State {
	  "state",					//message
    false,						//buzzers_open
    "",							  //selected_player
    0,							  //cost
    "",              //category
    "",							  //clue
    "",							  //response
    make(map[string]*Player),	//players
    "board",					//name
    false,						//double
    false,            //final
    0,						    //HasClueBeenShownBitset
  } 
}

func (game *Game) EndGame() {
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
}

// The state that gets sent to clients
type State struct {
  Message string				`json:"message"`
  Buzzers_open bool				`json:"buzzers_open"`
  Selected_player string		`json:"selected_player"`
  Cost int						`json:"cost"`
  Category string     `json:"category"`
  Clue string					`json:"clue"`
  Response string				`json:"response"`
  Players map[string]*Player	`json:"players"`
  Name string					`json:"name"`
  Double bool					`json:"double"`
  Final bool          `json:"final"`
  // If the clue at row i, col j has been shown,
  // then the bit at 2^(i*6+j) is 1
  HasClueBeenShownBitset int `json:"hasClueBeenShownBitset"`
}

// State which is used only locally and not sent to clients
type LocalState struct {
  hasPlayerBuzzed map[string]bool
}
