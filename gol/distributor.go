package gol

import (
	//	"net/rpc"
	"flag"
	"net/rpc"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"

	"fmt"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

func distributor(p Params, c distributorChannels) {
	server := flag.String("server", "127.0.0.1:8030", "IP:port string to connect to as server")
	flag.Parse()
	fmt.Println("Server: ", *server)
	// build an rpc client that mirrors the rpc server on the other side
	// dial server address that has been passed
	client, _ := rpc.Dial("tcp", *server)
	defer client.Close()

	filename := fmt.Sprintf("%vx%v", p.ImageWidth, p.ImageHeight)
	c.ioCommand <- ioInput
	c.ioFilename <- filename

	// Create a 2D slice to store the world.
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}

	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			world[i][j] = <-c.ioInput
		}
	}

	// defined request
	request := stubs.Request{
		Params: stubs.GolParams{
			Turns:       p.Turns,
			Threads:     p.Threads,
			ImageWidth:  p.ImageWidth,
			ImageHeight: p.ImageHeight,
		},
		World: world,
	}

	// created space for a response
	response := new(stubs.Response)
	fmt.Println("here!")

	client.Call(stubs.RunTurns, request, response)
	fmt.Println("makes it!")

	// TODO: Report the final state using FinalTurnCompleteEvent.
	alive := calculateAliveCells(p, response.World)
	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          alive,
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	aliveCells := make([]util.Cell, 0, p.ImageHeight*p.ImageWidth)
	for rowI, row := range world {
		for colI, cellVal := range row {
			if cellVal == 255 {
				aliveCells = append(aliveCells, util.Cell{X: colI, Y: rowI})
			}
		}
	}
	return aliveCells
}
