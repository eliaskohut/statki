package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	gui "github.com/grupawp/warships-gui/v2"
	"github.com/micmonay/keybd_event"
	"log"
	"main/client"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	waitingTime   = time.Second / 3
	countdownTime = 30
	hitRes        = "hit"
	missRes       = "miss"
	sunkRes       = "sunk"
	blankRes      = ""
	yBoards       = 8
	xPBoard       = 1
	xOBoard       = 100
)

var (
	pCtx = context.TODO()
)

func (a *App) Start() {
	for {
		ctx, cancelCtx := context.WithCancel(pCtx)
		ctxFleet, cancelCtxFleet := context.WithCancel(pCtx)
		ui, err := a.makeUI()
		if err != nil {
			log.Fatalf("app Start() 4, a.makeUI(); %v", err)
		}
		a.Ui = ui
		a.Client = client.NewClient()
		game, err := a.getDetails(a.Client, ctxFleet, cancelCtxFleet)
		if err != nil {
			log.Fatalf("app Start() 1, a.getDetails(); %v", err)
		}

		if a.Client.Token != "" {
			a.Status, err = a.Client.GetStatus()
			if err != nil {
				log.Fatalf("app Start() 2, client.GetStatus(); %v", err)
			}

			gameDesc, err := a.Client.GetDescription()
			if err != nil {
				log.Fatalf("app Start() 3, client.GetDescription(); %v", err)
			}
			a.Nick = gameDesc.Nick
			a.Desc = gameDesc.Desc
			a.TargetNick = gameDesc.Opponent
			a.ODesc = gameDesc.OppDesc

			guiBattle := a.buildBattlefield(ui)

			ans, err := a.Client.GetBoard()
			if err != nil {
				log.Fatalf("app Start() 5, client.GetBoard(); %v", err)
			}
			layout := ans.Board

			a.PlayerBoard = layout

			var mapped [][]int
			mapped, err = a.mappingChars(layout)
			if err != nil {
				log.Fatalf("app Start() 6, a.mappingChars(); %v", err)
			}
			states := [10][10]gui.State{}
			for _, i := range mapped {
				states[i[0]][i[1]-1] = gui.Ship
			}
			guiBattle.PlayerBoardStates = states
			guiBattle.PlayerBoard.SetStates(states)

			go a.startBattle(guiBattle, ctx, cancelCtx)

			guiBattle.Ui.Start(ctx, nil)

			fmt.Println(game)
		} else {
			cancelCtx()
			break
		}
	}
}

func (a *App) timerUpdate(guiB *GuiBattle, ctx context.Context, cancelCtx context.CancelFunc) {
	var winner string
	quitChan := make(chan bool)
	if a.Client.Token == "" {
		return
	}
	for a.Status.GameStatus != "ended" {
		select {
		default:
			var err error
			if a.Client.Token != "" {
				a.Status, err = a.Client.GetStatus()
				if err != nil {
					log.Fatalf("app timerUpdate() 7, client.GetStatus(); %v", err)
				}
				guiB.Timer.SetText(fmt.Sprintf("Time: %v", a.Status.Timer))
			} else {
				cancelCtx()
			}
		case <-ctx.Done():
			if a.Client.Token != "" {
				err := a.Client.Abandon()
				if err != nil {
					log.Fatalf("app timerUpdate() 8, client.Abandon(); %v", err)
				}
			}
			quitChan <- true
		}
	}
	winner = a.Nick
	if a.Status.LastGameStatus == "lose" {
		winner = a.TargetNick
	}
	guiB.Ui.Remove(guiB.PlayerNick)
	guiB.Ui.Remove(guiB.PlayerBoard)
	guiB.Ui.Remove(guiB.PlayerAccuracy)
	guiB.Ui.Remove(guiB.OpponentNick)
	guiB.Ui.Remove(guiB.OpponentBoard)
	guiB.Ui.Remove(guiB.ShotResult)
	guiB.Ui.Remove(guiB.OppShotResult)
	guiB.Ui.Remove(guiB.ShouldFire)
	guiB.Ui.Remove(guiB.Timer)
	guiB.Ui.Remove(guiB.OpponentAccuracy)
	guiB.Ui.Draw(gui.NewText(xPBoard, yBoards, fmt.Sprintf("Winner: %s", winner), nil))
	guiB.Exit.SetText("To start a new game press CTRL+C")
	quitChan <- true
}
func (a *App) getAdjacentCoordinates(coord string) []string {
	row := int(coord[1] - '1')
	col := int(coord[0] - 'A')
	directions := [8][2]int{
		{-1, -1}, {-1, 0}, {-1, 1},
		{0, -1}, {0, 1},
		{1, -1}, {1, 0}, {1, 1},
	}

	adjacentCoords := make([]string, 0)

	for _, dir := range directions {
		newRow := row + dir[0]
		newCol := col + dir[1]
		if newRow >= 0 && newRow < 10 && newCol >= 0 && newCol < 10 {
			newCoord := fmt.Sprintf("%c%d", 'A'+newCol, newRow+1)
			adjacentCoords = append(adjacentCoords, newCoord)
		}
	}

	return adjacentCoords
}

func (a *App) markAdjacentMisses(guiB *GuiBattle, hitShots []string, char string) {
	adjacentCoords := make(map[string]bool)

	// Iterate over each hitShot and collect all adjacent coordinates
	for _, shot := range hitShots {
		coords := a.getAdjacentCoordinates(shot)
		for _, coord := range coords {
			adjacentCoords[coord] = true
		}
	}

	// Mark adjacent tiles as Miss
	for coord := range adjacentCoords {
		row := int(coord[1] - '1')
		col := int(coord[0] - 'A')

		if guiB.OpponentBoardStates[row][col] != gui.Hit {
			guiB.OpponentBoardStates[row][col] = gui.Miss
		}
	}
}

func (a *App) startBattle(guiB *GuiBattle, ctx context.Context, cancelCtx context.CancelFunc) {
	var err error
	shots := make([]string, 0)
	hitShots := make([]string, 0)
	oppHitShots := make([]string, 0)
	quitChan := make(chan bool)
	go a.timerUpdate(guiB, ctx, cancelCtx)
	a.Status, err = a.Client.GetStatus()
	if err != nil {
		log.Fatalf("app startBattle() 9, client.GetStatus(); %v", err)
	}
	board, err := a.Client.GetBoard()
	if err != nil {
		log.Fatalf("app startBattle() 10, client.GetBoard(); %v", err)
	}
	a.PlayerBoard = board.Board
	for a.Status.GameStatus != "ended" {
		select {
		default:
			time.Sleep(waitingTime)
			a.Status, err = a.Client.GetStatus()
			if err != nil {
				log.Fatalf("app startBattle() 11, client.GetStatus(); %v", err)
			}
			if a.Status.ShouldFire {
				for _, shot := range a.Status.OppShots {
					res := ""
					x, y, err := a.stringCoordToInt(shot)
					if err != nil {
						log.Fatalf("app startBattle() 12, a.stringCoordToInt(); %v", err)
					}
					if a.contains(shot, a.PlayerBoard) && guiB.PlayerBoardStates[x][y-1] != gui.Hit {
						guiB.PlayerBoardStates[x][y-1] = gui.Hit
						oppHitShots = append(oppHitShots, shot)
						res = "hit"
					} else if !a.contains(shot, a.PlayerBoard) {
						guiB.PlayerBoardStates[x][y-1] = gui.Miss
						res = "miss"
					}
					guiB.OppShotResult.SetText(fmt.Sprintf("%s, %s on %s", a.TargetNick, res, shot))
				}
				guiB.OpponentAccuracy.SetText(fmt.Sprintf("Accuracy: %v / %v", len(oppHitShots), len(a.Status.OppShots)))
				guiB.PlayerBoard.SetStates(guiB.PlayerBoardStates)
				guiB.ShouldFire.SetText("Fire!")
				guiB.ShouldFire.SetFgColor(gui.Green)
				char := guiB.OpponentBoard.Listen(ctx)
				select {
				case <-ctx.Done():
					quitChan <- true
				default:
					break
				}
				x, y, err := a.stringCoordToInt(char)
				if err != nil {
					log.Fatalf("app startBattle() 13, a.stringCoordToInt(); %v", err)
				}
				for guiB.OpponentBoardStates[x][y-1] == gui.Miss || guiB.OpponentBoardStates[x][y-1] == gui.Hit {
					guiB.ShouldFire.SetText("You can't fire there!")
					char = guiB.OpponentBoard.Listen(ctx)
					x, y, err = a.stringCoordToInt(char)
					if err != nil {
						log.Fatalf("app startBattle() 14, a.stringCoordToInt(); %v", err)
					}
				}
				if a.Status.GameStatus == "ended" {
					break
				}
				shots = append(shots, char)
				result, err := a.Client.Shoot(char)
				if err != nil {
					log.Fatalf("app startBattle() 15, client.Shoot(); %v", err)
				}
				if result == hitRes {
					hitShots = append(hitShots, char)
					guiB.OpponentBoardStates[x][y-1] = gui.Hit
				} else if result == sunkRes {
					hitShots = append(hitShots, char)
					x, y, err := a.stringCoordToInt(char)
					if err != nil {
						log.Fatalf("app startBattle() miss check, a.stringCoordToInt(); %v", err)
					}
					guiB.OpponentBoardStates[x][y-1] = gui.Hit

					//a.markAdjacentMisses(guiB, hitShots, char)
				} else if result == blankRes {
					continue
				} else if result == missRes {
					guiB.OpponentBoardStates[x][y-1] = gui.Miss
					guiB.ShouldFire.SetText("It's not your turn!")
					guiB.ShouldFire.SetFgColor(gui.Red)
				}
				guiB.OpponentBoard.SetStates(guiB.OpponentBoardStates)
				guiB.ShotResult.SetText(fmt.Sprintf("%s, %s on %s", a.Nick, result, char))
				guiB.PlayerAccuracy.SetText(fmt.Sprintf("Accuracy: %v / %v", len(hitShots), len(shots)))
				a.Status, err = a.Client.GetStatus()
				if err != nil {
					log.Fatalf("app startBattle() 20, client.GetStatus(); %v", err)
				}
			} else {
				time.Sleep(waitingTime)
				a.Status, err = a.Client.GetStatus()
				if err != nil {
					log.Fatalf("app startBattle() 21, client.GetStatus(); %v", err)
				}
			}
			a.Status, err = a.Client.GetStatus()
			if err != nil {
				log.Fatalf("app startBattle() 22, client.GetStatus(); %v", err)
			}
		case <-ctx.Done():
			err := a.Client.Abandon()
			if err != nil {
				log.Fatalf("Error abandoning, %v", err)
			}
			return
		}
	}
	winner := a.Nick
	if a.Status.LastGameStatus == "lose" {
		winner = a.TargetNick
	}
	guiB.Ui.Remove(guiB.PlayerNick)
	guiB.Ui.Remove(guiB.PlayerBoard)
	guiB.Ui.Remove(guiB.PlayerAccuracy)
	guiB.Ui.Remove(guiB.OpponentNick)
	guiB.Ui.Remove(guiB.OpponentBoard)
	guiB.Ui.Remove(guiB.ShotResult)
	guiB.Ui.Remove(guiB.OppShotResult)
	guiB.Ui.Remove(guiB.ShouldFire)
	guiB.Ui.Remove(guiB.Timer)
	guiB.Ui.Remove(guiB.OpponentAccuracy)
	guiB.Ui.Draw(gui.NewText(xPBoard, yBoards, fmt.Sprintf("Winner: %s", winner), nil))
	guiB.Exit.SetText("To start a new game press CTRL+C")
	guiB.Ui.Log(fmt.Sprintf("Winner: %s", winner))
	quitChan <- true
}

var ErrInvalidCoord = errors.New("invalid coordinate")

// stringCoordToInt converts a string coordinate to int coordinates
func (a *App) stringCoordToInt(coord string) (int, int, error) {
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

func (a *App) intCoordToString(x, y int) (string, error) {
	if x < 0 || x >= 10 || y < 1 || y > 10 {
		return "", ErrInvalidCoord
	}
	return string(rune('A'+x)) + strconv.Itoa(y), nil
}

func (a *App) mappingInts(coords [][]int) ([]string, error) {
	var resSlice []string
	for _, coord := range coords {
		if len(coord) != 2 {
			return nil, ErrInvalidCoord
		}
		x, y := coord[0], coord[1]
		strCoord, err := a.intCoordToString(x, y)
		if err != nil {
			return nil, err
		}
		resSlice = append(resSlice, strCoord)
	}
	return resSlice, nil
}

// mappingChars maps layout from a slice of strings to a format which can be read by gui
func (a *App) mappingChars(layout []string) ([][]int, error) {
	var resSlice [][]int
	for _, i := range layout {
		x, y, err := a.stringCoordToInt(i)
		if err != nil {
			log.Fatalf("app mappingChars() 25, a.stringCoordToInt(); %v", err)
			return nil, err
		}
		resSlice = append(resSlice, []int{x, y})
	}
	return resSlice, nil
}

func (a *App) contains(e string, s []string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (a *App) getPlayerName() string {
	var playerName string
	nameInput := widgets.NewParagraph()
	nameInput.Title = "Enter Your Name. Press Enter if you want to auto-generate your name"
	nameInput.TextStyle = termui.NewStyle(termui.ColorYellow)
	nameInput.SetRect(0, 0, 80, 3)

	errorMsg := widgets.NewParagraph()
	errorMsg.TextStyle = termui.NewStyle(termui.ColorRed)
	errorMsg.SetRect(0, 4, 50, 7)

	termui.Render(nameInput, errorMsg)

	name := ""
	nameEvents := termui.PollEvents()
	for {
		nameEv := <-nameEvents
		switch nameEv.Type {
		case termui.KeyboardEvent:
			switch nameEv.ID {
			case "<Enter>":
				playerName = strings.TrimSpace(name)
				if playerName == "" {
					return ""
				} else if a.validateName(playerName) {
					return playerName
				} else {
					termui.Render(nameInput) // Clear error message from the screen
					if len(playerName) < 2 || len(playerName) > 10 {
						errorMsg.Text = "Please enter a name between 2 and 10 characters."
					} else {
						errorMsg.Text = "Name cannot contain symbols.\nPlease enter a valid name."
					}
					termui.Render(errorMsg)
				}
			case "<Escape>":
				termui.Close()
				kb, err := keybd_event.NewKeyBonding()
				if err != nil {
					log.Fatalf("ui keybind 26, keybd_event.NewKeyBonding(); %v", err)
				}
				if runtime.GOOS == "linux" {
					time.Sleep(2 * time.Second)
				}

				kb.SetKeys(keybd_event.VK_C)

				kb.HasCTRL(true)

				err = kb.Launching()
				if err != nil {
					log.Fatal(err)
				}

				err = kb.Press()
				if err != nil {
					log.Fatalf("ui keypress 27, kb.Press(); %v", err)
				}
				time.Sleep(10 * time.Millisecond)
				err = kb.Release()
				if err != nil {
					log.Fatalf("ui keypress 28, kb.Release(); %v", err)
				}
				break
			case "<Backspace>":
				if len(name) > 0 {
					name = name[:len(name)-1]
					nameInput.Text = name
					termui.Render(nameInput)
				}
			default:
				// Accumulate the characters in the name variable
				if len(nameEv.ID) == 1 && len(name) < 10 {
					name += nameEv.ID
					nameInput.Text = name
					termui.Render(nameInput)
				}
			}
		}
	}
}

func (a *App) validateName(name string) bool {
	return len(name) >= 2 && len(name) <= 10 && !a.containsSymbols(name)
}

func (a *App) containsSymbols(name string) bool {
	for _, ch := range name {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= 'A' && ch <= 'Z':
		case ch >= '0' && ch <= '9':
		case ch == ' ':
		default:
			return true
		}
	}
	return false
}

func (a *App) triggerResize(list *widgets.List) {
	termui.Clear()
	width, height := termui.TerminalDimensions()
	if list != nil {
		list.SetRect(0, 0, width, height)
	}
	termui.Render(list)
}

func (a *App) getPlayerDescription() string {
	descInput := widgets.NewParagraph()
	descInput.Title = "Enter Your Description"
	descInput.TextStyle = termui.NewStyle(termui.ColorYellow)
	descInput.SetRect(0, 0, 50, 10)

	errorMsg := widgets.NewParagraph()
	errorMsg.TextStyle = termui.NewStyle(termui.ColorRed)
	errorMsg.SetRect(0, 12, 50, 15)

	termui.Render(descInput, errorMsg)

	description := ""
	descEvents := termui.PollEvents()
	for {
		descEv := <-descEvents
		switch descEv.Type {
		case termui.KeyboardEvent:
			switch descEv.ID {
			case "<Enter>":
				description = strings.TrimSpace(description)
				if a.validateDescription(description) {
					return description
				} else {
					termui.Render(descInput)
					if len(description) < 5 || len(description) > 200 {
						errorMsg.Text = "Please enter a description between 5 and 200 characters."
					}
					termui.Render(errorMsg)
				}
			case "<Escape>":
				termui.Close()
				kb, err := keybd_event.NewKeyBonding()
				if err != nil {
					log.Fatalf("ui keybind 29, keybd_event.NewKeyBonding(); %v", err)
				}
				if runtime.GOOS == "linux" {
					time.Sleep(2 * time.Second)
				}

				kb.SetKeys(keybd_event.VK_C)

				kb.HasCTRL(true)

				err = kb.Launching()
				if err != nil {
					log.Fatal(err)
				}

				err = kb.Press()
				if err != nil {
					log.Fatalf("ui keypress 30, kb.Press(); %v", err)
				}
				time.Sleep(10 * time.Millisecond)
				err = kb.Release()
				if err != nil {
					log.Fatalf("ui keypress 31, kb.Release(); %v", err)
				}
				break
			case "<Backspace>":
				if len(description) > 0 {
					// Remove the last character from the description
					description = description[:len(description)-1]
					descInput.Text = description
					termui.Render(descInput)
				}
			default:
				// Accumulate the characters in the description variable
				if len(descEv.ID) == 1 && len(description) < 200 {
					description += descEv.ID
					descInput.Text = description
					termui.Render(descInput)
				}
			}
		}
	}
}

func (a *App) validateDescription(description string) bool {
	return len(description) >= 5 && len(description) <= 200
}

func (a *App) getDetails(c *client.Client, ctx context.Context, cancelFunc context.CancelFunc) (client.Game, error) {
	var nick string
	var pDes string
	game := client.Game{}
	if err := termui.Init(); err != nil {
		log.Fatalf("Failed to initialize termui 32: %v", err)
	}

	options := []string{"Play with a bot", "Wait for an opponent", "Challenge someone", "Show stats"}

	list := widgets.NewList()
	list.Title = "Choose an Option"
	list.Rows = options
	list.SelectedRowStyle = termui.NewStyle(termui.ColorGreen, termui.ColorBlack)

	termui.Render(list)
	a.triggerResize(list)

	uiEvents := termui.PollEvents()

mainLoop:
	for {
		ev := <-uiEvents
		switch ev.Type {
		case termui.KeyboardEvent:
			switch ev.ID {
			case "<Down>":
				list.ScrollDown()
			case "<Up>":
				list.ScrollUp()
			case "<Enter>":
				selectedOption := options[list.SelectedRow]

				if selectedOption == "Play with a bot" {
					termui.Clear()
					nick = a.getPlayerName()
					if nick != "" {
						termui.Clear()
						pDes = a.getPlayerDescription()
						fleet := a.getLayout(a.Ui, ctx, cancelFunc)
						termui.Clear()
						if len(fleet) != 0 {
							game, err := c.InitGame(client.Game{Nick: nick, Desc: pDes, WPBot: true, Coords: fleet})
							if err != nil {
								log.Fatalf("a.getDetails 33, c.InitGame; %v", err)
							}
							a.waitForOpponent(waitingTime)
							termui.Close()
							return game, nil
						} else {
							game, err := c.InitGame(client.Game{Nick: nick, Desc: pDes, WPBot: true})
							if err != nil {
								log.Fatalf("a.getDetails 33, c.InitGame; %v", err)
							}
							a.waitForOpponent(waitingTime)
							termui.Close()
							return game, nil
						}
					} else {
						fleet := a.getLayout(a.Ui, ctx, cancelFunc)
						termui.Clear()
						if len(fleet) != 0 {
							game, err := c.InitGame(client.Game{WPBot: true, Coords: fleet})
							if err != nil {
								log.Fatalf("a.getDetails 34, c.InitGame; %v", err)
							}
							a.waitForOpponent(waitingTime)
							termui.Close()
							return game, nil
						} else {
							game, err := c.InitGame(client.Game{WPBot: true})
							if err != nil {
								log.Fatalf("a.getDetails 34, c.InitGame; %v", err)
							}
							a.waitForOpponent(waitingTime)
							termui.Close()
							return game, nil
						}
					}
				} else if selectedOption == "Wait for an opponent" {
					termui.Clear()
					nick = a.getPlayerName()
					if nick != "" {
						termui.Clear()

						pDes = a.getPlayerDescription()
						termui.Clear()

						fleet := a.getLayout(a.Ui, ctx, cancelFunc)
						termui.Clear()

						if len(fleet) != 0 {
							game, err := c.InitGame(client.Game{Nick: nick, Desc: pDes, WPBot: false, Coords: fleet})
							if err != nil {
								log.Fatalf("a.getDetails 35, c.InitGame; %v", err)
							}
							a.waitForOpponent(waitingTime)
							termui.Close()
							return game, nil
						} else {
							game, err := c.InitGame(client.Game{Nick: nick, Desc: pDes, WPBot: false})
							if err != nil {
								log.Fatalf("a.getDetails 35, c.InitGame; %v", err)
							}
							a.waitForOpponent(waitingTime)
							termui.Close()
							return game, nil
						}
					} else {
						fleet := a.getLayout(a.Ui, ctx, cancelFunc)
						termui.Clear()
						if len(fleet) != 0 {
							game, err := c.InitGame(client.Game{WPBot: false, Coords: fleet})
							if err != nil {
								log.Fatalf("a.getDetails 36, c.InitGame; %v", err)
							}
							a.waitForOpponent(waitingTime)
							termui.Close()
							return game, nil
						} else {
							game, err := c.InitGame(client.Game{WPBot: false})
							if err != nil {
								log.Fatalf("a.getDetails 36, c.InitGame; %v", err)
							}
							a.waitForOpponent(waitingTime)
							termui.Close()
							return game, nil
						}
					}
				} else if selectedOption == "Challenge someone" {
					termui.Clear()
					nick = a.getPlayerName()
					if nick != "" {
						termui.Clear()
						pDes = a.getPlayerDescription()

						fleet := a.getLayout(a.Ui, ctx, cancelFunc)
						termui.Clear()

						targetNick := a.getTarget(c)
						if targetNick == "" {
							log.Fatalf("Target nick cannot be empty")
						}
						if len(fleet) != 0 {
							game, err := c.InitGame(client.Game{Nick: nick, Desc: pDes, WPBot: false, TargetNick: targetNick, Coords: fleet})
							if err != nil {
								log.Fatalf("a.getDetails 37, c.InitGame; %v", err)
							}
							a.waitForOpponent(waitingTime)
							termui.Close()
							return game, nil
						} else {
							game, err := c.InitGame(client.Game{Nick: nick, Desc: pDes, WPBot: false, TargetNick: targetNick})
							if err != nil {
								log.Fatalf("a.getDetails 37, c.InitGame; %v", err)
							}
							a.waitForOpponent(waitingTime)
							termui.Close()
							return game, nil
						}
					} else {
						termui.Clear()
						fleet := a.getLayout(a.Ui, ctx, cancelFunc)
						termui.Clear()
						targetNick := a.getTarget(c)
						if targetNick == "" {
							log.Fatalf("Target nick cannot be empty")
						}
						if len(fleet) != 0 {
							game, err := c.InitGame(client.Game{WPBot: false, TargetNick: targetNick, Coords: fleet})
							if err != nil {
								log.Fatalf("a.getDetails 38, c.InitGame; %v", err)
							}
							a.waitForOpponent(waitingTime)
							termui.Close()
							return game, nil
						} else {
							game, err := c.InitGame(client.Game{WPBot: false, TargetNick: targetNick})
							if err != nil {
								log.Fatalf("a.getDetails 38, c.InitGame; %v", err)
							}
							a.waitForOpponent(waitingTime)
							termui.Close()
							return game, nil
						}
					}
				} else if selectedOption == "Show stats" {
					termui.Clear()
					stats, err := a.Client.GetStats()
					if err != nil {
						log.Fatal(err)
					}
					stringStats := make([]string, 0)
					for i, stat := range stats.Stats {
						str := fmt.Sprintf("%d. Games: %d, Nick: %s, Points: %d, Rank: %d, Wins: %d",
							i+1, stat.Games, stat.Nick, stat.Points, stat.Rank, stat.Wins)
						stringStats = append(stringStats, str)
					}
					stringStats = append(stringStats, "Back")

					statList := widgets.NewList()
					statList.Rows = stringStats
					statList.SelectedRowStyle = termui.NewStyle(termui.ColorGreen, termui.ColorBlack)

					a.triggerResize(statList)
					termui.Render(statList)

					// Stats loop
					for {
						statEv := <-uiEvents
						switch statEv.Type {
						case termui.KeyboardEvent:
							switch statEv.ID {
							case "<Down>":
								statList.ScrollDown()
							case "<Up>":
								statList.ScrollUp()
							case "<Enter>":
								selectedStat := stringStats[statList.SelectedRow]
								if selectedStat == "Back" {
									termui.Clear()
									termui.Render(list)
									continue mainLoop
								}
							case "<Escape>":
								termui.Close()
								kb, err := keybd_event.NewKeyBonding()
								if err != nil {
									log.Fatalf("ui keybind 39, keybd_event.NewKeyBonding(); %v", err)
								}
								if runtime.GOOS == "linux" {
									time.Sleep(2 * time.Second)
								}

								kb.SetKeys(keybd_event.VK_C)

								kb.HasCTRL(true)

								err = kb.Launching()
								if err != nil {
									log.Fatal(err)
								}

								err = kb.Press()
								if err != nil {
									log.Fatalf("ui keypress 40, kb.Press(); %v", err)
								}
								time.Sleep(10 * time.Millisecond)
								err = kb.Release()
								if err != nil {
									log.Fatalf("ui keypress 41, kb.Release(); %v", err)
								}
								break
							}
							termui.Render(statList)
						case termui.ResizeEvent:
							payload := ev.Payload.(termui.Resize)
							termui.Clear()
							list.SetRect(0, 0, payload.Width, payload.Height)
							termui.Render(list)
						}
					}
				}
			case "<Escape>":
				termui.Close()
				kb, err := keybd_event.NewKeyBonding()
				if err != nil {
					log.Fatalf("ui keybind 39, keybd_event.NewKeyBonding(); %v", err)
				}
				if runtime.GOOS == "linux" {
					time.Sleep(2 * time.Second)
				}

				kb.SetKeys(keybd_event.VK_C)

				kb.HasCTRL(true)

				err = kb.Launching()
				if err != nil {
					log.Fatal(err)
				}

				err = kb.Press()
				if err != nil {
					log.Fatalf("ui keypress 40, kb.Press(); %v", err)
				}
				time.Sleep(10 * time.Millisecond)
				err = kb.Release()
				if err != nil {
					log.Fatalf("ui keypress 41, kb.Release(); %v", err)
				}
				break
			}
			termui.Render(list)
		case termui.ResizeEvent:
			payload := ev.Payload.(termui.Resize)
			termui.Clear()
			list.SetRect(0, 0, payload.Width, payload.Height)
			termui.Render(list)
		}

	}
	return game, nil
}

func (a *App) getLayout(ui *gui.GUI, ctx context.Context, cancelFunc context.CancelFunc) []string {
	options := []string{"Yes", "No"}
	termui.Clear()
	list := widgets.NewList()
	list.Title = "Do you want to build your own fleet?"
	list.Rows = options
	list.SelectedRowStyle = termui.NewStyle(termui.ColorGreen, termui.ColorBlack)
	termui.Render(list)
	a.triggerResize(list)

	uiEvents := termui.PollEvents()
	for {
		ev := <-uiEvents
		switch ev.Type {
		case termui.KeyboardEvent:
			switch ev.ID {
			case "<Down>":
				list.ScrollDown()
			case "<Up>":
				list.ScrollUp()
			case "<Enter>":
				selectedOption := options[list.SelectedRow]
				if selectedOption == "Yes" {
					fleet, err := a.makeFleet(ui, ctx, cancelFunc)
					if err != nil {
						log.Fatalf("app Start() 4, a.makeFleet(); %v", err)
					}
					return fleet
				} else if selectedOption == "No" {
					return nil
				}
			case "<Escape>":
				termui.Close()
				kb, err := keybd_event.NewKeyBonding()
				if err != nil {
					log.Fatalf("ui keybind 39, keybd_event.NewKeyBonding(); %v", err)
				}
				if runtime.GOOS == "linux" {
					time.Sleep(2 * time.Second)
				}

				kb.SetKeys(keybd_event.VK_C)

				kb.HasCTRL(true)

				err = kb.Launching()
				if err != nil {
					log.Fatal(err)
				}

				err = kb.Press()
				if err != nil {
					log.Fatalf("ui keypress 40, kb.Press(); %v", err)
				}
				time.Sleep(10 * time.Millisecond)
				err = kb.Release()
				if err != nil {
					log.Fatalf("ui keypress 41, kb.Release(); %v", err)
				}
				break
			}
			termui.Render(list)
		case termui.ResizeEvent:
			payload := ev.Payload.(termui.Resize)
			termui.Clear()
			list.SetRect(0, 0, payload.Width, payload.Height)
			termui.Render(list)
		}
	}

}

func (a *App) getTarget(c *client.Client) string {
	err := termui.Init()
	if err != nil {
		return ""
	}
	playerList, err := c.GetPlayers()
	if err != nil {
		log.Fatalf("a.getTarget 42, c.GetPlayers; %v", err)
	}

	arePlayers := false
	if len(playerList) == 0 {
		time.Sleep(waitingTime * 3)
		playerList, err := c.GetPlayers()
		if err != nil {
			log.Fatalf("a.getTarget 43, c.GetPlayers; %v", err)
		}
		if len(playerList) == 0 {
			timerMsg := widgets.NewParagraph()
			timerMsg.Text = "No players available"
			timerMsg.TextStyle = termui.NewStyle(termui.ColorYellow)
			timerMsg.SetRect(0, 0, 50, 3)

			termui.Render(timerMsg)

			countdown := countdownTime
			ticker := time.NewTicker(time.Second * 1)
			defer ticker.Stop()

			for range ticker.C {
				countdown--
				timerMsg.Text = fmt.Sprintf("No players available - Waiting %d seconds", countdown)
				termui.Render(timerMsg)
				if countdown%2 == 0 {
					playerList, err = c.GetPlayers()
					if err != nil {
						log.Fatalf("a.getTarget 44, c.GetPlayers; %v", err)
					}
				}
				if countdown == 0 || len(playerList) != 0 {
					arePlayers = true
					break
				}
			}

			termui.Clear()

			if !arePlayers {
				noPlayersMsg := widgets.NewParagraph()
				noPlayersMsg.Text = "No players available, Connection lost"
				noPlayersMsg.TextStyle = termui.NewStyle(termui.ColorYellow)
				noPlayersMsg.SetRect(0, 0, 30, 3)

				termui.Render(noPlayersMsg)
				uiEvents := termui.PollEvents()
				for {
					ev := <-uiEvents
					if ev.Type == termui.KeyboardEvent && ev.ID == "<Enter>" || ev.Type == termui.KeyboardEvent && ev.ID == "<Escape>" {
						termui.Clear()
						return ""
					}
				}
			}
		}

	} else {
		arePlayers = true
	}

	playerList, err = c.GetPlayers()
	if err != nil {
		log.Fatalf("a.getTarget 45, c.GetPlayers; %v", err)
	}
	selectedPlayerIndex := 0
	selectedPlayerStyle := termui.NewStyle(termui.ColorGreen, termui.ColorBlack)

	playerNames := make([]string, len(playerList))
	for i, player := range playerList {
		playerNames[i] = player.Nick
	}

	playerListWidget := widgets.NewList()
	playerListWidget.Title = "Select a Player"
	playerListWidget.Rows = playerNames
	playerListWidget.SelectedRow = selectedPlayerIndex
	playerListWidget.SelectedRowStyle = selectedPlayerStyle

	termui.Render(playerListWidget)
	a.triggerResize(playerListWidget)

	uiEvents := termui.PollEvents()
	for {
		ev := <-uiEvents
		switch ev.Type {
		case termui.KeyboardEvent:
			switch ev.ID {
			case "<Down>":
				if selectedPlayerIndex < len(playerList)-1 {
					selectedPlayerIndex++
					playerListWidget.SelectedRow = selectedPlayerIndex
				}
			case "<Up>":
				if selectedPlayerIndex > 0 {
					selectedPlayerIndex--
					playerListWidget.SelectedRow = selectedPlayerIndex
				}
			case "<Enter>":
				selectedPlayer := playerList[selectedPlayerIndex].Nick
				termui.Clear()
				return selectedPlayer
			case "<Escape>":
				termui.Close()
				kb, err := keybd_event.NewKeyBonding()
				if err != nil {
					log.Fatalf("ui keybind 46, keybd_event.NewKeyBonding(); %v", err)
				}
				if runtime.GOOS == "linux" {
					time.Sleep(2 * time.Second)
				}

				kb.SetKeys(keybd_event.VK_C)

				kb.HasCTRL(true)

				err = kb.Launching()
				if err != nil {
					log.Fatal(err)
				}

				err = kb.Press()
				if err != nil {
					log.Fatalf("ui keypress 47, kb.Press(); %v", err)
				}
				time.Sleep(10 * time.Millisecond)
				err = kb.Release()
				if err != nil {
					log.Fatalf("ui keypress 48, kb.Release(); %v", err)
				}
				break
			}
			termui.Render(playerListWidget)
		case termui.ResizeEvent:
			payload := ev.Payload.(termui.Resize)
			termui.Clear()
			playerListWidget.SetRect(0, 0, payload.Width, payload.Height)
			termui.Render(playerListWidget)
		}
	}
}
func (a *App) waitForOpponent(waitingTime time.Duration) {
	err := termui.Init()
	if err != nil {
		return
	}
	termui.Clear()
	a.Status, err = a.Client.GetStatus()
	if err != nil {
		log.Fatalf("a.waitForOpponent 49, c.GetStatus; %v", err)
	}
	waitingMsg := widgets.NewParagraph()
	waitingMsg.TextStyle = termui.NewStyle(termui.ColorYellow)
	waitingMsg.SetRect(0, 0, 30, 3)

	termui.Render(waitingMsg)

	for a.Status.GameStatus == "waiting" || a.Status.GameStatus == "waiting_wpbot" {
		var err error
		time.Sleep(waitingTime)
		a.Status, err = a.Client.GetStatus()
		if err != nil {
			log.Fatalf("a.waitForOpponent 50, c.GetStatus; %v", err)
		}

		switch time.Now().Second() % 3 {
		case 0:
			waitingMsg.Text = "Waiting for an opponent."
		case 1:
			waitingMsg.Text = "Waiting for an opponent.."
		case 2:
			waitingMsg.Text = "Waiting for an opponent..."
		}

		termui.Render(waitingMsg)
	}

	termui.Clear()
}

func (a *App) makeUI() (*gui.GUI, error) {
	ui := gui.NewGUI(true)
	return ui, nil
}

func (a *App) buildBattlefield(ui *gui.GUI) *GuiBattle {
	guiBattle := GuiBattle{}
	guiBattle.Ui = ui

	accConfig := gui.TextConfig{BgColor: gui.Grey, FgColor: gui.Blue}
	pAcc := gui.NewText(xPBoard, yBoards-7, fmt.Sprintf("Accuracy"), &accConfig)
	guiBattle.Ui.Draw(pAcc)
	guiBattle.PlayerAccuracy = pAcc
	oAcc := gui.NewText(xOBoard, yBoards-7, fmt.Sprintf("Accuracy"), &accConfig)
	guiBattle.Ui.Draw(oAcc)
	guiBattle.OpponentAccuracy = oAcc

	exit := gui.NewText(xPBoard, yBoards-6, "To exit press CTRL+C", nil)
	guiBattle.Ui.Draw(exit)
	guiBattle.Exit = exit

	timer := gui.NewText(xPBoard, yBoards-5, fmt.Sprintf("Time: %v", a.Status.Timer), nil)
	guiBattle.Ui.Draw(timer)
	guiBattle.Timer = timer

	shouldFire := gui.NewText(xPBoard, yBoards-4, fmt.Sprintf("You should fire: %v", a.Status.ShouldFire), nil)
	shouldFire.SetBgColor(gui.Black)
	shouldFire.SetFgColor(gui.White)
	guiBattle.Ui.Draw(shouldFire)
	guiBattle.ShouldFire = shouldFire

	shotRes := gui.NewText(xPBoard, yBoards-2, "", nil)
	guiBattle.Ui.Draw(shotRes)
	guiBattle.ShotResult = shotRes

	oppShotRes := gui.NewText(xOBoard, yBoards-2, "", nil)
	guiBattle.Ui.Draw(oppShotRes)
	guiBattle.OppShotResult = oppShotRes

	pBoard := gui.NewBoard(xPBoard, yBoards, nil)
	guiBattle.Ui.Draw(pBoard)
	guiBattle.PlayerBoard = pBoard
	pStates := [10][10]gui.State{}
	for i := range pStates {
		pStates[i] = [10]gui.State{}
	}
	pBoard.SetStates(pStates)

	oBoard := gui.NewBoard(xOBoard, yBoards, nil)
	guiBattle.Ui.Draw(oBoard)
	guiBattle.OpponentBoard = oBoard
	oStates := [10][10]gui.State{}
	for i := range oStates {
		oStates[i] = [10]gui.State{}
	}
	oBoard.SetStates(oStates)

	nickConfig := gui.TextConfig{BgColor: gui.Black, FgColor: gui.White}

	pNick := gui.NewText(xPBoard, yBoards+25, fmt.Sprintf("%s", a.Nick), &nickConfig)
	guiBattle.Ui.Draw(pNick)
	guiBattle.PlayerNick = pNick
	oNick := gui.NewText(xOBoard, yBoards+25, fmt.Sprintf("%s", a.TargetNick), &nickConfig)
	guiBattle.Ui.Draw(oNick)
	guiBattle.OpponentNick = oNick

	a.formatString(a.Desc, xOBoard-1, xPBoard, yBoards+26, guiBattle.Ui)
	a.formatString(a.ODesc, xOBoard-1, xOBoard, yBoards+26, guiBattle.Ui)
	return &guiBattle
}

func (a *App) formatString(s string, n, x, y int, ui *gui.GUI) {
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

func (a *App) makeFleet(ui *gui.GUI, ctx context.Context, ctxCancel context.CancelFunc) ([]string, error) {
	cfg := gui.NewBoardConfig()
	cfg.HitChar = 'A'
	cfg.HitColor = gui.Grey
	board := gui.NewBoard(35, 1, cfg)
	states := [10][10]gui.State{}
	for i := range states {
		for j := range states[i] {
			states[i][j] = gui.Empty
		}
	}
	board.SetStates(states)
	shipCoords := make([]string, 0)
	ui.Draw(board)

	go func() {
		for len(shipCoords) != 20 {
			char := board.Listen(ctx)
			x, y, err := a.stringCoordToInt(char)
			if err != nil {
				log.Fatal(err)
			}
			if states[x][y-1] != gui.Ship {
				states[x][y-1] = gui.Ship
				shipCoords = append(shipCoords, char)
			} else {
				states[x][y-1] = gui.Empty
				for j := len(shipCoords) - 1; j >= 0; j-- {
					if shipCoords[j] == char {
						shipCoords = append(shipCoords[:j], shipCoords[j+1:]...)
						break
					}
				}
			}
			board.SetStates(states)
		}
		ui.Remove(board)
		ctxCancel()
	}()

	ui.Start(ctx, nil)

	return shipCoords, nil
}
func (a *App) checkShips(coords []string) bool {
	shipCount := map[int]int{
		1: 4, // One-tilers
		2: 3, // Two-tilers
		3: 2, // Three-tilers
		4: 1, // Four-tilers
	}

	coordMap := make(map[string]bool)
	for _, coord := range coords {
		coordMap[coord] = true
	}

	for size, count := range shipCount {
		missingCount := count
		for coord := range coordMap {
			x, y, err := a.stringCoordToInt(coord)
			if err != nil {
				log.Fatalf("a.checkShips: stringCoordToInt; %v", err)
			}
			if a.isValidShip(coordMap, size, x, y) {
				missingCount--
				if missingCount == 0 {
					break
				}
			}
		}
		if missingCount > 0 {
			return false
		}
	}

	return true
}

func (a *App) isValidShip(coordMap map[string]bool, size, x, y int) bool {
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			if a.isShipInOrientation(coordMap, size, x, y, dx, dy) {
				return true
			}
		}
	}
	return false
}

func (a *App) isShipInOrientation(coordMap map[string]bool, size, x, y, dx, dy int) bool {
	for i := 0; i < size; i++ {
		newX := x + (i * dx)
		newY := y + (i * dy)
		if newX < 0 || newX >= 10 || newY < 1 || newY > 10 {
			continue
		}
		coord, err := a.intCoordToString(newX, newY)
		if err != nil {
			log.Fatalf("a.isShipInOrientation: intCoordToString; %v", err)
		}
		if _, ok := coordMap[coord]; !ok {
			return false
		}
	}
	return true
}
