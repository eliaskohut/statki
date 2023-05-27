package app

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	waitingTime    = time.Second / 3
	squarePickTime = time.Millisecond * 10
	missDelayTime  = time.Millisecond * 100
	hitRes         = "hit"
	missRes        = "miss"
	sunkRes        = "sunk"
	blankRes       = ""
	clearCmd       = "clear"
)

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

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func nicknameCheck(nickname string) bool {
	if len(nickname) < 2 || len(nickname) > 10 {
		return false
	}
	return true
}
