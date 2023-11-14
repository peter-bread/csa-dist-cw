package stubs

var (
	RunGame         = "GolOperations.RunGame"
	AliveCellsCount = "GolOperations.AliveCellsCount"
	Screenshot      = "GolOperations.Screenshot"
	Quit            = "GolOperations.Quit"
	Shutdown        = "GolOperations.Shutdown"
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

type QuitRequest struct{}

type QuitResponse struct {
	Turn int
}

type CloseServerResponse struct{}

type CloseServerRequest struct{}
