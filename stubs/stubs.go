package stubs

var (
	RunGame         = "GolOperations.RunGame"
	AliveCellsCount = "GolOperations.AliveCellsCount"
)

type RunGameResponse struct {
	World [][]byte
}

type RunGameRequest struct {
	Turns  int
	Height int
	Width  int
	World  [][]byte
}

type AliveCellsCountResponse struct {
	CompletedTurns int
	CellsCount     int
}

type AliveCellsCountRequest struct {
	Height int
	Width  int
}
