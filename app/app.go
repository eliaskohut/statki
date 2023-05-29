package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	gui "github.com/grupawp/warships-gui/v2"
	"log"
	"main/client"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	waitingTime    = time.Second / 3
	squarePickTime = time.Millisecond * 10
	countdownTime  = 30
	hitRes         = "hit"
	missRes        = "miss"
	sunkRes        = "sunk"
	blankRes       = ""
	fourTileCount  = 1
	threeTileCount = 2
	twoTileCount   = 3
	oneTileCount   = 4
	yBoards        = 8
	xPBoard        = 1
	xOBoard        = 55
)

var (
	ctx = context.TODO()
)

type App struct {
	Client      *client.Client
	PlayerBoard []string
	Nick        string
	TargetNick  string
	Status      client.StatusResponse
	Desc        string
	ODesc       string
}

type GuiConstruct struct {
	Board     *gui.Board
	FourTile  *gui.Text
	ThreeTile *gui.Text
	TwoTile   *gui.Text
	OneTile   *gui.Text
	Ui        *gui.GUI
}

type GuiBattle struct {
	PlayerBoard         *gui.Board
	PlayerBoardStates   [10][10]gui.State
	OpponentBoard       *gui.Board
	OpponentBoardStates [10][10]gui.State
	PlayerNick          *gui.Text
	OpponentNick        *gui.Text
	PlayerAccuracy      *gui.Text
	OpponentAccuracy    *gui.Text
	ShouldFire          *gui.Text
	Exit                *gui.Text
	Timer               *gui.Text
	ShotResult          *gui.Text
	Ui                  *gui.GUI
}

func (a *App) Start() {
	for {
		time.Sleep(time.Second * 2)
		a.Client = client.NewClient()
		game, err := a.getDetails(a.Client)
		if err != nil {
			log.Fatal(err)
		}

		if a.Client.Token != "" {
			a.Status, err = a.Client.GetStatus()
			if err != nil {
				log.Fatal(err)
			}

			gameDesc, err := a.Client.GetDescription()
			if err != nil {
				log.Fatal(err)
			}
			a.Nick = gameDesc.Nick
			a.Desc = gameDesc.Desc
			a.TargetNick = gameDesc.Opponent
			a.ODesc = gameDesc.OppDesc

			ui, err := a.makeUI()
			if err != nil {
				return
			}

			guiBattle := a.buildBattlefield(ui)

			ans, err := a.Client.GetBoard()
			if err != nil {
				log.Fatal(err)
			}
			layout := ans.Board

			a.PlayerBoard = layout

			var mapped [][]int
			mapped, err = a.mappingChars(layout)
			if err != nil {
				log.Fatal(err)
				return
			}
			states := [10][10]gui.State{}
			for _, i := range mapped {
				states[i[0]][i[1]-1] = gui.Ship
			}
			guiBattle.PlayerBoardStates = states
			guiBattle.PlayerBoard.SetStates(states)

			go a.startBattle(guiBattle)

			guiBattle.Ui.Start(ctx, nil)

			fmt.Println(game)
		} else {
			fmt.Println("Successfully finished")
			break
		}

	}
}

func (a *App) timerUpdate(guiB *GuiBattle) {
	var winner string
	quitChan := make(chan bool)
	for a.Status.GameStatus != "ended" {
		time.Sleep(time.Millisecond * 200)
		var err error
		a.Status, err = a.Client.GetStatus()
		if err != nil {
			log.Fatal(err)
		}
		guiB.Timer.SetText(fmt.Sprintf("Time: %v", a.Status.Timer))
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
	guiB.Ui.Remove(guiB.ShouldFire)
	guiB.Ui.Remove(guiB.Timer)
	guiB.Ui.Remove(guiB.OpponentAccuracy)
	guiB.Ui.Draw(gui.NewText(xPBoard, yBoards, fmt.Sprintf("Winner: %s", winner), nil))
	guiB.Exit.SetText("To start a new game press CTRL+C")
	quitChan <- true
}

func (a *App) startBattle(guiB *GuiBattle) {
	var winner string
	var err error
	var acc float64
	var oppAcc float64
	shots := make([]string, 0)
	hitShots := make([]string, 0)
	oppHitShots := make([]string, 0)
	quitChan := make(chan bool)
	go a.timerUpdate(guiB)
	a.Status, err = a.Client.GetStatus()
	if err != nil {
		log.Fatal(err)
	}
	for a.Status.GameStatus != "ended" {
		a.Status, err = a.Client.GetStatus()
		if err != nil {
			log.Fatal(err)
		}
		if a.Status.ShouldFire {
			for _, shot := range a.Status.OppShots {
				x, y, err := a.stringCoordToInt(shot)
				if err != nil {
					log.Fatal(err)
					return
				}
				if a.contains(shot, a.PlayerBoard) {
					guiB.PlayerBoardStates[x][y-1] = gui.Hit
					oppHitShots = append(oppHitShots, shot)
				} else {
					guiB.PlayerBoardStates[x][y-1] = gui.Miss
				}
			}
			if len(oppHitShots) != 0 {
				oppAcc = float64(len(oppHitShots) / len(a.Status.OppShots))
				if oppAcc != math.NaN() {
					guiB.OpponentAccuracy.SetText(fmt.Sprintf("Accuracy: %v", oppAcc))
				} else {
					guiB.OpponentAccuracy.SetText(fmt.Sprintf("Accuracy: %v", 0))
				}
			} else {
				guiB.PlayerAccuracy.SetText(fmt.Sprintf("Accuracy: %v", 0))
			}
			guiB.PlayerBoard.SetStates(guiB.PlayerBoardStates)
			guiB.ShouldFire.SetText("Fire!")
			guiB.ShouldFire.SetFgColor(gui.Green)
			char := guiB.OpponentBoard.Listen(ctx)
			x, y, err := a.stringCoordToInt(char)
			if err != nil {
				log.Fatal(err)
				return
			}
			for guiB.OpponentBoardStates[x][y-1] == gui.Miss || guiB.OpponentBoardStates[x][y-1] == gui.Hit {
				guiB.ShouldFire.SetText("You can't fire there!")
				char = guiB.OpponentBoard.Listen(ctx)
				x, y, err = a.stringCoordToInt(char)
				if err != nil {
					log.Fatal(err)
					return
				}
			}
			shots = append(shots, char)
			result, err := a.Client.Shoot(char)
			if err != nil {
				log.Fatal(err)
				return
			}
			if result == hitRes || result == sunkRes {
				hitShots = append(hitShots, char)
				guiB.OpponentBoardStates[x][y-1] = gui.Hit
			} else if result == blankRes {
				continue
			} else if result == missRes {
				guiB.OpponentBoardStates[x][y-1] = gui.Miss
			}
			guiB.OpponentBoard.SetStates(guiB.OpponentBoardStates)
			guiB.ShotResult.SetText(fmt.Sprintf("%s on %s", result, char))
			if len(hitShots) != 0 {
				acc = float64(len(hitShots) / len(shots))
				if acc != math.NaN() {
					guiB.PlayerAccuracy.SetText(fmt.Sprintf("Accuracy: %v", acc))
				} else {
					guiB.PlayerAccuracy.SetText(fmt.Sprintf("Accuracy: %v", 0))
				}
			} else {
				guiB.PlayerAccuracy.SetText(fmt.Sprintf("Accuracy: %v", 0))
			}
		} else {
			guiB.ShouldFire.SetText("It's not your turn!")
			guiB.ShouldFire.SetFgColor(gui.Red)
			a.Status, err = a.Client.GetStatus()
			if err != nil {
				log.Fatal(err)
			}
		}
		a.Status, err = a.Client.GetStatus()
		if err != nil {
			log.Fatal(err)
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
	guiB.Ui.Remove(guiB.ShouldFire)
	guiB.Ui.Remove(guiB.Timer)
	guiB.Ui.Remove(guiB.OpponentAccuracy)
	guiB.Ui.Draw(gui.NewText(xPBoard, yBoards, fmt.Sprintf("Winner: %s", winner), nil))
	guiB.Exit.SetText("To start a new game press CTRL+C")
	quitChan <- true
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
			log.Fatal(err)
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
				fmt.Println("The user pressed exit")
				termui.Close()
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
				fmt.Println("The user pressed exit")
				return ""
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

func (a *App) getDetails(c *client.Client) (client.Game, error) {

	var nick string
	var pDes string
	game := client.Game{}
	if err := termui.Init(); err != nil {
		log.Fatalf("Failed to initialize termui: %v", err)
	}

	options := []string{"Play with a bot", "Wait for an opponent", "Challenge someone", "Show stats"}

	list := widgets.NewList()
	list.Title = "Choose an Option"
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

				if selectedOption == "Play with a bot" {
					termui.Clear()
					nick = a.getPlayerName()
					if nick != "" {
						termui.Clear()
						pDes = a.getPlayerDescription()
						game, err := c.InitGame(client.Game{Nick: nick, Desc: pDes, WPBot: true})
						if err != nil {
							log.Fatal(err)
						}
						a.waitForOpponent(c, waitingTime)
						termui.Close()
						return game, nil
					} else {
						game, err := c.InitGame(client.Game{WPBot: true})
						if err != nil {
							log.Fatal(err)
						}
						a.waitForOpponent(c, waitingTime)
						termui.Close()
						return game, nil
					}
				} else if selectedOption == "Wait for an opponent" {
					termui.Clear()
					nick = a.getPlayerName()
					if nick != "" {
						termui.Clear()
						pDes = a.getPlayerDescription()
						game, err := c.InitGame(client.Game{Nick: nick, Desc: pDes, WPBot: false})
						if err != nil {
							log.Fatal(err)
						}
						a.waitForOpponent(c, waitingTime)
						termui.Close()
						return game, nil
					} else {
						game, err := c.InitGame(client.Game{WPBot: false})
						if err != nil {
							log.Fatal(err)
						}
						a.waitForOpponent(c, waitingTime)
						termui.Close()
						return game, nil
					}
				} else if selectedOption == "Challenge someone" {
					termui.Clear()
					nick = a.getPlayerName()
					if nick != "" {
						termui.Clear()
						pDes = a.getPlayerDescription()
						termui.Clear()
						targetNick := a.getTarget(c)
						if targetNick == "" {
							log.Fatalf("Target nick cannot be empty")
						}
						game, err := c.InitGame(client.Game{Nick: nick, Desc: pDes, WPBot: false, TargetNick: targetNick})
						if err != nil {
							log.Fatal(err)
						}
						a.waitForOpponent(c, waitingTime)
						termui.Close()
						return game, nil
					} else {
						termui.Clear()
						targetNick := a.getTarget(c)
						if targetNick == "" {
							log.Fatalf("Target nick cannot be empty")
						}
						game, err := c.InitGame(client.Game{WPBot: false, TargetNick: targetNick})
						if err != nil {
							log.Fatal(err)
						}
						a.waitForOpponent(c, waitingTime)
						termui.Close()
						return game, nil
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
					list := widgets.NewList()
					list.Rows = stringStats
					list.SelectedRowStyle = termui.NewStyle(termui.ColorGreen, termui.ColorBlack)
					termui.Render(list)
					a.triggerResize(list)
					return game, nil
				}
			case "<Escape>":
				termui.Close()
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
func (a *App) getTarget(c *client.Client) string {
	playerList, err := c.GetPlayers()
	if err != nil {
		log.Fatal(err)
	}
	arePlayers := false
	if len(playerList) == 0 {
		time.Sleep(waitingTime * 3)
		playerList, err := c.GetPlayers()
		if err != nil {
			log.Fatal(err)
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
						log.Fatal(err)
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
		log.Fatal(err)
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
				termui.Clear()
				termui.Close()
				return ""
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
func (a *App) waitForOpponent(c *client.Client, waitingTime time.Duration) {
	termui.Clear()
	var err error
	a.Status, err = a.Client.GetStatus()
	if err != nil {
		log.Fatal(err)
	}
	waitingMsg := widgets.NewParagraph()
	waitingMsg.TextStyle = termui.NewStyle(termui.ColorYellow)
	waitingMsg.SetRect(0, 0, 30, 3)

	termui.Render(waitingMsg)

	for a.Status.GameStatus == "waiting" || a.Status.GameStatus == "waiting_wpbot" {
		var err error
		time.Sleep(waitingTime)
		a.Status, err = c.GetStatus()
		if err != nil {
			log.Fatal(err)
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

func (a *App) makeFleet(ui *gui.GUI) ([]string, error) {
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

	fourTile := fourTileCount
	threeTile := threeTileCount
	twoTile := twoTileCount
	oneTile := oneTileCount

	fourTileTxt := gui.NewText(5, 1, fmt.Sprintf("4 tiles left: %d", fourTile), nil)
	threeTileTxt := gui.NewText(5, 3, fmt.Sprintf("3 tiles left: %d", threeTile), nil)
	twoTileTxt := gui.NewText(5, 5, fmt.Sprintf("2 tiles left: %d", twoTile), nil)
	oneTileTxt := gui.NewText(5, 7, fmt.Sprintf("1 tile left: %d", oneTile), nil)

	ui.Draw(fourTileTxt)
	ui.Draw(threeTileTxt)
	ui.Draw(twoTileTxt)
	ui.Draw(oneTileTxt)

	ui.Draw(board)

	var shipCoords [][]int // Slice to store ship coordinates
	hasHitTiles := false

	go func(hasHitTiles *bool) {
		for {
			for _, row := range states {
				for _, state := range row {
					if state == gui.Hit {
						*hasHitTiles = true
						break
					}
				}
				break
			}
			char := board.Listen(ctx)
			x, y, err := a.stringCoordToInt(char)
			if err != nil {
				log.Fatal(err)
			}

			if !*hasHitTiles {
				states = a.markAllowed(x, y, states, hasHitTiles)
				board.SetStates(states)
				continue
			} else if *hasHitTiles {
				if states[x][y-1] == gui.Hit {
					states = a.markAllowed(x, y, states, hasHitTiles)
					board.SetStates(states)
					continue
				} else if states[x][y-1] == gui.Ship {
					states = a.markAllowed(x, y, states, hasHitTiles)
					board.SetStates(states)
					continue
				} else if states[x][y-1] == gui.Empty {
					continue
				}
			}
		}
	}(&hasHitTiles)

	ui.Start(ctx, nil)

	// Convert shipCoords to slice of strings in the format "A1", "G7"
	shipCoordsStr, err := a.mappingInts(shipCoords)
	if err != nil {
		return nil, err
	}

	return shipCoordsStr, nil
}

func (a *App) markAllowed(x, y int, states [10][10]gui.State, hasHitTiles *bool) [10][10]gui.State {
	if states[x][y-1] == gui.Empty && !*hasHitTiles {
		if y != 1 && states[x][y-2] != gui.Ship {
			states[x][y-2] = gui.Hit
		}
		if x != 0 && states[x-1][y-1] != gui.Ship {
			states[x-1][y-1] = gui.Hit
		}
		if x != 9 && states[x+1][y-1] != gui.Ship {
			states[x+1][y-1] = gui.Hit
		}
		if y != 10 && states[x][y] != gui.Ship {
			states[x][y] = gui.Hit
		}
		states[x][y-1] = gui.Ship
	} else if states[x][y-1] == gui.Ship {
		if y != 1 && states[x][y-2] != gui.Ship {
			states[x][y-2] = gui.Empty
		}
		if x != 0 && states[x-1][y-1] != gui.Ship {
			states[x-1][y-1] = gui.Empty
		}
		if x != 9 && states[x+1][y-1] != gui.Ship {
			states[x+1][y-1] = gui.Empty
		}
		if y != 10 && states[x][y] != gui.Ship {
			states[x][y] = gui.Empty
		}
		states[x][y-1] = gui.Empty
		hasShipTiles := false
		for _, row := range states {
			for _, state := range row {
				if state == gui.Ship {
					hasShipTiles = true
					break
				}
			}
			break
		}
		if hasShipTiles {
			states[x][y-1] = gui.Hit
		}
	} else if states[x][y-1] == gui.Hit {
		if y != 1 && states[x][y-2] != gui.Ship {
			states[x][y-2] = gui.Hit
		}
		if x != 0 && states[x-1][y-1] != gui.Ship {
			states[x-1][y-1] = gui.Hit
		}
		if x != 9 && states[x+1][y-1] != gui.Ship {
			states[x+1][y-1] = gui.Hit
		}
		if y != 10 && states[x][y] != gui.Ship {
			states[x][y] = gui.Hit
		}
		states[x][y-1] = gui.Ship
	}

	return states
}
