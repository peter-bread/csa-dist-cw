package stubs

var (
	RunGame         = "Broker.RunGame"
	AliveCellsCount = "Broker.AliveCellsCount"
	Screenshot      = "Broker.Screenshot"
	Quit            = "Broker.Quit"
	Shutdown        = "Broker.CloseServer"
	Pause           = "Broker.Pause"
	Restart         = "Broker.Restart"
)

type RunGameRequest struct {
	Turns  int
	Height int
	Width  int
	World  [][]byte
}

type RunGameResponse struct {
	World [][]byte
	Turn  int
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

type CloseBrokerRequest struct{}

type CloseBrokerResponse struct{}

type PauseRequest struct{}

type PauseResponse struct {
	Turn int
}

type RestartRequest struct{}

type RestartResponse struct {
	Turn int
}
