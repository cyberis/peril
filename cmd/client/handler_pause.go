package main

import (
	"fmt"

	"github.com/cyberis/peril/internal/gamelogic"
	"github.com/cyberis/peril/internal/routing"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}
