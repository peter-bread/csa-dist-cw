package gol

import (
	"log"
	"net/rpc"
	"time"

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
	keyPresses <-chan rune
}

func makeRunGameCall(client *rpc.Client, world [][]byte, p Params, resultChan chan<- stubs.RunGameResponse) {
	// defined req
	req := stubs.RunGameRequest{
		Turns:  p.Turns,
		Height: p.ImageHeight,
		Width:  p.ImageWidth,
		World:  world,
	}
	res := new(stubs.RunGameResponse)
	client.Call(stubs.RunGame, req, res)

	resultChan <- *res
}

func makeAliveCellsCountCall(client *rpc.Client, resultChan chan<- stubs.AliveCellsCountResponse) {
	req := stubs.AliveCellsCountRequest{}
	res := new(stubs.AliveCellsCountResponse)
	client.Call(stubs.AliveCellsCount, req, res)
	resultChan <- *res
}

func makeScreenshotCall(client *rpc.Client, resultChan chan<- stubs.ScreenshotResponse) {
	req := stubs.ScreenshotRequest{}
	res := new(stubs.ScreenshotResponse)
	client.Call(stubs.Screenshot, req, res)
	resultChan <- *res
}

func makeQuitCall(client *rpc.Client, resultChan chan<- stubs.QuitResponse) {
	req := stubs.QuitRequest{}
	res := new(stubs.QuitResponse)
	client.Call(stubs.Quit, req, res)
	resultChan <- *res
}

func makeCloseServerCall(client *rpc.Client, resultChan chan<- stubs.CloseServerResponse) {
	req := stubs.CloseServerRequest{}
	res := new(stubs.CloseServerResponse)
	client.Call(stubs.Shutdown, req, res)
	resultChan <- *res
}

func makePauseCall(client *rpc.Client, resultChan chan<- stubs.PauseResponse) {
	req := stubs.PauseRequest{}
	res := new(stubs.PauseResponse)
	client.Call(stubs.Pause, req, res)
	resultChan <- *res
}

func distributor(p Params, c distributorChannels) {
	server := "127.0.0.1:8030"
	fmt.Println("Server: ", server)
	// dial server address that has been passed
	client, err := rpc.Dial("tcp", server)
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

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			world[y][x] = <-c.ioInput
			if world[y][x] == 255 {
				c.events <- CellFlipped{
					CompletedTurns: 0,
					Cell:           util.Cell{X: x, Y: y},
				}
			}
		}
	}

	ticker := time.NewTicker(2 * time.Second)

	runGameResultChannel := make(chan stubs.RunGameResponse)
	go makeRunGameCall(client, world, p, runGameResultChannel)

	aliveCellsCountResultChannel := make(chan stubs.AliveCellsCountResponse)
	go func() {
		for {
			select {
			case <-ticker.C:
				go makeAliveCellsCountCall(client, aliveCellsCountResultChannel)
				result := <-aliveCellsCountResultChannel
				c.events <- AliveCellsCount{
					CompletedTurns: result.CompletedTurns,
					CellsCount:     result.CellsCount,
				}
			}
		}
	}()

	go func() {
	keysLoop:
		for {
			select {
			case key := <-c.keyPresses:
				pgmResultChannel := make(chan stubs.ScreenshotResponse)
				switch key {
				case 's':
					go makeScreenshotCall(client, pgmResultChannel)
					generatePGM(p, c, (<-pgmResultChannel).World)
				case 'q':
					quitChannel := make(chan stubs.QuitResponse)
					go makeQuitCall(client, quitChannel)
					c.events <- StateChange{(<-quitChannel).Turn, Quitting}
					c.ioCommand <- ioCheckIdle
					<-c.ioIdle
					close(c.events)
					break keysLoop
				case 'k':
					closeServer := make(chan stubs.CloseServerResponse)
					go makeCloseServerCall(client, closeServer)
					fmt.Println("hello")
					quitChannel := make(chan stubs.QuitResponse)
					go makeQuitCall(client, quitChannel)
					c.events <- StateChange{(<-quitChannel).Turn, Quitting}
					c.ioCommand <- ioCheckIdle
					<-c.ioIdle
					close(c.events)
					break keysLoop
				case 'p':
					pauseChan := make(chan stubs.PauseResponse)
					go makePauseCall(client, pauseChan)
					fmt.Printf("Current Turn: %d\n", (<-pauseChan).Turn)
				}
			}
		}
	}()

	finalWorld := (<-runGameResultChannel).World
	ticker.Stop()

	// Report the final state using FinalTurnCompleteEvent.
	// ? Should the final alive cells be calculated in the server??? does this count as GOL logic???
	alive := calculateAliveCells(p, finalWorld)
	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          alive,
	}

	generatePGM(p, c, finalWorld)

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

// ? see query above
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

func generatePGM(p Params, c distributorChannels, world [][]byte) {
	// after all turns send state of board to be outputted as a .pgm image

	filename := fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, p.Turns)
	c.ioCommand <- ioOutput
	c.ioFilename <- filename

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- world[y][x]
		}
	}
}

func print2DArray(arr [][]byte) {
	for i := 0; i < len(arr); i++ {
		for j := 0; j < len(arr[i]); j++ {
			fmt.Printf("%d\t", arr[i][j])
		}
		fmt.Println()
	}
}
