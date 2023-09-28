package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"github.com/EricKarschner37/Go-Jeopardy/connections"
	"github.com/rs/cors"
)

func main() {
	connections.Upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	games := map[int]connections.Game{}
	gameNum := 0

	mux := http.NewServeMux()

	mux.HandleFunc("/api/games", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		res := []GameListing{}
		for num := range games {
			created := games[num].Created
			res = append(res, GameListing{num, created.Unix()})
		}

		payload, err := json.Marshal(res)

		if err != nil {
			fmt.Fprintln(w, err)
			return
		}

		fmt.Fprintln(w, string(payload))

		return
	})

	mux.HandleFunc("/api/end", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
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
		game := games[num]
		game.EndGame()
		delete(games, num)
		fmt.Println("games:", games)
		fmt.Printf("Game %d ended\n", num)
	})

	mux.HandleFunc("/api/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
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

		mux.HandleFunc(fmt.Sprintf("/ws/%d/buzzer", gameNum), game.AcceptPlayer)
		mux.HandleFunc(fmt.Sprintf("/ws/%d/host", gameNum), game.AcceptHost)
		mux.HandleFunc(fmt.Sprintf("/ws/%d/board", gameNum), game.AcceptBoard)

		games[gameNum] = game
		fmt.Fprintf(w, "{\"gameNum\": %d}", gameNum)

		gameNum++
		return
	})

	handler := cors.Default().Handler(mux)
	fmt.Println("Listening on port 10001...")
	err := http.ListenAndServe("0.0.0.0:10001", handler)
	if err != nil {
		fmt.Println(err)
	}
}

type GameListing struct {
	Num     int   `json:"num"`
	Created int64 `json:"created"`
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
	if _, err := os.Stat(fmt.Sprintf("games/%d/final_responses.csv", num)); os.IsNotExist(err) {
		return false
	}
	return true
}
