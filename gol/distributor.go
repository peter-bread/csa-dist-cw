package gol

import (
	"log"
	"net"
	"net/rpc"
	"sync"
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

var (
	wg             sync.WaitGroup
	worldStateChan chan stubs.SendWorldStateRequest
)

type Controller struct{}

func (c *Controller) SendWorldState(req stubs.SendWorldStateRequest, res *stubs.SendWorldStateResponse) (err error) {
	wg.Add(1)
	defer wg.Done() // the last one of these must finsih processing before the client can be shut down
	worldStateChan <- req
	return
}

// define makeReadyToDialCall to tell broker it is safe to dial the client
func makeReadyToDialCall(client *rpc.Client, resultChan chan<- stubs.ReadyToDialResponse) {
	req := stubs.ReadyToDialRequest{S: "controller is connected to broker"}
	res := new(stubs.ReadyToDialResponse)
	client.Call(stubs.ReadyToDial, req, res)
	fmt.Println(res.S)
	resultChan <- *res
}

func makeRunGameCall(client *rpc.Client, world [][]byte, p Params, resultChan chan<- stubs.RunGameResponse) {
	defer wg.Done()
	req := stubs.RunGameRequest{
		Turns:   p.Turns,
		Height:  p.ImageHeight,
		Width:   p.ImageWidth,
		Threads: p.Threads,
		World:   world,
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

func makeCloseBrokerCall(client *rpc.Client) {
	req := stubs.CloseBrokerRequest{}
	res := new(stubs.CloseBrokerResponse)
	client.Call(stubs.CloseBroker, req, res)
}

func makePauseCall(client *rpc.Client, resultChan chan<- stubs.PauseResponse) {
	req := stubs.PauseRequest{}
	res := new(stubs.PauseResponse)
	client.Call(stubs.Pause, req, res)
	resultChan <- *res
}

func makeRestartCall(client *rpc.Client, resultChan chan<- stubs.RestartResponse) {
	req := stubs.RestartRequest{}
	res := new(stubs.RestartResponse)
	client.Call(stubs.Restart, req, res)
	resultChan <- *res
}

func distributor(p Params, c distributorChannels) {
	broker := "127.0.0.1:8030"
	fmt.Println("Broker: ", broker)

	// dial Broker address that has been passed
	client, err := rpc.Dial("tcp", broker)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer client.Close()

	rpc.Register(&Controller{})

	pAddr := "8020"

	listener, err := net.Listen("tcp", ":"+pAddr)
	if err != nil {
		fmt.Println("Error starting Controller:", err)
		return
	}

	worldStateChan = make(chan stubs.SendWorldStateRequest, 1000000)

	// start listening to broker on 8020
	go func() {
		defer listener.Close()

		fmt.Println("Controller listening on", listener.Addr())

		// Accept connections and serve them
		rpc.Accept(listener)
	}()

	// send request to broker to say broker can dial the client
	readyToDialResultChannel := make(chan stubs.ReadyToDialResponse)
	go makeReadyToDialCall(client, readyToDialResultChannel)

	// wait for response to say the broker has dialled client successfully (2-way comms is now available)
	<-readyToDialResultChannel

	// read in image
	filename := fmt.Sprintf("%vx%v", p.ImageWidth, p.ImageHeight)
	c.ioCommand <- ioInput
	c.ioFilename <- filename

	// Create a 2D slice to store the world.
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}

	// send initial CellFlipped events
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

	stopListening := make(chan struct{})

	// receive world state updates after every turn and send the data down the events channel
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case s := <-worldStateChan:

				// send CellFlipped events
				for _, cell := range s.CellsFlipped {
					c.events <- CellFlipped{
						CompletedTurns: s.CompletedTurns,
						Cell:           cell,
					}
				}

				// send TurnComplete event
				c.events <- TurnComplete{
					CompletedTurns: s.CompletedTurns,
				}

			case <-stopListening:
				return
			}
		}
	}()

	// start ticker
	ticker := time.NewTicker(2 * time.Second)

	wg.Add(1)
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

	paused := false // stores whether execution has been paused
	// finalTurns := p.Turns // number of turns completed when program exits

	go func() {
		for {
			select {
			case key := <-c.keyPresses:
				switch key {
				case 's':
					pgmResultChannel := make(chan stubs.ScreenshotResponse)
					go makeScreenshotCall(client, pgmResultChannel)
					generatePGM(p, c, (<-pgmResultChannel).World)
				case 'q':
					quitResultChannel := make(chan stubs.QuitResponse)
					go makeQuitCall(client, quitResultChannel)
					<-quitResultChannel
					return
				case 'k':
					// send quit request
					quitResultChannel := make(chan stubs.QuitResponse)
					go makeQuitCall(client, quitResultChannel)
					<-quitResultChannel

					// wait for world to be read from Broker
					wg.Wait()

					// send close request
					go makeCloseBrokerCall(client)
					return
				case 'p':
					paused = !paused
					if paused {
						pauseResultChan := make(chan stubs.PauseResponse)
						go makePauseCall(client, pauseResultChan)
						ticker.Stop()
						c.events <- StateChange{(<-pauseResultChan).Turn, Paused}
					} else {
						restartResultChan := make(chan stubs.RestartResponse)
						go makeRestartCall(client, restartResultChan)
						ticker.Reset(2 * time.Second)
						c.events <- StateChange{(<-restartResultChan).Turn, Executing}
					}
				}
			}
		}
	}()

	// get game result from broker
	runGameResult := <-runGameResultChannel
	ticker.Stop()

	// stop receiving world updates
	close(stopListening)

	// get final world and turns completed
	finalWorld := runGameResult.World
	finalCompletedTurns := runGameResult.CompletedTurns
	finalAliveCells := runGameResult.AliveCells

	// generate pgm image of final world state
	generatePGM(p, c, finalWorld)

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{finalCompletedTurns, Quitting}

	// Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{
		CompletedTurns: finalCompletedTurns,
		Alive:          finalAliveCells,
	}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
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
