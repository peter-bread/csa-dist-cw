package stubs

import "time"

var RunTurns = "GolOperations.ProcessTurns"
var ReturnAlive = "GolOperations.TickerInstant"

type Response struct {
	World [][]byte
}

type Request struct {
	Turns  int
	Height int
	Width  int
	World  [][]byte
}

type TickerResponse struct {
	CompletedTurns int
	CellsCount     int
}

type TickerRequest struct {
	TickerChan <-chan time.Time
	Height     int
	Width      int
	World      [][]byte
}
