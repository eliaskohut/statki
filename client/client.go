package client

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	httpApiUrlAddress = "https://go-pjatk-server.fly.dev/api"
	httpClientTimeout = time.Duration(30) * time.Second
)

type Game struct {
	Coords     []string `json:"coords"`
	Desc       string   `json:"desc"`
	Nick       string   `json:"nick"`
	TargetNick string   `json:"target_nick"`
	WPBot      bool     `json:"wpbot"`
}

type StatusResponse struct {
	Desc           string   `json:"desc"`
	GameStatus     string   `json:"game_status"`
	LastGameStatus string   `json:"last_game_status"`
	Nick           string   `json:"nick"`
	OppDesc        string   `json:"opp_desc"`
	OppShots       []string `json:"opp_shots"`
	Opponent       string   `json:"opponent"`
	ShouldFire     bool     `json:"should_fire"`
	Timer          int      `json:"timer"`
}

func InitGame() (string, Game, []string, *StatusResponse, error) {
	url, err := url.JoinPath(httpApiUrlAddress, "/game")
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		log.Fatal(err)
	}
	client := &http.Client{
		Timeout: httpClientTimeout,
	}
	response, err := client.Do(resp)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	token := response.Header.Get("X-Auth-Token")
	resp, err = http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}
	resp.Header.Set("X-Auth-Token", token)
	response, err = client.Do(resp)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	game := Game{}
	err = json.Unmarshal(body, &game)
	if err != nil {
		log.Fatal(err)
	}
	game.WPBot = true

	layout, err := Board(token)
	if err != nil {
		log.Fatal(err)
	}

	game.Coords = layout

	status, err := Status(token)
	if err != nil {
		log.Fatal(err)
	}

	game.Desc = status.Desc

	return token, game, layout, status, err
}
func Board(token string) ([]string, error) {
	type Board struct {
		Board []string `json:"board"`
	}

	url, err := url.JoinPath(httpApiUrlAddress, "/game/board")
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("X-Auth-Token", token)
	client := &http.Client{
		Timeout: httpClientTimeout,
	}

	response, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var board Board
	err = json.Unmarshal(body, &board)
	if err != nil {
		log.Fatal(err)
	}
	return board.Board, err
}

func Status(token string) (*StatusResponse, error) {
	url, err := url.JoinPath(httpApiUrlAddress, "/game")
	if err != nil {
		log.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("X-Auth-Token", token)
	client := &http.Client{
		Timeout: httpClientTimeout,
	}

	response, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	var statusResponse StatusResponse
	err = json.Unmarshal(body, &statusResponse)
	if err != nil {
		log.Fatal(err)
	}
	return &statusResponse, err
}

// func Fire(coord string) (string, error) {

// }