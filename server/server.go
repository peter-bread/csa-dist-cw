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
	closeServerChan chan bool
	pausing         chan bool
	restarting      chan bool
)

func RunTurns(turns int, resultChan chan<- [][]byte, shutdown <-chan bool) (err error) {
	turn = 0
	for ; turn < turns; turn++ {
		select {
		case <-shutdown:
			return
			// to shut the server down, it needs to be a goroutine from the client as it wont respond since its shut down
			// to shut the client down just do it in the client file.
		default:
			newWorld := calculateNextState()
			mutex.Lock()
			copy(world, newWorld)
			mutex.Unlock()
		}
	}
	resultChan <- world
	return
}

type GolOperations struct{}

func (g *GolOperations) RunGame(req stubs.RunGameRequest, res *stubs.RunGameResponse) (err error) {

	// global variables
	world = req.World
	height = req.Height // should never change
	width = req.Width   // should never change

	resultChan := make(chan [][]byte)
	go RunTurns(req.Turns, resultChan, closeServerChan)
	res.World = <-resultChan
	return
}

func (g *GolOperations) AliveCellsCount(req stubs.AliveCellsCountRequest, res *stubs.AliveCellsCountResponse) (err error) {
	res.CompletedTurns = turn
	res.CellsCount = len(calculateAliveCells(req.Height, req.Width))
	return
}

func (g *GolOperations) Screenshot(req stubs.ScreenshotRequest, res *stubs.ScreenshotResponse) (err error) {
	mutex.Lock()
	defer mutex.Unlock()
	print2DArray(res.World)
	newWorld := make([][]byte, height)
	for i := 0; i < height; i++ {
		newWorld[i] = make([]byte, width)
	}
	copy(newWorld, world)
	res.World = newWorld
	return
}

func (g *GolOperations) Quit(req stubs.QuitRequest, res *stubs.QuitResponse) (err error) {
	res.Turn = turn
	return
}

func (g *GolOperations) CloseServer(req stubs.CloseServerRequest, res *stubs.CloseServerResponse) (err error) {
	closeServerChan = make(chan bool, 2)
	closeServerChan <- true
	closeServerChan <- true
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
	// registering our service
	rpc.Register(&GolOperations{})

	// create a network listener
	listener, _ := net.Listen("tcp", ":"+pAddr)
	defer listener.Close()

	// want service to start accepting communications
	// this service will listen for communications from client trying to call that function
	go rpc.Accept(listener)
	// select {
	// case <-shutdownChan:
	// 	listener.Close()
	// }

	<-closeServerChan
	fmt.Println("ee")
	listener.Close()
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

func print2DArray(arr [][]byte) {
	for i := 0; i < len(arr); i++ {
		for j := 0; j < len(arr[i]); j++ {
			fmt.Printf("%d\t", arr[i][j])
		}
		fmt.Println()
	}
}
