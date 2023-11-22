package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"

	"uk.ac.bris.cs/gameoflife/stubs"
)

var closeServerChan chan struct{}

type Server struct{}

func (s *Server) ReturnNextState(req stubs.NextStateRequest, res *stubs.NextStateResponse) (err error) {
	res.World = calculateNextState(req.Height, req.Width, req.World)
	return
}

func (s *Server) CloseServer(req stubs.CloseServerRequest, res *stubs.CloseServerResponse) (err error) {
	close(closeServerChan)
	return
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

func main() {
	var pAddr string
	flag.StringVar(&pAddr, "port", "8050", "set the port that the server will listen on")
	flag.Parse()
	fmt.Println(pAddr)

	rpc.Register(&Server{})
	listener, err := net.Listen("tcp", ":"+pAddr)
	if err != nil {
		fmt.Println(err)
	}
	closeServerChan = make(chan struct{})
	go func() {
		fmt.Println("Server listening on", listener.Addr())
		defer listener.Close()
		rpc.Accept(listener)
	}()

	<-closeServerChan
	fmt.Println("Server shutdown complete")

}
