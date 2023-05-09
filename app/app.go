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

	"github.com/dariubs/percent"

	gui "github.com/grupawp/warships-gui/v2"
)

const (
	waitingTime    = time.Second / 3
	squarePickTime = time.Millisecond * 10
	missDelayTime  = time.Millisecond * 100
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
	var token string
	var game client.Game
	var layout []string
	var err error
	for {
		fmt.Println("Enter your nickname")
		fmt.Println("Press ENTER if you want to auto-generate your nick")
		nickname := ""
		fmt.Scanln(&nickname)

		pDes := ""
		if nickname != "" {
			fmt.Println("Describe yourself")
			fmt.Scanln(&pDes)
		}

		input := ""
		fmt.Println("If you want to play with a bot, type Y")
		fmt.Println("If you want to play with a person, press ENTER")
		fmt.Scanln(&input)
		if err != nil {
			log.Fatal(err)
			return
		}
		if input == "" {
			if nickname != "" {
				token, game, layout, err = client.InitGame(client.Game{WPBot: false, Nick: nickname, Desc: pDes})
				if err != nil {
					log.Fatal(err)
					return
				}
				break
			}
			token, game, layout, err = client.InitGame(client.Game{WPBot: false})
			oppName := ""
			playerList, err := client.GetPlayers()
			if err != nil {
				log.Fatal(err)
				return
			}
			iterate := true
			for iterate {
				printPlayerList(playerList)
				fmt.Println("Enter the name of the player you want to challenge")
				fmt.Scanln(&oppName)
				for _, p := range playerList.PlayerList {
					if p.Nick == oppName {
						iterate = false
						game.TargetNick = oppName
					}
				}
				playerList, err = client.GetPlayers()
				if err != nil {
					log.Fatal(err)
					return
				}
			}
			break

		} else if input == "y" || input == "Y" {
			if nickname != "" {
				token, game, layout, err = client.InitGame(client.Game{WPBot: true, Nick: nickname, Desc: pDes})
				if err != nil {
					log.Fatal(err)
					return
				}
				break
			}
			token, game, layout, err = client.InitGame(client.Game{WPBot: true})
			if err != nil {
				log.Fatal(err)
				return
			}
			break
		}
	}

	status, err := client.Status(token)
	if err != nil {
		log.Fatal(err)
		return
	}
	//If status is "waiting", wait till the game starts

	gameDesc, err := client.Description(token)
	if err != nil {
		log.Fatal(err)
		return
	}

	var mapped [][]int
	mapped, err = mapping(layout)
	if err != nil {
		log.Fatal(err)
		return
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

	board := gui.NewBoard(1, 7, nil)
	states := [10][10]gui.State{}
	for _, i := range mapped {
		states[i[0]][i[1]-1] = gui.Ship
	}
	board.SetStates(states)
	playerName := gui.NewText(1, 30, status.Nick, nil)
	ui.Draw(board)
	ui.Draw(playerName)
	formatString(gameDesc.Desc, 45, 1, 35, *ui)
	accuracy := gui.NewText(1, 5, "", nil)
	ui.Draw(accuracy)

	oppBoard := gui.NewBoard(70, 7, nil)
	oppStates := [10][10]gui.State{}
	oppBoard.SetStates(oppStates)
	oppName := gui.NewText(70, 30, status.Opponent, nil)
	ui.Draw(oppBoard)
	ui.Draw(oppName)
	formatString(gameDesc.OppDesc, 45, 70, 35, *ui)
	oppAccuracy := gui.NewText(70, 5, "", nil)
	ui.Draw(oppAccuracy)

	shots := []string{}
	hitShots := []string{}
	oppShots := []string{}
	oppHitShots := []string{}

	sunkCount := 0
	oppSunkCount := 0

	oppShotsLen := 0
	go func() {
		status, err = client.Status(token)
		if err != nil {
			log.Fatal(err)
			return
		}

		for {
			for status.ShouldFire {
				txtShouldFire.SetFgColor(gui.Red)
				txtShouldFire.SetText("You should fire!")
				ui.Log("You should fire!")
				char := oppBoard.Listen(context.TODO())
				time.Sleep(squarePickTime)
				x, y, err := stringCoordToInt(char)
				if err != nil {
					log.Fatal(err)
					return
				}
				for oppStates[x][y-1] == gui.Miss || oppStates[x][y-1] == gui.Hit {
					char = oppBoard.Listen(context.TODO())
					time.Sleep(squarePickTime)
					x, y, err = stringCoordToInt(char)
					if err != nil {
						log.Fatal(err)
						return
					}
				}
				shotRes, err := client.Fire(token, char)
				if err != nil {
					log.Fatal(err)
					return
				}
				if shotRes == hitRes {
					oppStates[x][y-1] = gui.Hit
					hitShots = append(hitShots, char)
				} else if shotRes == missRes {
					oppStates[x][y-1] = gui.Miss
					time.Sleep(missDelayTime)
					status, err = client.Status(token)
					if err != nil {
						log.Fatal(err)
						return
					}
					status.ShouldFire = false
				} else if shotRes == blankRes {
					time.Sleep(time.Second * 1)
					continue
				} else if shotRes == sunkRes {
					oppStates[x][y-1] = gui.Hit
					hitShots = append(hitShots, char)
					sunkCount++
				}
				oppBoard.SetStates(oppStates)
				shots = append(shots, char)
				txt.SetText(fmt.Sprintf("Coordinate: %s, %s", char, shotRes))
				ui.Log("%s; Coordinate: %s, %s", status.Nick, char, shotRes)
				perAccuracy := percent.PercentOf(len(hitShots), len(shots))
				accuracy.SetText(fmt.Sprintf("Accuracy: %f", perAccuracy))
				if perAccuracy >= 0.6 {
					accuracy.SetBgColor(gui.Green)
				} else if perAccuracy >= 0.4 {
					accuracy.SetBgColor(gui.Blue)
				} else {
					accuracy.SetBgColor(gui.Red)
				}
			}
			for !status.ShouldFire {
				for len(status.OppShots) <= oppShotsLen && status.GameStatus != "ended" {
					txtShouldFire.SetText("Waiting for opponent to shoot.")
					time.Sleep(waitingTime)
					status, err = client.Status(token)
					if err != nil {
						log.Fatal(err)
						return
					}
					txtShouldFire.SetText("Waiting for opponent to shoot..")
					time.Sleep(waitingTime)
					status, err = client.Status(token)
					if err != nil {
						log.Fatal(err)
						return
					}
					txtShouldFire.SetText("Waiting for opponent to shoot...")
					time.Sleep(waitingTime)
				}

				txtShouldFire.SetText("It's not your turn")
				txtShouldFire.SetFgColor(gui.Blue)
				time.Sleep(waitingTime)
				oppShots = status.OppShots
				char := oppShots[oppShotsLen]
				x, y, err := stringCoordToInt(char)
				if err != nil {
					log.Fatal(err)
					return
				}
				if contains(game.Coords, char) && states[x][y-1] != gui.Hit {
					states[x][y-1] = gui.Hit
					board.SetStates(states)
					txt.SetText(fmt.Sprintf("Your opponent hit on %s", char))
					ui.Log("%s; Coordinate: %s, %s", status.Opponent, char, "hit")
					status, err = client.Status(token)
					if err != nil {
						log.Fatal(err)
						return
					}
					oppHitShots = append(oppHitShots, char)
					status.ShouldFire = false
				} else {
					states[x][y-1] = gui.Miss
					txt.SetText(fmt.Sprintf("Your opponent missed on %s", char))
					ui.Log("%s; Coordinate: %s, %s", status.Opponent, char, "miss")
					board.SetStates(states)
					status.ShouldFire = true
				}
				oppPerAccuracy := percent.PercentOf(len(oppHitShots), oppShotsLen)
				oppAccuracy.SetText(fmt.Sprintf("Accuracy: %f", oppPerAccuracy))
				if oppPerAccuracy >= 0.6 {
					oppAccuracy.SetBgColor(gui.Green)
				} else if oppPerAccuracy >= 0.4 {
					oppAccuracy.SetBgColor(gui.Blue)
				} else {
					oppAccuracy.SetBgColor(gui.Red)
				}
				oppShotsLen++
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
			return
		}
	}()
	go func() {
		time.Sleep(waitingTime * 3)
		for status.Timer != 0 {
			status, err = client.Status(token)
			if err != nil {
				log.Fatal(err)
				return
			}
			txtTimer.SetText(strconv.Itoa(status.Timer))
			time.Sleep(waitingTime)
		}
		if status.ShouldFire {
			txt.SetText("You won!")
			txt.SetBgColor(gui.Green)
			txtShouldFire.SetText("Congratulations!")
			txtShouldFire.SetFgColor(gui.Green)
		} else if !status.ShouldFire {
			txt.SetText("You lost!")
			txt.SetBgColor(gui.Red)
			txtShouldFire.SetText("No surprise you lost!")
			txtShouldFire.SetFgColor(gui.Red)
		}
	}()
	ui.Start(nil)

	fmt.Println(gameDesc)
	fmt.Println(game)
	fmt.Println(shots)
	fmt.Println(status.OppShots)
	fmt.Println(status)
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
			return nil, err
		}
		resSlice = append(resSlice, []int{x, y})
	}
	return resSlice, nil
}
func formatString(s string, n, x, y int, ui gui.GUI) {
	var substrings []string

	for len(s) > 0 {
		end := n
		if len(s) < n {
			end = len(s)
		}
		lastSpace := strings.LastIndex(s[:end], " ")
		if lastSpace == -1 {
			lastSpace = end
		}

		substrings = append(substrings, s[:lastSpace])

		if lastSpace == end {
			s = s[end:]
		} else {
			s = s[lastSpace+1:]
		}
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
func printPlayerList(playerList client.PlayerList) {
	fmt.Printf("%+v\n", playerList)
}
