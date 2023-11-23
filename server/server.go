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

func calcHeights(imageHeight, threads int) []int {
	baseHeight := imageHeight / threads
	remainder := imageHeight % threads
	heights := make([]int, threads)

	for i := 0; i < threads; i++ {
		if remainder > 0 { // distribute the remainder as evenly as possible
			heights[i] = baseHeight + 1
			remainder--
		} else {
			heights[i] = baseHeight
		}
	}
	return heights
}

func (s *Server) ReturnNextState(req stubs.NextStateRequest, res *stubs.NextStateResponse) (err error) {
	threads := req.Threads

	// initialise worker channels
	workers := make([]chan [][]byte, threads)
	for i := 0; i < threads; i++ {
		workers[i] = make(chan [][]byte)
	}

	// split heights as evenly as possible
	heights := calcHeights(req.EndY-req.StartY, threads)

	start := req.StartY

	// start workers
	for i := 0; i < threads; i++ {
		// TODO don't start worker if heights[i] == 0. This is not massively important as it just starts workers that return empty slices immediately but its probably better if it doesnt do that
		// TODO will need to change threads to the how many non-zero heights there are
		go worker(start, start+heights[i], req.StartX, req.EndX, req.WorldHeight, req.WorldWidth, req.World, workers[i])
		start += heights[i]
	}

	// store next state here
	var newWorld [][]byte

	// reassemble world
	for i := 0; i < threads; i++ {
		newWorld = append(newWorld, <-workers[i]...)
	}
	res.World = newWorld
	return
}

func worker(startY, endY, startX, endX, world_height, world_width int, world [][]byte, out chan<- [][]byte) {
	out <- calculateNextState(startY, endY, startX, endX, world_height, world_width, world)
}

func calculateNextState(startY, endY, startX, endX, world_height, world_width int, world [][]byte) [][]byte {
	//   world[ row ][ col ]
	//      up/down   left/right

	height := endY - startY
	width := endX - startX

	newWorld := make([][]byte, height)
	for i := range newWorld {
		newWorld[i] = make([]byte, width)
	}

	for rowI, row := range world[startY:endY] { // for each row of the grid
		for colI, cellVal := range row { // for each cell in the row
			aliveNeighbours := 0 // initially there are 0 living neighbours

			// iterate through neighbours
			for i := -1; i < 2; i++ {
				for j := -1; j < 2; j++ {

					// if cell is a neighbour (i.e. not the cell having its neighbours checked)
					if i != 0 || j != 0 {

						// Calculate neighbour coordinates with wrapping
						neighbourRow := (rowI + i + startY + world_height) % world_height
						neighbourCol := (colI + j + world_width) % world_width

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
				newWorld[rowI][colI] = world[rowI+startY][colI+startX]
			}
		}
	}
	return newWorld
}

func (s *Server) CloseServer(req stubs.CloseServerRequest, res *stubs.CloseServerResponse) (err error) {
	close(closeServerChan)
	return
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
