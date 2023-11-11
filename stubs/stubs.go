package stubs

var RunGame = "GolOperations.RunGame"

type RunGameResponse struct {
	World [][]byte
}

type RunGameRequest struct {
	Turns  int
	Height int
	Width  int
	World  [][]byte
}

type AliveCellCountResponse struct {
}

type AliveCellCountRequest struct {
}
