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

func Start() {
	//Initializing the game, getting token, game status,
	//and layout
	token, game, layout, err := client.InitGame(true)
	if err != nil {
		log.Fatal(err)
	}
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

	//Mapping layout from a slice of strings to
	//a format, which can be read by gui
	var mapped [][]int
	mapped, err = mapping(layout)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(mapped)

	//Craeting ui
	ui := gui.NewGUI(true)
	txt := gui.NewText(1, 1, "Press on any coordinate to log it.", nil)
	txtShouldFire := gui.NewText(1, 3, "", nil)
	ui.Draw(txt)
	ui.Draw(gui.NewText(1, 2, "Press Ctrl+C to exit", nil))

	//Creating player's board
	board := gui.NewBoard(1, 5, nil)
	states := [10][10]gui.State{}
	for _, i := range mapped {
		states[i[0]][i[1]-1] = gui.Ship
	}
	board.SetStates(states)
	playerName := gui.NewText(1, 30, status.Nick, nil)
	playerDesc := gui.NewText(1, 32, formatString(status.Desc, 5), nil)
	ui.Draw(board)
	ui.Draw(playerName)
	ui.Draw(playerDesc)

	//Creating opponent's board
	oppboard := gui.NewBoard(70, 5, nil)
	oppstates := [10][10]gui.State{}
	for i := range states {
		states[i] = [10]gui.State{}
	}
	oppboard.SetStates(oppstates)
	oppName := gui.NewText(70, 30, status.Opponent, nil)
	oppDesc := gui.NewText(70, 32, formatString(status.OppDesc, 5), nil)
	ui.Draw(oppboard)
	ui.Draw(oppName)
	ui.Draw(oppDesc)

	go func() {
		for {
			if status.ShouldFire {
				txtShouldFire.SetFgColor(gui.Red)
				txtShouldFire.SetText("You should fire!")
				ui.Draw(txtShouldFire)
				ui.Log("You should fire!")
			} else {
				txtShouldFire.SetText("It's not your turn")
				txtShouldFire.SetFgColor(gui.Blue)
				ui.Draw(txtShouldFire)
				ui.Log("You should fire!")
			}
			char := board.Listen(context.TODO())
			txt.SetText(fmt.Sprintf("Coordinate: %s", char))
			ui.Log("Coordinate: %s", char) // logs are displayed after the game exits
		}
	}()

	ui.Start(nil)

	// fmt.Printf("\n\nMe: %s \n", status.Nick)
	// fmt.Printf("%s \n\n", status.Desc)
	// fmt.Printf("Player2: %s \n", status.Opponent)
	// fmt.Printf("Opponent description: %s \n\n", status.OppDesc)
	// fmt.Println(token)

	fmt.Println(game)
	// fmt.Println(status)
}

var ErrInvalidCoord = errors.New("invalid coordinate")

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
func mapping(layout []string) ([][]int, error) {
	var resslice [][]int
	for _, i := range layout {
		x, y, err := stringCoordToInt(i)
		if err != nil {
			log.Fatal(err)
		}
		resslice = append(resslice, []int{x, y})
	}
	return resslice, nil
}
func formatString(s string, n int) string {
	runes := []rune(s)
	var result string
	for i, r := range runes {
		if i > 0 && i%n == 0 {
			result += "\n"
		}
		result += string(r)
	}
	return result
}
