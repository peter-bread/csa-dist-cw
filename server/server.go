package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

/** Super-Secret `reversing a string' method we can't allow clients to see. **/
func RunTurns(p stubs.GolParams, world [][]byte) [][]byte {
	fmt.Println("heyy")

	turn := 0
	for ; turn < p.Turns; turn++ {
		// start worker goroutines here
		world = calculateNextState(p, world)
	}

	return world
}

type GolOperations struct {
}

func (g *GolOperations) ProcessTurns(req stubs.Request, res *stubs.Response) (err error) {
	fmt.Println("called")

	res.World = RunTurns(req.Params, req.World)
	// returning by altering value of res at the pointer
	fmt.Println("returned")

	return nil
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	// part of the req string implementation
	rand.Seed(time.Now().UnixNano())
	fmt.Print("HEllo")
	// registering our service
	rpc.Register(&GolOperations{})
	fmt.Println("he223llo")

	// create a network listener
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()

	// want service to start accepting communications
	// this service will listen for communications from client trying to call that function
	rpc.Accept(listener)
}

func calculateNextState(p stubs.GolParams, world [][]byte) [][]byte {
	//   world[ row ][ col ]
	//      up/down    left/right

	newWorld := make([][]byte, p.ImageHeight)
	for i := range newWorld {
		newWorld[i] = make([]byte, p.ImageWidth)
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
						neighbourRow := (rowI + i + p.ImageHeight) % p.ImageHeight
						neighbourCol := (colI + j + p.ImageWidth) % p.ImageWidth

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
