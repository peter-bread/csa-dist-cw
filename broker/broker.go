package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"strconv"
	"sync"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

var (
	world                 [][]byte
	height                int
	width                 int
	turn                  int
	threads               int
	mutex                 sync.Mutex
	closeBrokerChan       chan struct{}
	stopTurnsChan         chan struct{}
	turnExecutionFinished sync.WaitGroup
	servers               []string
	distClient            *rpc.Client
)

func makeNextStateCall(client *rpc.Client, resultChan chan<- stubs.NextStateResponse, i int) {
	mutex.Lock()
	// store copy of the world to send to server
	tempWorld := make([][]byte, height)
	for i := 0; i < height; i++ {
		tempWorld[i] = make([]byte, width)
	}
	copy(tempWorld, world)

	// copy these values while mutex is locked to prevent any race conditions
	h := height
	w := width
	t := threads
	mutex.Unlock()

	sliceHeight := h / 4 // this should always divide nicely (since we are hardcoding 4 servers and all given input files are divisible by 4)

	req := stubs.NextStateRequest{
		World:       tempWorld,
		WorldHeight: h,
		WorldWidth:  w,
		StartX:      0,
		EndX:        w,
		StartY:      sliceHeight * i,
		EndY:        sliceHeight * (i + 1),
		Threads:     t,
	}

	res := new(stubs.NextStateResponse)
	client.Call(stubs.NextState, req, res)
	resultChan <- *res
}

func makeSendWorldStateCall(world [][]byte, cellsFlipped []util.Cell, completedTurns, cellsCount int) {
	req := stubs.SendWorldStateRequest{
		CellsFlipped:   cellsFlipped,
		CompletedTurns: completedTurns,
		CellsCount:     cellsCount,
	}
	res := new(stubs.SendWorldStateResponse)
	distClient.Call(stubs.SendWorldState, req, res)
}

// define ReadyToDial that tells the broker it is safe to dial the distributor
func (g *Broker) ReadyToDial(req stubs.ReadyToDialRequest, res *stubs.ReadyToDialResponse) (err error) {
	// dial distributor
	fmt.Println(req.S)
	distributor := "127.0.0.1:8020"
	client, err := rpc.Dial("tcp", distributor)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	res.S = "broker is connected to controller"
	distClient = client
	return
}

func RunTurns(turns int, resultChan chan<- [][]byte) (err error) {
	defer turnExecutionFinished.Done()
	turn = 0

	clients := make([]*rpc.Client, 4)

	for i := 0; i < 4; i++ {
		clients[i], err = rpc.Dial("tcp", servers[i])
		if err != nil {
			log.Fatal("dialing:", err)
		}
	}

TurnsLoop:
	for ; turn < turns; turn++ {
		select {
		case <-stopTurnsChan:
			break TurnsLoop
		default:

			// list of channels to recieve newe world states
			nextStateResultChannels := make([]chan stubs.NextStateResponse, 4)

			// dial servers make rpc calls
			for i := 0; i < 4; i++ {
				nextStateResultChannels[i] = make(chan stubs.NextStateResponse)
				go makeNextStateCall(clients[i], nextStateResultChannels[i], i)
			}

			var newWorld [][]byte

			// reassemble new world state
			for i := 0; i < 4; i++ {
				newWorld = append(newWorld, (<-nextStateResultChannels[i]).World...)
			}

			// TODO make call to client to send world state

			// get world data
			mutex.Lock()

			// copy of current world world
			oldWorld := make([][]byte, height)
			for i := 0; i < height; i++ {
				oldWorld[i] = make([]byte, width)
			}
			copy(oldWorld, world)

			copy(world, newWorld)
			cellsCount := len(calculateAliveCells())
			cellsFlipped := calculateFlippedCells(oldWorld, world)
			makeSendWorldStateCall(newWorld, cellsFlipped, turn+1, cellsCount)
			mutex.Unlock()
		}
	}
	mutex.Lock()
	resultChan <- world
	mutex.Unlock()
	return
}

type Broker struct{}

func (g *Broker) RunGame(req stubs.RunGameRequest, res *stubs.RunGameResponse) (err error) {

	// set global variables
	mutex.Lock()
	world = req.World     // changes after every turn
	height = req.Height   // should only change after Quit has been called and a new world is passed in to RunGame
	width = req.Width     // should only change after Quit has been called and a new world is passed in to RunGame
	threads = req.Threads // should only change after Quit has been called and a new world is passed in to RunGame
	mutex.Unlock()

	resultChan := make(chan [][]byte)
	turnExecutionFinished.Add(1)

	go RunTurns(req.Turns, resultChan)
	res.World = <-resultChan

	mutex.Lock()
	res.CompletedTurns = turn
	res.AliveCells = calculateAliveCells()
	mutex.Unlock()

	return
}

func (g *Broker) AliveCellsCount(req stubs.AliveCellsCountRequest, res *stubs.AliveCellsCountResponse) (err error) {
	mutex.Lock()
	res.CompletedTurns = turn
	res.CellsCount = len(calculateAliveCells())
	mutex.Unlock()
	return
}

func (g *Broker) Screenshot(req stubs.ScreenshotRequest, res *stubs.ScreenshotResponse) (err error) {
	newWorld := make([][]byte, height)
	for i := 0; i < height; i++ {
		newWorld[i] = make([]byte, width)
	}
	mutex.Lock()
	copy(newWorld, world)
	res.World = newWorld
	mutex.Unlock()
	return
}

func (g *Broker) Quit(req stubs.QuitRequest, res *stubs.QuitResponse) (err error) {
	stopTurnsChan <- struct{}{} // signal that the client wants to quit

	turnExecutionFinished.Wait() // wait for last turn to be completed

	mutex.Lock()
	// reset state
	turn = 0
	height = 0
	width = 0
	world = nil
	// TODO reset distClient ???
	mutex.Unlock()

	return
}

func (g *Broker) CloseBroker(req stubs.CloseBrokerRequest, res *stubs.CloseBrokerResponse) (err error) {
	close(stopTurnsChan) // close channel (even though that doesn't trigger anything, just cleaning up)

	// close servers
	// ! if these requests/responses ever become stateful then will need to make a new req/res pair for each CloseServer call
	closeServerReq := stubs.CloseServerRequest{}
	closeServerRes := new(stubs.CloseServerResponse)
	err = makeCloseServerCall(closeServerReq, closeServerRes)
	if err != nil {
		log.Fatal("Error closing the server:", err)
	}

	close(closeBrokerChan) // signal that we want to close the Broker down
	return
}

func makeCloseServerCall(req stubs.CloseServerRequest, res *stubs.CloseServerResponse) (err error) {

	// Create rpc clients to connect to the servers
	clients := make([]*rpc.Client, 4)

	for i := 0; i < 4; i++ {
		clients[i], err = rpc.Dial("tcp", servers[i]) // dial server
		if err != nil {
			log.Fatal("dialing:", err)
		}
		err = clients[i].Call(stubs.CloseServer, req, res) // close server
		if err != nil {
			log.Fatal("Error calling CloseServer on the server:", err)
		}
	}
	return
}

func (g *Broker) Pause(req stubs.PauseRequest, res *stubs.PauseResponse) (err error) {
	mutex.Lock()
	res.Turn = turn
	return
}

func (g *Broker) Restart(req stubs.PauseRequest, res *stubs.PauseResponse) (err error) {
	res.Turn = turn
	mutex.Unlock()
	return
}

func main() {
	pAddr := "8030"
	// Registering our service
	rpc.Register(&Broker{})

	// Create a network listener
	listener, err := net.Listen("tcp", ":"+pAddr)
	if err != nil {
		fmt.Println("Error starting Broker:", err)
		return
	}

	// initialise server addresses
	servers = make([]string, 4)
	for i := 0; i < 4; i++ {
		servers[i] = "127.0.0.1:" + strconv.Itoa(8050+i)
	}

	// Initialise closeBrokerChan and stopTurnsChan
	closeBrokerChan = make(chan struct{})
	stopTurnsChan = make(chan struct{})

	// Goroutine to accept connections using rpc.Accept
	go func() {
		defer listener.Close()

		fmt.Println("Broker listening on", listener.Addr())

		// Accept connections and serve them
		rpc.Accept(listener)
	}()

	// Block until a close signal is received
	<-closeBrokerChan
	fmt.Println("Broker shutdown complete")
}

func calculateAliveCells() []util.Cell {
	aliveCells := make([]util.Cell, 0, height*width)
	for rowI, row := range world {
		for colI, cellVal := range row {
			if cellVal == 255 {
				aliveCells = append(aliveCells, util.Cell{X: colI, Y: rowI})
			}
		}
	}
	return aliveCells
}

func calculateFlippedCells(oldWorld, newWorld [][]byte) []util.Cell {
	cells := make([]util.Cell, 0, height*width)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if oldWorld[y][x] != newWorld[y][x] {
				cells = append(cells, util.Cell{X: x, Y: y})
			}
		}
	}
	return cells
}
