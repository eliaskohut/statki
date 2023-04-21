package app

import (
	"fmt"
	"log"
	"main/client"
	"time"

	board "github.com/grupawp/warships-lightgui"
)

func Start() {
	token, game, layout, err := client.InitGame(true)
	if err != nil {
		log.Fatal(err)
	}
	status, err := client.Status(token)
	if err != nil {
		log.Fatal(err)
	}
	for {
		if status.GameStatus != "waiting" {
			break
		}
		fmt.Println("Waiting for an opponent...")
		time.Sleep(1 * time.Second)
	}

	board := board.New(board.NewConfig())

	board.Import(layout)
	board.Display()
	fmt.Printf("Me: %s \n", status.Nick)
	fmt.Printf("Player2: %s \n", status.Opponent)
	fmt.Printf("Opponent description: %s \n", status.OppDesc)

	// if(status.ShouldFire==true){

	// }

	fmt.Println(game)
	fmt.Println(status)
	fmt.Println(client.Fire(0, layout[0], token, board))
}
