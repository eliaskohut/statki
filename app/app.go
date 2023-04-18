package app

import (
	"fmt"
	"log"
	"main/client"

	"github.com/fatih/color"
	board "github.com/grupawp/warships-lightgui"
)

func Start() {
	token, game, layout, status, err := client.InitGame()
	if err != nil {
		log.Fatal(err)
	}

	board := board.New(
		board.ConfigParams().
			HitChar('#').
			HitColor(color.FgRed).
			BorderColor(color.BgRed).
			RulerTextColor(color.BgYellow).
			NewConfig())

	for _, i := range layout {
		board.Set(0, i, 4)
	}

	board.Display()

	fmt.Println(token)
	fmt.Println(game)
	fmt.Println(layout)
	fmt.Println(status)
}
