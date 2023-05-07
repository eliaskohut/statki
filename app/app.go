package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"main/client"
	"strconv"
	"strings"
	"time"

	gui "github.com/grupawp/warships-gui/v2"
)

// Start initializes the game, gets token, game status, and layout
// then maps the layout from a slice of strings to a format which can be read by gui.
// It creates ui, player's board, and opponent's board. It then listens to player's input,
// logs it, and checks the status to see if the player should fire.
func Start() {
	token, game, layout, err := client.InitGame(true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(token)
	status, err := client.Status(token)
	if err != nil {
		log.Fatal(err)
	}
	//If status is "waiting", wait till the game starts
	for {
		if status.GameStatus != "waiting" {
			break
		}
		fmt.Println("Waiting for an opponent...")
		time.Sleep(1 * time.Second)
	}

	gameDesc, err := client.Description(token)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(gameDesc)

	var mapped [][]int
	mapped, err = mapping(layout)
	if err != nil {
		log.Fatal(err)
	}

	//Craeting ui
	ui := gui.NewGUI(true)
	txt := gui.NewText(1, 1, "Press on any coordinate to log it.", nil)
	txtShouldFire := gui.NewText(1, 3, "", nil)
	ui.Draw(txt)
	ui.Draw(gui.NewText(1, 2, "Press Ctrl+C to exit", nil))

	board := gui.NewBoard(1, 5, nil)
	states := [10][10]gui.State{}
	for _, i := range mapped {
		states[i[0]][i[1]-1] = gui.Ship
	}
	board.SetStates(states)
	playerName := gui.NewText(1, 30, status.Nick, nil)
	playerDesc := gui.NewText(1, 32, formatString(gameDesc.Desc, 5), nil)
	ui.Draw(board)
	ui.Draw(playerName)
	ui.Draw(playerDesc)

	oppBoard := gui.NewBoard(70, 5, nil)
	oppStates := [10][10]gui.State{}
	for i := range states {
		states[i] = [10]gui.State{}
	}
	oppBoard.SetStates(oppStates)
	oppName := gui.NewText(70, 30, status.Opponent, nil)
	oppDesc := gui.NewText(70, 32, formatString(gameDesc.OppDesc, 5), nil)
	ui.Draw(oppBoard)
	ui.Draw(oppName)
	ui.Draw(oppDesc)

	shots := []string{}
	oppShots := status.OppShots
	go func() {
		sunkCount := 0
		oppSunkCount := 0
		states := [10][10]gui.State{}
		for _, i := range mapped {
			states[i[0]][i[1]-1] = gui.Ship
		}
		for {
			if status.ShouldFire {
				txtShouldFire.SetFgColor(gui.Red)
				txtShouldFire.SetText("You should fire!")
				ui.Draw(txtShouldFire)
				ui.Log("You should fire!")
				char := oppBoard.Listen(context.TODO())
				shotRes, err := client.Fire(token, char)
				if err != nil {
					log.Fatal(err)
				}
				x, y, err := stringCoordToInt(char)
				if err != nil {
					log.Fatal(err)
				}
				if shotRes == "hit" {
					oppStates[x][y-1] = gui.Hit
					status.ShouldFire = true
					time.Sleep(time.Millisecond * 200)
				} else if shotRes == "miss" {
					oppStates[x][y-1] = gui.Miss
					time.Sleep(time.Millisecond * 400)
					status.ShouldFire = false
				} else if shotRes == "" {
					time.Sleep(time.Second * 1)
					continue
				} else if shotRes == "sunk" {
					oppStates[x][y-1] = gui.Hit
					sunkCount++
					status.ShouldFire = true
					time.Sleep(time.Millisecond * 200)
				}
				oppBoard.SetStates(oppStates)
				shots = append(shots, char)
				txt.SetText(fmt.Sprintf("Coordinate: %s, %s", char, shotRes))
				ui.Log("Coordinate: %s, %s", char, shotRes)
			} else if !status.ShouldFire {
				txtShouldFire.SetText("It's not your turn")
				txtShouldFire.SetFgColor(gui.Blue)
				ui.Draw(txtShouldFire)
				ui.Log("It's not your turn")
				if len(oppShots) == 0 {
					txtShouldFire.SetText("Waiting for opponent to shoot...")
					time.Sleep(time.Millisecond * 400)
				}
				char := oppShots[len(oppShots)-1]
				shotRes, err := client.Fire(token, char)
				if err != nil {
					log.Fatal(err)
				}
				x, y, err := stringCoordToInt(char)
				if err != nil {
					log.Fatal(err)
				}
				if shotRes == "hit" {
					states[x][y-1] = gui.Hit
					txt.SetText(fmt.Sprintf("Your opponent hit on %s", char))
					ui.Log("Your opponent hit on %s", char)
					status.ShouldFire = false
					time.Sleep(time.Second * 1)
				} else if shotRes == "miss" {
					states[x][y-1] = gui.Miss
					status.ShouldFire = true
					txt.SetText(fmt.Sprintln("Your opponent missed, now your turn"))
					ui.Log("Your opponent missed, now your turn")
					time.Sleep(time.Second * 1)
				} else if shotRes == "sunk" {
					states[x][y-1] = gui.Hit
					txt.SetText(fmt.Sprintln("Your opponent sunk your ship"))
					ui.Log("Your opponent sunk your ship")
					oppSunkCount++
					status.ShouldFire = false
					time.Sleep(time.Second * 1)
				} else if shotRes == "" {
					time.Sleep(time.Second * 1)
					continue
				}

				if sunkCount == 10 {
					txt.SetText("You won!")
					txt.SetBgColor(gui.Green)
					break
				}
				if oppSunkCount == 10 {
					txt.SetText("You lost!")
					txt.SetBgColor(gui.Red)
					break
				}

				board.SetStates(states)

				ui.Log("Coordinate: %s, %s", char, shotRes)
			}
			status, err = client.Status(token)
			if err != nil {
				log.Fatal(err)
			}
			oppShots = status.OppShots
		}
	}()

	ui.Start(nil)

	fmt.Println(game)
	fmt.Println(shots)
}

var ErrInvalidCoord = errors.New("invalid coordinate")

// stringCoordToInt converts a string coordinate to int coordinates
func stringCoordToInt(coord string) (int, int, error) {
	if len(coord) < 2 || len(coord) > 3 {
		return 0, 0, ErrInvalidCoord
	}
	coord = strings.ToUpper(coord)
	if coord[0] < 'A' || coord[0] > 'K' {
		return 0, 0, ErrInvalidCoord
	}
	x := int(coord[0] - 'A')
	y, err := strconv.Atoi(coord[1:])
	if err != nil {
		return 0, 0, ErrInvalidCoord
	}
	return x, y, nil

}

// mapping maps layout from a slice of strings to a format which can be read by gui
func mapping(layout []string) ([][]int, error) {
	var resSlice [][]int
	for _, i := range layout {
		x, y, err := stringCoordToInt(i)
		if err != nil {
			log.Fatal(err)
		}
		resSlice = append(resSlice, []int{x, y})
	}
	return resSlice, nil
}
func formatString(s string, n int) string {
	if n >= len(s) {
		return s
	}

	var buf bytes.Buffer
	for i := 0; i < len(s); i++ {
		if i > 0 && i%n == 0 {
			buf.WriteByte('\n')
		}
		buf.WriteByte(s[i])
	}
	return buf.String()
}
