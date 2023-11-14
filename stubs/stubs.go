package stubs

var (
	RunGame         = "GolOperations.RunGame"
	AliveCellsCount = "GolOperations.AliveCellsCount"
	Screenshot      = "GolOperations.Screenshot"
)

type RunGameRequest struct {
	Turns  int
	Height int
	Width  int
	World  [][]byte
}

type RunGameResponse struct {
	World [][]byte
}

type AliveCellsCountRequest struct {
	Height int
	Width  int
}

type AliveCellsCountResponse struct {
	CompletedTurns int
	CellsCount     int
}

type ScreenshotRequest struct{}

type ScreenshotResponse struct {
	World [][]byte
}
