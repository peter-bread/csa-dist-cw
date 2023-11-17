package stubs

var (
	RunGame         = "GolOperations.RunGame"
	AliveCellsCount = "GolOperations.AliveCellsCount"
	Screenshot      = "GolOperations.Screenshot"
	Quit            = "GolOperations.Quit"
	Shutdown        = "GolOperations.CloseServer"
	Pause           = "GolOperations.Pause"
	Restart         = "GolOperations.Restart"
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

type CloseServerRequest struct{}

type CloseServerResponse struct {
	Turn  int
	World [][]byte
}

type PauseRequest struct{}

type PauseResponse struct {
	Turn int
}

type RestartRequest struct{}

type RestartResponse struct {
	Turn int
}
