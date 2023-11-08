package stubs

var RunTurns = "GolOperations.ProcessTurns"

type Response struct {
	World [][]byte
}

type Request struct {
	Turns  int
	Height int
	Width  int
	World  [][]byte
}
