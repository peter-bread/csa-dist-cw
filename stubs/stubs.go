package stubs

import "uk.ac.bris.cs/gameoflife/util"

var (
	ReadyToDial     = "Broker.ReadyToDial"
	RunGame         = "Broker.RunGame"
	AliveCellsCount = "Broker.AliveCellsCount"
	Screenshot      = "Broker.Screenshot"
	Quit            = "Broker.Quit"
	CloseBroker     = "Broker.CloseBroker"
	Pause           = "Broker.Pause"
	Restart         = "Broker.Restart"
	NextState       = "Server.ReturnNextState"
	CloseServer     = "Server.CloseServer"
	SendWorldState  = "Controller.SendWorldState"
)

type ReadyToDialRequest struct {
	S    string
	Port string
}

type ReadyToDialResponse struct {
	S string
}

type SendWorldStateRequest struct {
	CellsFlipped   []util.Cell
	CompletedTurns int
	CellsCount     int
}

type SendWorldStateResponse struct{}

type RunGameRequest struct {
	Turns   int
	Height  int
	Width   int
	Threads int
	World   [][]byte
}

type RunGameResponse struct {
	World          [][]byte
	AliveCells     []util.Cell
	CompletedTurns int
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

type QuitResponse struct{}

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
	StartY      int
	EndY        int
	StartX      int
	EndX        int
	WorldHeight int
	WorldWidth  int
	Threads     int
	World       [][]byte
}

type NextStateResponse struct {
	World [][]byte
}

type CloseServerRequest struct{}

type CloseServerResponse struct{}
