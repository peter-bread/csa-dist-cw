package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

var (
	world                 [][]byte
	height                int
	width                 int
	turn                  int
	mutex                 sync.Mutex
	closeBrokerChan       chan struct{}
	stopTurnsChan         chan struct{}
	turnExecutionFinished sync.WaitGroup
)

func makeNewStateCall(client *rpc.Client, resultChan chan<- stubs.NextStateResponse) {
	req := stubs.NextStateRequest{
		World:  world,
		Height: height,
		Width:  width,
	}
	res := new(stubs.NextStateResponse)
	client.Call(stubs.NextState, req, res)
	resultChan <- *res
}

func RunTurns(turns int, resultChan chan<- [][]byte) (err error) {
	defer turnExecutionFinished.Done()
	turn = 0

TurnsLoop:
	for ; turn < turns; turn++ {
		select {
		case <-stopTurnsChan:
			break TurnsLoop
		default:
			// TODO 1.  split the world into 4 slices and send each of them to different servers to be processed
			// TODO 2a. start by hardcoding 4 different servers on 4 ports (8050-8053) (manually start servers in separate shell sessions)
			// TODO 2b. use os/exec to start shell sessions and run servers in there

			server := "127.0.0.1:8050"

			// dial server address
			client, err := rpc.Dial("tcp", server)
			if err != nil {
				log.Fatal("dialing:", err)
			}

			nextStateResultChannel := make(chan stubs.NextStateResponse)
			go makeNewStateCall(client, nextStateResultChannel)
			newWorld := (<-nextStateResultChannel).World
			mutex.Lock()
			copy(world, newWorld)
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
	world = req.World
	height = req.Height // should only change after Quit has been called and a new world is passed in to RunGame
	width = req.Width   // should only change after Quit has been called and a new world is passed in to RunGame
	mutex.Unlock()

	resultChan := make(chan [][]byte)
	turnExecutionFinished.Add(1)
	go RunTurns(req.Turns, resultChan)
	res.World = <-resultChan
	return
}

func (g *Broker) AliveCellsCount(req stubs.AliveCellsCountRequest, res *stubs.AliveCellsCountResponse) (err error) {
	mutex.Lock()
	res.CompletedTurns = turn
	mutex.Unlock()
	res.CellsCount = len(calculateAliveCells())
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

	res.Turn = turn

	// reset state
	turn = 0
	height = 0
	width = 0
	world = nil

	mutex.Unlock()

	return
}

func (g *Broker) CloseBroker(req stubs.CloseBrokerRequest, res *stubs.CloseBrokerResponse) (err error) {
	close(stopTurnsChan)   // close channel (even though that doesn't trigger anything, just cleaning up)
	close(closeBrokerChan) // signal that we want to close the Broker down
	return
}

func (g *Broker) Pause(req stubs.PauseRequest, res *stubs.PauseResponse) (err error) {
	mutex.Lock()
	res.Turn = turn
	return
}

func (g *Broker) Restart(req stubs.PauseRequest, res *stubs.PauseResponse) (err error) {
	mutex.Unlock()
	res.Turn = turn
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

	// Initialize closeBrokerChan and stopTurnsChan
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
	mutex.Lock()
	defer mutex.Unlock()
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
