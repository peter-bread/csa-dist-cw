package main

import (
	"fmt"
	"net"
	"net/rpc"
	"sync"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

var (
	world           [][]byte
	height          int
	width           int
	turn            int
	mutex           sync.Mutex
	closeServerChan chan struct{}
	stopTurnsChan   chan struct{}
	quitClientChan  chan struct{}
	calcFinished    sync.WaitGroup
)

func RunTurns(turns int, resultChan chan<- [][]byte, stopChan <-chan struct{}) (err error) {
	defer calcFinished.Done()
	turn = 0
TurnsLoop:
	for ; turn < turns; turn++ {
		fmt.Println(turn)
		select {
		case <-stopChan:
			// TODO ensure world is copied here
			break TurnsLoop
			// return // Terminate the goroutine if the stop signal is received
		default:
			newWorld := calculateNextState()
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

type GolOperations struct{}

func (g *GolOperations) RunGame(req stubs.RunGameRequest, res *stubs.RunGameResponse) (err error) {

	// TODO make sure a world is returned if the server is shut prematurely

	// global variables
	mutex.Lock()
	world = req.World
	height = req.Height // should never change
	width = req.Width   // should never change
	mutex.Unlock()

	resultChan := make(chan [][]byte)
	calcFinished.Add(1)
	go RunTurns(req.Turns, resultChan, quitClientChan)
	res.World = <-resultChan
	return
}

func (g *GolOperations) AliveCellsCount(req stubs.AliveCellsCountRequest, res *stubs.AliveCellsCountResponse) (err error) {
	mutex.Lock()
	res.CompletedTurns = turn
	mutex.Unlock()
	res.CellsCount = len(calculateAliveCells(req.Height, req.Width))
	return
}

func (g *GolOperations) Screenshot(req stubs.ScreenshotRequest, res *stubs.ScreenshotResponse) (err error) {
	mutex.Lock()
	defer mutex.Unlock()
	newWorld := make([][]byte, height)
	for i := 0; i < height; i++ {
		newWorld[i] = make([]byte, width)
	}
	copy(newWorld, world)
	res.World = newWorld
	return
}

func (g *GolOperations) Quit(req stubs.QuitRequest, res *stubs.QuitResponse) (err error) {
	close(quitClientChan)

	calcFinished.Wait()
	res.Turn = turn

	// reset state
	turn = 0
	height = 0
	width = 0
	world = nil

	// open new channel to listen for quit requests
	quitClientChan = make(chan struct{})

	return
}

func (g *GolOperations) CloseServer(req stubs.CloseServerRequest, res *stubs.CloseServerResponse) (err error) {
	// FIXME i think res.World needs to be initialised
	closeServerChan <- struct{}{}
	res.World = world
	res.Turn = turn
	return
}

func (g *GolOperations) Pause(req stubs.PauseRequest, res *stubs.PauseResponse) (err error) {
	mutex.Lock()
	res.Turn = turn
	return
}

func (g *GolOperations) Restart(req stubs.PauseRequest, res *stubs.PauseResponse) (err error) {
	mutex.Unlock()
	res.Turn = turn
	return
}

func main() {
	pAddr := "8030"
	// Registering our service
	rpc.Register(&GolOperations{})

	// Create a network listener
	listener, err := net.Listen("tcp", ":"+pAddr)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	// Initialize closeServerChan and stopTurnsChan
	closeServerChan = make(chan struct{})
	stopTurnsChan = make(chan struct{})
	quitClientChan = make(chan struct{})

	// Goroutine to accept connections using rpc.Accept
	go func() {
		defer listener.Close()
		defer close(closeServerChan) // Close closeServerChan when the goroutine exits

		fmt.Println("Server listening on", listener.Addr())

		// Accept connections and serve them
		rpc.Accept(listener)
	}()

	// Block until a close signal is received
	<-closeServerChan

	fmt.Println("Shutting down...")

	// Signal the RunTurns goroutine to stop
	close(stopTurnsChan)

	fmt.Println("Server shutdown complete")
}

func calculateNextState() [][]byte {
	//   world[ row ][ col ]
	//      up/down    left/right

	newWorld := make([][]byte, height)
	for i := range newWorld {
		newWorld[i] = make([]byte, width)
	}

	for rowI, row := range world { // for each row of the grid
		for colI, cellVal := range row { // for each cell in the row

			aliveNeighbours := 0 // initially there are 0 living neighbours

			// iterate through neighbours
			for i := -1; i < 2; i++ {
				for j := -1; j < 2; j++ {

					// if cell is a neighbour (i.e. not the cell having its neighbours checked)
					if i != 0 || j != 0 {

						// Calculate neighbour coordinates with wrapping
						neighbourRow := (rowI + i + height) % height
						neighbourCol := (colI + j + width) % width

						// Check if the wrapped neighbour is alive
						if world[neighbourRow][neighbourCol] == 255 {
							aliveNeighbours++
						}
					}
				}
			}
			// implement rules
			if cellVal == 255 && aliveNeighbours < 2 { // cell is lonely and dies
				newWorld[rowI][colI] = 0
			} else if cellVal == 255 && aliveNeighbours > 3 { // cell killed by overpopulation
				newWorld[rowI][colI] = 0
			} else if cellVal == 0 && aliveNeighbours == 3 { // new cell is born
				newWorld[rowI][colI] = 255
			} else { // cell remains as it is
				newWorld[rowI][colI] = world[rowI][colI]
			}
		}
	}
	return newWorld
}

func calculateAliveCells(height, width int) []util.Cell {
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
