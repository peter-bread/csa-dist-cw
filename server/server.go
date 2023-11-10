package main

import (
	"net"
	"net/rpc"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

var (
	world  [][]byte
	height int
	width  int
	turn   int
	mutex  sync.Mutex
)

func RunTurns(turns int) [][]byte {
	for ; turn < turns; turn++ {
		mutex.Lock()
		world = calculateNextState(height, width, world)
		mutex.Unlock()
	}

	return world
}

func ReturnAlive(tickerChan <-chan time.Time) (int, int) {
	select {
	case <-tickerChan:
		mutex.Lock()
		aliveCount := len(calculateAliveCells(height, width, world))
		mutex.Unlock()
		return turn, aliveCount
	}
}

type GolOperations struct {
}

func (g *GolOperations) ProcessTurns(req stubs.Request, res *stubs.Response) (err error) {
	world = req.World
	height = req.Height
	width = req.Width
	turn = 0
	res.World = RunTurns(req.Turns)
	return
}

func (g *GolOperations) TickerInstant(req stubs.TickerRequest, res *stubs.TickerResponse) (err error) {
	res.CompletedTurns, res.CellsCount = ReturnAlive(req.TickerChan)
	return
}

func main() {
	pAddr := "8030"
	// registering our service
	rpc.Register(&GolOperations{})

	// create a network listener
	listener, _ := net.Listen("tcp", ":"+pAddr)
	defer listener.Close()

	// want service to start accepting communications
	// this service will listen for communications from client trying to call that function
	rpc.Accept(listener)
}

func calculateNextState(height, width int, world [][]byte) [][]byte {
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
						neighbourRow := (rowI + i + height) % width
						neighbourCol := (colI + j + height) % width

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

func calculateAliveCells(height, width int, world [][]byte) []util.Cell {
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
