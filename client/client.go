package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	httpAPIURLAddress = "https://go-pjatk-server.fly.dev/api"
)

var (
	contentType      = "application/json"
	initGameDelay    = 1 * time.Second
	boardDelay       = time.Millisecond * 300
	statusDelay      = time.Millisecond * 300
	fireDelay        = time.Millisecond * 300
	descriptionDelay = time.Millisecond * 300
	playersDelay     = time.Millisecond * 300
	abandonDelay     = time.Millisecond * 300
)

type Client struct {
	client  *http.Client
	baseURL string
	token   string
}

func NewClient() *Client {
	return &Client{
		client:  &http.Client{},
		baseURL: httpAPIURLAddress,
	}
}

func (c *Client) InitGame(game Game) (Game, error) {
	urlPath := c.buildURL("/game")
	gameJSON, err := json.Marshal(game)
	if err != nil {
		return Game{}, fmt.Errorf("InitGame: json.Marshal: %v", err)
	}
	req, err := c.newRequest(http.MethodPost, urlPath, bytes.NewReader(gameJSON))
	if err != nil {
		return Game{}, fmt.Errorf("InitGame: sendRequest: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return Game{}, fmt.Errorf("InitGame: client.Do(req): %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Game{}, fmt.Errorf("InitGame: unexpected response status: %s", resp.Status)
	}
	c.token = resp.Header.Get("X-Auth-Token")
	time.Sleep(initGameDelay)
	return game, nil
}

func (c *Client) GetBoard() (Board, error) {
	if c.token != "" {
		urlPath := c.buildURL("/game/board")
		req, err := c.newRequest(http.MethodGet, urlPath, nil)
		if err != nil {
			return Board{}, fmt.Errorf("GetBoard: sendRequest: %w", err)
		}
		resp, err := c.client.Do(req)
		if err != nil {
			return Board{}, fmt.Errorf("GetBoard: client.Do(req): %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return Board{}, fmt.Errorf("GetBoard: unexpected response status: %s", resp.Status)
		}
		var board Board
		err = json.NewDecoder(resp.Body).Decode(&board)
		if err != nil {
			return Board{}, fmt.Errorf("GetBoard: error decoding response body: %w", err)
		}
		time.Sleep(boardDelay)
		return board, nil
	} else {
		return Board{}, fmt.Errorf("GetBoard: no token")
	}
}

func (c *Client) GetStatus() (StatusResponse, error) {
	if c.token != "" {
		urlPath := c.buildURL("/game")
		req, err := c.newRequest(http.MethodGet, urlPath, nil)
		if err != nil {
			return StatusResponse{}, fmt.Errorf("GetStatus: sendRequest: %w", err)
		}
		resp, err := c.client.Do(req)
		if err != nil {
			return StatusResponse{}, fmt.Errorf("GetStatus: client.Do(req): %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return StatusResponse{}, fmt.Errorf("GetStatus: unexpected response status: %s", resp.Status)
		}
		var status StatusResponse
		err = json.NewDecoder(resp.Body).Decode(&status)
		if err != nil {
			return StatusResponse{}, fmt.Errorf("GetStatus: error decoding response body: %w", err)
		}
		time.Sleep(statusDelay)
		return status, nil
	} else {
		return StatusResponse{}, fmt.Errorf("GetStatus: no token")
	}
}

func (c *Client) Shoot(coord string) (string, error) {
	if c.token != "" {
		urlPath := c.buildURL("/game/fire")
		shot := Shot{Coord: coord}
		shotJSON, err := json.Marshal(shot)
		if err != nil {
			return "", fmt.Errorf("Shoot: json.Marshal: %w", err)
		}
		req, err := c.newRequest(http.MethodPost, urlPath, bytes.NewReader(shotJSON))
		if err != nil {
			return "", fmt.Errorf("Shoot: sendRequest: %w", err)
		}
		resp, err := c.client.Do(req)
		if err != nil {
			return "", fmt.Errorf("Shoot: client.Do(req): %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("Shoot: unexpected response status: %s", resp.Status)
		}
		var result ShotResult
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return "", fmt.Errorf("Shoot: error decoding response body: %w", err)
		}
		time.Sleep(fireDelay)
		return result.Result, nil
	} else {
		return "", fmt.Errorf("Shoot: no token")
	}
}

func (c *Client) GetDescription() (GameDesc, error) {
	if c.token != "" {
		urlPath := c.buildURL("/game/desc")
		req, err := c.newRequest(http.MethodGet, urlPath, nil)
		if err != nil {
			return GameDesc{}, fmt.Errorf("GetDescription: sendRequest: %w", err)
		}
		resp, err := c.client.Do(req)
		if err != nil {
			return GameDesc{}, fmt.Errorf("GetDescription: client.Do(req): %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return GameDesc{}, fmt.Errorf("GetDescription: unexpected response status: %s", resp.Status)
		}
		var desc GameDesc
		err = json.NewDecoder(resp.Body).Decode(&desc)
		if err != nil {
			return GameDesc{}, fmt.Errorf("GetDescription: error decoding response body: %w", err)
		}
		time.Sleep(descriptionDelay)
		return desc, nil
	} else {
		return GameDesc{}, fmt.Errorf("GetDescription: no token")
	}
}

func (c *Client) GetPlayers() (PlayersStatus, error) {
	urlPath := c.buildURL("/game/lobby")
	req, err := c.newRequest(http.MethodPost, urlPath, nil)
	if err != nil {
		return PlayersStatus{}, fmt.Errorf("GetPlayers: sendRequest: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return PlayersStatus{}, fmt.Errorf("GetPlayers: client.Do(req): %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return PlayersStatus{}, fmt.Errorf("GetPlayers: unexpected response status: %s", resp.Status)
	}
	var players PlayersStatus
	err = json.NewDecoder(resp.Body).Decode(&players)
	if err != nil {
		return PlayersStatus{}, fmt.Errorf("GetPlayers: error decoding response body: %w", err)
	}
	time.Sleep(playersDelay)
	return players, nil
}

func (c *Client) Abandon() error {
	urlPath := c.buildURL("/game/abandon")
	req, err := c.newRequest(http.MethodDelete, urlPath, nil)
	if err != nil {
		return fmt.Errorf("Abandon: sendRequest: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("Abandon: client.Do(req): %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Abandon: unexpected response status: %s", resp.Status)
	}
	time.Sleep(abandonDelay)
	return nil
}

func (c *Client) buildURL(endpoint string) string {
	baseURL, _ := url.Parse(c.baseURL)
	return baseURL.JoinPath(endpoint).String()
}

func (c *Client) newRequest(method, url string, body *bytes.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(context.Background(), method, url, body)
	if err != nil {
		return nil, fmt.Errorf("newRequest: http.NewRequestWithContext: %w", err)
	}
	if c.token != "" {
		req.Header.Set("X-Auth-Token", c.token)
	}
	req.Header.Set("Content-Type", contentType)
	return req, nil
}
