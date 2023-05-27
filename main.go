package main

import "main/app"

var (
	selectedOption string
)

func main() {
	game := app.App{}
	game.Start()
}
