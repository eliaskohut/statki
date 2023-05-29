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

const (
	clientTimeout    = time.Second * 30
	contentType      = "application/json"
	initGameDelay    = 1 * time.Second
	boardDelay       = time.Millisecond * 300
	statusDelay      = time.Millisecond * 300
	fireDelay        = time.Millisecond * 300
	descriptionDelay = time.Millisecond * 300
	playersDelay     = time.Millisecond * 300
	abandonDelay     = time.Millisecond * 300
	requestDelay     = time.Millisecond * 500
	maxRequests      = 10
)

type Client struct {
	client  *http.Client
	baseURL string
	Token   string
}

func NewClient() *Client {
	return &Client{
		client: &http.Client{
			Timeout: clientTimeout,
		},
		baseURL: httpAPIURLAddress,
	}
}

func (c *Client) InitGame(game Game) (Game, error) {
	requestFunc := func() (interface{}, error) {
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
		c.Token = resp.Header.Get("X-Auth-Token")
		time.Sleep(initGameDelay)
		return game, nil
	}

	resp, err := c.doRequest(requestFunc)
	if err != nil {
		return Game{}, fmt.Errorf("InitGame: %w", err)
	}

	result, ok := resp.(Game)
	if !ok {
		return Game{}, fmt.Errorf("InitGame: unexpected response type")
	}
	return result, nil
}

func (c *Client) GetBoard() (Board, error) {
	requestFunc := func() (interface{}, error) {
		if c.Token == "" {
			return Board{}, fmt.Errorf("GetBoard: no token")
		}
		urlPath := c.buildURL("/game/board")
		reqBody := bytes.NewReader([]byte{})
		req, err := c.newRequest(http.MethodGet, urlPath, reqBody)
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
	}

	resp, err := c.doRequest(requestFunc)
	if err != nil {
		return Board{}, fmt.Errorf("GetBoard: %w", err)
	}

	result, ok := resp.(Board)
	if !ok {
		return Board{}, fmt.Errorf("GetBoard: unexpected response type")
	}
	return result, nil
}

func (c *Client) GetStatus() (StatusResponse, error) {
	requestFunc := func() (interface{}, error) {
		if c.Token == "" {
			return StatusResponse{}, fmt.Errorf("GetStatus: no token")
		}
		urlPath := c.buildURL("/game")
		reqBody := bytes.NewReader([]byte{})
		req, err := c.newRequest(http.MethodGet, urlPath, reqBody)
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
	}

	resp, err := c.doRequest(requestFunc)
	if err != nil {
		return StatusResponse{}, fmt.Errorf("GetStatus: %w", err)
	}

	result, ok := resp.(StatusResponse)
	if !ok {
		return StatusResponse{}, fmt.Errorf("GetStatus: unexpected response type")
	}
	return result, nil
}

func (c *Client) Shoot(coord string) (string, error) {
	requestFunc := func() (interface{}, error) {
		if c.Token == "" {
			return "", fmt.Errorf("Shoot: no token")
		}
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
	}

	resp, err := c.doRequest(requestFunc)
	if err != nil {
		return "", fmt.Errorf("Shoot: %w", err)
	}

	result, ok := resp.(string)
	if !ok {
		return "", fmt.Errorf("Shoot: unexpected response type")
	}
	return result, nil
}

func (c *Client) GetDescription() (GameDesc, error) {
	requestFunc := func() (interface{}, error) {
		if c.Token == "" {
			return GameDesc{}, fmt.Errorf("GetDescription: no token")
		}
		urlPath := c.buildURL("/game/desc")
		reqBody := bytes.NewReader([]byte{})
		req, err := c.newRequest(http.MethodGet, urlPath, reqBody)
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
	}

	resp, err := c.doRequest(requestFunc)
	if err != nil {
		return GameDesc{}, fmt.Errorf("GetDescription: %w", err)
	}

	result, ok := resp.(GameDesc)
	if !ok {
		return GameDesc{}, fmt.Errorf("GetDescription: unexpected response type")
	}
	return result, nil
}

func (c *Client) GetPlayers() (PlayersStatus, error) {
	requestFunc := func() (interface{}, error) {
		urlPath := c.buildURL("/lobby")
		reqBody := bytes.NewReader([]byte{})
		req, err := c.newRequest(http.MethodGet, urlPath, reqBody)
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

	resp, err := c.doRequest(requestFunc)
	if err != nil {
		return PlayersStatus{}, fmt.Errorf("GetPlayers: %w", err)
	}

	players, ok := resp.(PlayersStatus)
	if !ok {
		return PlayersStatus{}, fmt.Errorf("GetPlayers: unexpected response type")
	}
	return players, nil
}

func (c *Client) Abandon() error {
	requestFunc := func() (interface{}, error) {
		urlPath := c.buildURL("/game/abandon")
		reqBody := bytes.NewReader([]byte{})
		req, err := c.newRequest(http.MethodDelete, urlPath, reqBody)
		if err != nil {
			return nil, fmt.Errorf("Abandon: sendRequest: %w", err)
		}
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Abandon: client.Do(req): %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Abandon: unexpected response status: %s", resp.Status)
		}
		time.Sleep(abandonDelay)
		return nil, nil
	}

	_, err := c.doRequest(requestFunc)
	if err != nil {
		return fmt.Errorf("Abandon: %w", err)
	}
	return nil
}

func (c *Client) GetStats() (StatsList, error) {
	requestFunc := func() (interface{}, error) {
		urlPath := c.buildURL("/stats")
		reqBody := bytes.NewReader([]byte{})
		req, err := c.newRequest(http.MethodGet, urlPath, reqBody)
		if err != nil {
			return StatsList{}, fmt.Errorf("GetStats: sendRequest: %w", err)
		}
		resp, err := c.client.Do(req)
		if err != nil {
			return StatsList{}, fmt.Errorf("GetStats: client.Do(req): %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return StatsList{}, fmt.Errorf("GetStats: unexpected response status: %s", resp.Status)
		}
		var stats StatsList
		err = json.NewDecoder(resp.Body).Decode(&stats)
		if err != nil {
			return StatsList{}, fmt.Errorf("GetStats: error decoding response body: %w", err)
		}
		time.Sleep(statusDelay)
		return stats, nil
	}

	resp, err := c.doRequest(requestFunc)
	if err != nil {
		return StatsList{}, fmt.Errorf("GetStats: %w", err)
	}

	result, ok := resp.(StatsList)
	if !ok {
		return StatsList{}, fmt.Errorf("GetStats: unexpected response type")
	}
	return result, nil
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
	if c.Token != "" {
		req.Header.Set("X-Auth-Token", c.Token)
	}
	req.Header.Set("Content-Type", contentType)
	return req, nil
}
func (c *Client) doRequest(requestFunc func() (interface{}, error)) (interface{}, error) {
	var error error
	for i := 0; i < maxRequests; i++ {
		resp, err := requestFunc()
		if err == nil {
			return resp, nil
		}
		error = err
		time.Sleep(requestDelay)
	}
	return nil, fmt.Errorf("request failed after %d retries; %v", maxRequests, error)
}
