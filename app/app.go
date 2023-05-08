package app

import (
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

const (
	waitingTime    = time.Second / 3
	squarePickTime = time.Millisecond * 30
	missDelayTime  = time.Millisecond * 400
	hitRes         = "hit"
	missRes        = "miss"
	sunkRes        = "sunk"
	blankRes       = ""
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
	for status.GameStatus == "waiting" {
		fmt.Println("Waiting for an opponent.")
		time.Sleep(waitingTime)
		status, err = client.Status(token)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Waiting for an opponent..")
		time.Sleep(waitingTime)
		status, err = client.Status(token)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Waiting for an opponent...")
		time.Sleep(waitingTime)
		status, err = client.Status(token)
		if err != nil {
			log.Fatal(err)
		}
	}

	gameDesc, err := client.Description(token)
	if err != nil {
		log.Fatal(err)
	}

	var mapped [][]int
	mapped, err = mapping(layout)
	if err != nil {
		log.Fatal(err)
	}

	//Craeting ui
	ui := gui.NewGUI(true)
	txt := gui.NewText(1, 1, "Press on any coordinate to log it.", nil)
	txtShouldFire := gui.NewText(1, 3, "", nil)
	txtTimer := gui.NewText(1, 4, strconv.Itoa(status.Timer), nil)
	ui.Draw(txt)
	ui.Draw(txtShouldFire)
	ui.Draw(txtTimer)
	ui.Draw(gui.NewText(1, 2, "Press Ctrl+C to exit", nil))

	board := gui.NewBoard(1, 5, nil)
	states := [10][10]gui.State{}
	for _, i := range mapped {
		states[i[0]][i[1]-1] = gui.Ship
	}
	board.SetStates(states)
	playerName := gui.NewText(1, 30, status.Nick, nil)
	ui.Draw(board)
	ui.Draw(playerName)
	formatString(gameDesc.Desc, 40, 1, 32, *ui)

	oppBoard := gui.NewBoard(70, 5, nil)
	oppStates := [10][10]gui.State{}
	oppBoard.SetStates(oppStates)
	oppName := gui.NewText(70, 30, status.Opponent, nil)
	ui.Draw(oppBoard)
	ui.Draw(oppName)
	formatString(gameDesc.OppDesc, 40, 70, 32, *ui)

	shots := []string{}

	sunkCount := 0
	oppSunkCount := 0

	go func() {
		for {
			if status.ShouldFire {
				txtShouldFire.SetFgColor(gui.Red)
				txtShouldFire.SetText("You should fire!")
				ui.Log("You should fire!")
				char := oppBoard.Listen(context.TODO())
				time.Sleep(squarePickTime)
				x, y, err := stringCoordToInt(char)
				if err != nil {
					log.Fatal(err)
				}
				for oppStates[x][y-1] == gui.Miss || oppStates[x][y-1] == gui.Hit {
					char = oppBoard.Listen(context.TODO())
					time.Sleep(squarePickTime)
					x, y, err = stringCoordToInt(char)
					if err != nil {
						log.Fatal(err)
					}
				}
				shotRes, err := client.Fire(token, char)
				if err != nil {
					log.Fatal(err)
				}
				if shotRes == hitRes {
					oppStates[x][y-1] = gui.Hit
				} else if shotRes == missRes {
					oppStates[x][y-1] = gui.Miss
					status.ShouldFire = false
					time.Sleep(missDelayTime)
				} else if shotRes == blankRes {
					time.Sleep(time.Second * 1)
					continue
				} else if shotRes == sunkRes {
					oppStates[x][y-1] = gui.Hit
					sunkCount++
				}
				oppBoard.SetStates(oppStates)
				shots = append(shots, char)
				txt.SetText(fmt.Sprintf("Coordinate: %s, %s", char, shotRes))
				ui.Log("%s; Coordinate: %s, %s", status.Nick, char, shotRes)
			} else if !status.ShouldFire {
				oppShotsLen := len(status.OppShots)
				for {
					status, err = client.Status(token)
					if err != nil {
						log.Fatal(err)
					}
					if len(status.OppShots) > oppShotsLen {
						break
					}
					txtShouldFire.SetText("Waiting for opponent to shoot.")
					time.Sleep(waitingTime)
					status, err = client.Status(token)
					if err != nil {
						log.Fatal(err)
					}
					txtShouldFire.SetText("Waiting for opponent to shoot..")
					time.Sleep(waitingTime)
					status, err = client.Status(token)
					if err != nil {
						log.Fatal(err)
					}
					txtShouldFire.SetText("Waiting for opponent to shoot...")
					time.Sleep(waitingTime)
				}

				txtShouldFire.SetText("It's not your turn")
				txtShouldFire.SetFgColor(gui.Blue)
				oppShots := status.OppShots
				char := oppShots[len(oppShots)-1]
				x, y, err := stringCoordToInt(char)
				if err != nil {
					log.Fatal(err)
				}
				if contains(game.Coords, char) {
					states[x][y-1] = gui.Hit
					time.Sleep(time.Millisecond * 50)
					board.SetStates(states)
					txt.SetText(fmt.Sprintf("Your opponent hit on %s", char))
					ui.Log("%s; Coordinate: %s, %s", status.Opponent, char, "hit")
				} else {
					states[x][y-1] = gui.Miss
					txt.SetText(fmt.Sprintf("Your opponent missed on %s", char))
					ui.Log("%s; Coordinate: %s, %s", status.Opponent, char, "miss")
					board.SetStates(states)
					status.ShouldFire = true
				}

			}
			if sunkCount == 10 {
				txt.SetText("You won!")
				txt.SetBgColor(gui.Green)
				break
			} else if oppSunkCount == 10 {
				txt.SetText("You lost!")
				txt.SetBgColor(gui.Red)
				break
			}
		}
		status, err = client.Status(token)
		if err != nil {
			log.Fatal(err)
		}
	}()
	go func() {
		for status.Timer != 0 {
			status, err = client.Status(token)
			if err != nil {
				log.Fatal(err)
			}
			txtTimer.SetText(strconv.Itoa(status.Timer))
			time.Sleep(waitingTime * 3)
		}
		if status.ShouldFire {
			txt.SetText("You won!")
			txt.SetBgColor(gui.Green)
		} else if !status.ShouldFire {
			txt.SetText("You lost!")
			txt.SetBgColor(gui.Red)
		}
	}()
	ui.Start(nil)

	fmt.Println(gameDesc)
	fmt.Println(game)
	fmt.Println(shots)
	fmt.Println(status.OppShots)
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
func formatString(s string, n, x, y int, ui gui.GUI) {
	numSubstrings := len(s) / n
	if len(s)%n != 0 {
		numSubstrings++
	}

	// Create a slice to store the substrings
	substrings := make([]string, numSubstrings)

	// Loop through the string, creating substrings of length n
	for i := 0; i < numSubstrings; i++ {
		start := i * n
		end := (i + 1) * n
		if end > len(s) {
			end = len(s)
		}
		substrings[i] = s[start:end]
	}
	for i, s := range substrings {
		ui.Draw(gui.NewText(x, y+i, s, nil))
	}
}
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
