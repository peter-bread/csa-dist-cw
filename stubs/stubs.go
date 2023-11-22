package stubs

var (
	RunGame         = "Broker.RunGame"
	AliveCellsCount = "Broker.AliveCellsCount"
	Screenshot      = "Broker.Screenshot"
	Quit            = "Broker.Quit"
	CloseBroker     = "Broker.CloseBroker"
	Pause           = "Broker.Pause"
	Restart         = "Broker.Restart"
	NextState       = "Server.ReturnNextState"
	CloseServer     = "Server.CloseServer"
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

type AliveCellsCountRequest struct{}

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

type NextStateRequest struct {
	// Height     int
	// Width      int
	// WholeWorld [][]byte
	// WorldSlice [][]byte // for when it is only operating on a slice, World will be for comparing (as in parallel implementation)
	StartY      int
	EndY        int
	StartX      int
	EndX        int
	WorldHeight int
	WorldWidth  int
	World       [][]byte
}

type NextStateResponse struct {
	World [][]byte
}

type CloseServerRequest struct{}

type CloseServerResponse struct{}
