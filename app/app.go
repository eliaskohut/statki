package app

import (
	"errors"
	"fmt"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	gui "github.com/grupawp/warships-gui/v2"
	"log"
	"main/client"
	"strconv"
	"strings"
	"time"
)

const (
	waitingTime     = time.Second / 3
	squarePickTime  = time.Millisecond * 10
	missDelayTime   = time.Millisecond * 100
	refreshInterval = 1 * time.Second
	maxWaitTime     = 30 * time.Second
	countdownTime   = 30
	hitRes          = "hit"
	missRes         = "miss"
	sunkRes         = "sunk"
	blankRes        = ""
	clearCmd        = "clear"
)

type App struct {
	Client        *client.Client
	PlayerBoard   [10][10]gui.State
	OpponentBoard [10][10]gui.State
	Nick          string
	TargetNick    string
	Status        client.StatusResponse
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
	PlayerBoard      *gui.Board
	OpponentBoard    *gui.Board
	PlayerNick       *gui.Text
	OpponentNick     *gui.Text
	PlayerDesc       *gui.Text
	OpponentDesc     *gui.Text
	PlayerAccuracy   *gui.Text
	OpponentAccuracy *gui.Text
	ShouldFire       *gui.Text
	Exit             *gui.Text
	Timer            *gui.Text
	ShotResult       *gui.Text
	Ui               *gui.GUI
}

func (a *App) Start() {
	c := client.NewClient()
	game, err := a.getDetails(c)
	if err != nil {
		log.Fatal(err)
	}

	a.Status, err = c.GetStatus()
	if err != nil {
		log.Fatal(err)
	}

	if a.Status.GameStatus == "waiting" {
		a.waitForOpponent(c, waitingTime)
	}

	fmt.Println(game)
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

// mapping maps layout from a slice of strings to a format which can be read by gui
func (a *App) mapping(layout []string) ([][]int, error) {
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

func (a *App) contains(s []string, e string) bool {
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
	nameInput.TextStyle = ui.NewStyle(ui.ColorYellow)
	nameInput.SetRect(0, 0, 80, 3)

	errorMsg := widgets.NewParagraph()
	errorMsg.TextStyle = ui.NewStyle(ui.ColorRed)
	errorMsg.SetRect(0, 4, 50, 7)

	ui.Render(nameInput, errorMsg)

	name := ""
	nameEvents := ui.PollEvents()
	for {
		nameEv := <-nameEvents
		switch nameEv.Type {
		case ui.KeyboardEvent:
			switch nameEv.ID {
			case "<Enter>":
				playerName = strings.TrimSpace(name)
				if playerName == "" {
					return ""
				} else if a.validateName(playerName) {
					return playerName
				} else {
					ui.Render(nameInput) // Clear error message from the screen
					if len(playerName) < 2 || len(playerName) > 10 {
						errorMsg.Text = "Please enter a name between 2 and 10 characters."
					} else {
						errorMsg.Text = "Name cannot contain symbols.\nPlease enter a valid name."
					}
					ui.Render(errorMsg)
				}
			case "<Escape>":
				fmt.Println("The user pressed exit")
				return ""
			case "<Backspace>":
				if len(name) > 0 {
					name = name[:len(name)-1]
					nameInput.Text = name
					ui.Render(nameInput)
				}
			default:
				// Accumulate the characters in the name variable
				if len(nameEv.ID) == 1 && len(name) < 10 {
					name += nameEv.ID
					nameInput.Text = name
					ui.Render(nameInput)
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
	ui.Clear()
	width, height := ui.TerminalDimensions()
	if list != nil {
		list.SetRect(0, 0, width, height)
	}
	ui.Render(list)
}

func (a *App) getPlayerDescription() string {
	descInput := widgets.NewParagraph()
	descInput.Title = "Enter Your Description"
	descInput.TextStyle = ui.NewStyle(ui.ColorYellow)
	descInput.SetRect(0, 0, 50, 10)

	errorMsg := widgets.NewParagraph()
	errorMsg.TextStyle = ui.NewStyle(ui.ColorRed)
	errorMsg.SetRect(0, 12, 50, 15)

	ui.Render(descInput, errorMsg)

	description := ""
	descEvents := ui.PollEvents()
	for {
		descEv := <-descEvents
		switch descEv.Type {
		case ui.KeyboardEvent:
			switch descEv.ID {
			case "<Enter>":
				description = strings.TrimSpace(description)
				if a.validateDescription(description) {
					return description
				} else {
					ui.Render(descInput)
					if len(description) < 5 || len(description) > 200 {
						errorMsg.Text = "Please enter a description between 5 and 200 characters."
					}
					ui.Render(errorMsg)
				}
			case "<Escape>":
				fmt.Println("The user pressed exit")
				return ""
			case "<Backspace>":
				if len(description) > 0 {
					// Remove the last character from the description
					description = description[:len(description)-1]
					descInput.Text = description
					ui.Render(descInput)
				}
			default:
				// Accumulate the characters in the description variable
				if len(descEv.ID) == 1 && len(description) < 200 {
					description += descEv.ID
					descInput.Text = description
					ui.Render(descInput)
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
	if err := ui.Init(); err != nil {
		log.Fatalf("Failed to initialize termui: %v", err)
	}

	options := []string{"Play with a bot", "Wait for an opponent", "Challenge someone"}

	list := widgets.NewList()
	list.Title = "Choose an Option"
	list.Rows = options
	list.SelectedRowStyle = ui.NewStyle(ui.ColorGreen, ui.ColorBlack)

	ui.Render(list)
	a.triggerResize(list)

	uiEvents := ui.PollEvents()
	for {
		ev := <-uiEvents
		switch ev.Type {
		case ui.KeyboardEvent:
			switch ev.ID {
			case "<Down>":
				list.ScrollDown()
			case "<Up>":
				list.ScrollUp()
			case "<Enter>":
				selectedOption := options[list.SelectedRow]

				if selectedOption == "Play with a bot" {
					ui.Clear()
					nick = a.getPlayerName()
					if nick != "" {
						ui.Clear()
						pDes = a.getPlayerDescription()
						game, err := c.InitGame(client.Game{Nick: nick, Desc: pDes, WPBot: true})
						if err != nil {
							log.Fatal(err)
						}
						ui.Close()
						return game, nil
					} else {
						game, err := c.InitGame(client.Game{WPBot: true})
						if err != nil {
							log.Fatal(err)
						}
						ui.Close()
						return game, nil
					}
				} else if selectedOption == "Wait for an opponent" {
					ui.Clear()
					nick = a.getPlayerName()
					if nick != "" {
						ui.Clear()
						pDes = a.getPlayerDescription()
						game, err := c.InitGame(client.Game{Nick: nick, Desc: pDes, WPBot: false})
						if err != nil {
							log.Fatal(err)
						}
						return game, nil
					} else {
						game, err := c.InitGame(client.Game{WPBot: false})
						if err != nil {
							log.Fatal(err)
						}
						return game, nil
					}
				} else if selectedOption == "Challenge someone" {
					ui.Clear()
					nick = a.getPlayerName()
					if nick != "" {
						ui.Clear()
						pDes = a.getPlayerDescription()
						ui.Clear()
						targetNick := a.getTarget(c)
						if targetNick == "" {
							log.Fatalf("Target nick cannot be empty")
						}
						game, err := c.InitGame(client.Game{Nick: nick, Desc: pDes, WPBot: false, TargetNick: targetNick})
						if err != nil {
							log.Fatal(err)
						}
						ui.Close()
						return game, nil
					} else {
						ui.Clear()
						targetNick := a.getTarget(c)
						if targetNick == "" {
							log.Fatalf("Target nick cannot be empty")
						}
						game, err := c.InitGame(client.Game{WPBot: false, TargetNick: targetNick})
						if err != nil {
							log.Fatal(err)
						}
						ui.Close()
						return game, nil
					}
				}
			case "<Escape>":
				fmt.Println("The user pressed exit")
				return game, nil
			}
			ui.Render(list)
		case ui.ResizeEvent:
			payload := ev.Payload.(ui.Resize)
			ui.Clear()
			list.SetRect(0, 0, payload.Width, payload.Height)
			ui.Render(list)
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
			timerMsg.TextStyle = ui.NewStyle(ui.ColorYellow)
			timerMsg.SetRect(0, 0, 50, 3)

			ui.Render(timerMsg)

			countdown := countdownTime
			ticker := time.NewTicker(time.Second * 1)
			defer ticker.Stop()

			for range ticker.C {
				countdown--
				timerMsg.Text = fmt.Sprintf("No players available - Waiting %d seconds", countdown)
				ui.Render(timerMsg)
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

			ui.Clear()

			if !arePlayers {
				noPlayersMsg := widgets.NewParagraph()
				noPlayersMsg.Text = "No players available, Connection lost"
				noPlayersMsg.TextStyle = ui.NewStyle(ui.ColorYellow)
				noPlayersMsg.SetRect(0, 0, 30, 3)

				ui.Render(noPlayersMsg)
				uiEvents := ui.PollEvents()
				for {
					ev := <-uiEvents
					if ev.Type == ui.KeyboardEvent && ev.ID == "<Enter>" || ev.Type == ui.KeyboardEvent && ev.ID == "<Escape>" {
						ui.Clear()
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
	selectedPlayerStyle := ui.NewStyle(ui.ColorGreen, ui.ColorBlack)

	playerNames := make([]string, len(playerList))
	for i, player := range playerList {
		playerNames[i] = player.Nick
	}

	playerListWidget := widgets.NewList()
	playerListWidget.Title = "Select a Player"
	playerListWidget.Rows = playerNames
	playerListWidget.SelectedRow = selectedPlayerIndex
	playerListWidget.SelectedRowStyle = selectedPlayerStyle

	ui.Render(playerListWidget)
	a.triggerResize(playerListWidget)

	uiEvents := ui.PollEvents()
	for {
		ev := <-uiEvents
		switch ev.Type {
		case ui.KeyboardEvent:
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
				ui.Clear()
				return selectedPlayer
			case "<Escape>":
				ui.Clear()
				return ""
			}
			ui.Render(playerListWidget)
		case ui.ResizeEvent:
			payload := ev.Payload.(ui.Resize)
			ui.Clear()
			playerListWidget.SetRect(0, 0, payload.Width, payload.Height)
			ui.Render(playerListWidget)
		}
	}
}
func (a *App) waitForOpponent(c *client.Client, waitingTime time.Duration) {
	ui.Clear()
	waitingMsg := widgets.NewParagraph()
	waitingMsg.Text = "Waiting for an opponent."
	waitingMsg.TextStyle = ui.NewStyle(ui.ColorYellow)
	waitingMsg.SetRect(0, 0, 30, 3)

	ui.Render(waitingMsg)

	for a.Status.GameStatus == "waiting" {
		var err error
		time.Sleep(waitingTime * 5)
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

		ui.Render(waitingMsg)
	}

	ui.Clear()
	ui.Close()
}
