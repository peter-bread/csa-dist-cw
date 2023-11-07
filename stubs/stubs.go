package stubs

var RunTurns = "GolOperations.ProcessTurns"

type GolParams struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

type Response struct {
	World [][]byte
}

type Request struct {
	Params GolParams
	World  [][]byte
}
