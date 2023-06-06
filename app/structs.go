package app

import (
	gui "github.com/grupawp/warships-gui/v2"
	"main/client"
)

type App struct {
	Client      *client.Client
	PlayerBoard []string
	Nick        string
	TargetNick  string
	Status      client.StatusResponse
	Desc        string
	ODesc       string
	Ui          *gui.GUI
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
	OppShotResult       *gui.Text
	Ui                  *gui.GUI
}
