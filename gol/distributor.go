package gol

import (
	"flag"
	"log"
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

func makeCall(client *rpc.Client, world [][]byte, p Params) {
	// defined request
	request := stubs.Request{
		Turns:  p.Turns,
		Height: p.ImageHeight,
		Width:  p.ImageWidth,
		World:  world,
	}
	response := new(stubs.Response)
	client.Call(stubs.RunTurns, request, response)
	fmt.Println(world)
}

func distributor(p Params, c distributorChannels) {
	server := flag.String("server", "127.0.0.1:8030", "IP:port string to connect to as server")
	flag.Parse()
	fmt.Println("Server: ", *server)
	// dial server address that has been passed
	client, err := rpc.Dial("tcp", *server)
	if err != nil {
		log.Fatal("dialing:", err)
	}
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

	fmt.Println(world)

	makeCall(client, world, p)

	// TODO: Report the final state using FinalTurnCompleteEvent.
	alive := calculateAliveCells(p, world)
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
