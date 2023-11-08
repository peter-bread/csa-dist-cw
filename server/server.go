package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"uk.ac.bris.cs/gameoflife/stubs"
)

func RunTurns(turns, height, width int, world [][]byte) [][]byte {
	turn := 0
	for ; turn < turns; turn++ {
		fmt.Println(turn)
		world = calculateNextState(height, width, world)
	}

	return world
}

type GolOperations struct {
}

func (g *GolOperations) ProcessTurns(req stubs.Request, res *stubs.Response) (err error) {
	res.World = RunTurns(req.Turns, req.Height, req.Width, req.World)
	return
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	// registering our service
	rpc.Register(&GolOperations{})

	// create a network listener
	listener, _ := net.Listen("tcp", ":"+*pAddr)
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
