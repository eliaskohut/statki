package client

import (
	"bytes"
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

type PlayerStatus struct {
	GameStatus string `json:"game_status"`
	Nick       string `json:"nick"`
}

type PlayerList struct {
	PlayerList []PlayerStatus
}

type Game struct {
	Coords     []string `json:"coords"`
	Desc       string   `json:"desc"`
	Nick       string   `json:"nick"`
	TargetNick string   `json:"target_nick"`
	WPBot      bool     `json:"wpbot"`
}

type StatusResponse struct {
	GameStatus     string   `json:"game_status"`
	LastGameStatus string   `json:"last_game_status"`
	Nick           string   `json:"nick"`
	OppShots       []string `json:"opp_shots"`
	Opponent       string   `json:"opponent"`
	ShouldFire     bool     `json:"should_fire"`
	Timer          int      `json:"timer"`
}

type GameDesc struct {
	Desc     string `json:"desc"`
	Nick     string `json:"nick"`
	OppDesc  string `json:"opp_desc"`
	Opponent string `json:"opponent"`
}

type Shot struct {
	Coord string `json:"coord"`
}

type ShotResult struct {
	Result string `json:"result"`
}

func InitGame(game Game) (string, Game, []string, error) {
	url, err := url.JoinPath(httpApiUrlAddress, "/game")
	if err != nil {
		log.Fatal(err)
		return "", Game{}, nil, err
	}

	gameJSON, err := json.Marshal(game)
	if err != nil {
		log.Fatal(err)
		return "", Game{}, nil, err
	}

	resp, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(gameJSON))
	if err != nil {
		log.Fatal(err)
		return "", Game{}, nil, err
	}
	resp.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: httpClientTimeout,
	}
	response, err := client.Do(resp)
	if err != nil {
		log.Fatal(err)
		return "", Game{}, nil, err
	}
	defer response.Body.Close()

	token := response.Header.Get("X-Auth-Token")
	time.Sleep(2 * time.Second)

	resp, err = http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
		return "", Game{}, nil, err
	}
	resp.Header.Set("X-Auth-Token", token)
	response, err = client.Do(resp)
	if err != nil {
		log.Fatal(err)
		return "", Game{}, nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
		return "", Game{}, nil, err
	}

	err = json.Unmarshal(body, &game)
	if err != nil {
		log.Fatal(err)
		return "", Game{}, nil, err
	}
	layout, err := Board(token)
	if err != nil {
		log.Fatal(err)
		return "", Game{}, nil, err
	}

	game.Coords = layout
	time.Sleep(1 * time.Second)
	return token, game, layout, err

}
func Board(token string) ([]string, error) {
	type Board struct {
		Board []string `json:"board"`
	}

	url, err := url.JoinPath(httpApiUrlAddress, "/game/board")
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	req.Header.Set("X-Auth-Token", token)
	client := &http.Client{
		Timeout: httpClientTimeout,
	}

	response, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	var board Board
	err = json.Unmarshal(body, &board)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	time.Sleep(time.Millisecond * 300)
	return board.Board, err
}

func Status(token string) (*StatusResponse, error) {
	url, err := url.JoinPath(httpApiUrlAddress, "/game")
	var statusResponse StatusResponse
	if err != nil {
		log.Fatal(err)
		return &statusResponse, err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
		return &statusResponse, err
	}

	req.Header.Set("X-Auth-Token", token)
	client := &http.Client{
		Timeout: httpClientTimeout,
	}

	response, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		return &statusResponse, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
		return &statusResponse, err
	}
	err = json.Unmarshal(body, &statusResponse)
	if err != nil {
		log.Fatal(err)
		return &statusResponse, err
	}
	time.Sleep(time.Millisecond * 300)
	return &statusResponse, err
}
func Fire(token string, coord string) (string, error) {
	shot := Shot{Coord: coord}
	shotJson, err := json.Marshal(shot)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	payload := bytes.NewReader(shotJson)
	url, err := url.JoinPath(httpApiUrlAddress, "/game/fire")
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, url, payload)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	req.Header.Set("X-Auth-Token", token)
	client := &http.Client{
		Timeout: httpClientTimeout,
	}
	response, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	var shotRes ShotResult
	err = json.Unmarshal(body, &shotRes)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	time.Sleep(time.Millisecond * 300)
	return shotRes.Result, err
}
func Description(token string) (GameDesc, error) {
	url, err := url.JoinPath(httpApiUrlAddress, "/game/desc")
	if err != nil {
		log.Fatal(err)
		return GameDesc{}, err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
		return GameDesc{}, err
	}

	req.Header.Set("X-Auth-Token", token)
	client := &http.Client{
		Timeout: httpClientTimeout,
	}

	response, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		return GameDesc{}, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
		return GameDesc{}, err
	}
	var gameDesc GameDesc
	err = json.Unmarshal(body, &gameDesc)
	if err != nil {
		log.Fatal(err)
		return GameDesc{}, err
	}
	time.Sleep(time.Millisecond * 300)
	return gameDesc, err
}

func GetPlayers() (PlayerList, error) {
	url, err := url.JoinPath(httpApiUrlAddress, "/game/list")
	if err != nil {
		log.Fatal(err)
		return PlayerList{}, err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
		return PlayerList{}, err
	}
	client := &http.Client{
		Timeout: httpClientTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		return PlayerList{}, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
		return PlayerList{}, err
	}
	var arr []PlayerStatus
	err = json.Unmarshal(body, &arr)
	if err != nil {
		log.Fatal(err)
		return PlayerList{}, err
	}
	time.Sleep(time.Millisecond * 300)
	return PlayerList{PlayerList: arr}, err
}

func Abandon(token string) error {
	url, err := url.JoinPath(httpApiUrlAddress, "/game/abandon")
	if err != nil {
		log.Fatal(err)
		return err
	}

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		log.Fatal(err)
		return err
	}

	req.Header.Set("X-Auth-Token", token)
	client := &http.Client{
		Timeout: httpClientTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer resp.Body.Close()
	time.Sleep(time.Millisecond * 300)
	return err
}
