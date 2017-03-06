package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

// CycleDelay - delay between cycling words in milliseconds
const CycleDelay = 20000

// listens for incoming udp packets on port 3333 and prints them to stdout
func main() {

	// ordered list of vote station addresses & last beats
	votestns := []int{101, 102, 103}
	votestnbeats := make([]int64, len(votestns))

	// open file to log word & vote events
	// create wrdr to manage cycling words & writing events to json logfile
	f, err := os.OpenFile("wordlog.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	wrdr := NewWrdr(votestns, f)
	defer f.Close() // close word log file on exit

	// read led position file and create bllnkr with data
	leddata, err := ioutil.ReadFile("led_locations.json")
	fmt.Println(len(leddata))
	if err != nil {
		log.Fatal(err)
	}
	blnkr, err := NewBlnkr(leddata)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("lamps:")
	for ip, lmp := range blnkr.Lmps {
		fmt.Println(ip)
		for _, p := range lmp.Pnts {
			fmt.Println(p)
		}
		fmt.Println()
	}

	fmt.Println("topos:")
	for i, topo := range blnkr.Topos {
		fmt.Printf("< %v\n", float32(i+1)*blnkr.TopoStep)
		for _, pr := range topo {
			fmt.Println(blnkr.Lmps[pr.IP].Pnts[pr.Dx])
		}
		fmt.Println()
	}

	// buffered channel to pass vote colors to blnkr
	rgbch := make(chan RGB, 64)

	// buffered channel to receive udp teensymsgs
	tch := make(chan TeensyMsg, 64)

	// buffered channel to receive websocket data clients
	dch := make(chan DataClient, 16)

	// index of connected clients
	dcdx := make(map[string]DataClient)

	// channel to trigger word cycling
	cych := make(chan bool)
	go Metronome(cych, CycleDelay)

	// listen for teensy messages on udp port 3333 and pass them up channel
	go TeensySocket(tch)

	// listen for data clients on ws port 8888 and pass them up channel
	go DataSocket(dch, tch)

	// pass color channel to blnkr udpcast routine
	go blnkr.Cast(rgbch)

	// loop over channels & handle messages
	for {
		select {

		// incoming teensy message channel
		case tm := <-tch:

			// log message
			log.Printf("received: %+v", tm)

			switch tm.Flavor {

			case "touch_beat": // log heartbeat
				var sdx = int(-1)
				for i, src := range votestns {
					if src == tm.Source {
						sdx = i
					}
				}
				if sdx < 0 {
					log.Printf("ERROR: unrecognized touch beat source '%v'\n", tm.Source)
				} else {
					votestnbeats[sdx] = NowMs() // set last beat time to now
				}

			case "start_touch", "end_touch": // broadcast to data clients
				wrdp, err := wrdr.LogTouch(tm.Source, tm.Flavor, tm.Choice)
				if err != nil {
					log.Printf("ERROR: cant log touch: %v", err)
				} else if tm.Flavor == "end_touch" {
					select {
					case rgbch <- wrdp.Clr:
					default:
						log.Printf("ERROR: rgbch full!")
					}
				}

				dm := DataMsg{Source: tm.Source, Flavor: tm.Flavor, Choice: tm.Choice}
				bcastMsg(dm, dcdx)
			}

			fmt.Printf("@")

		// incoming data client channel
		case dc := <-dch:
			log.Printf("new data client at %v", dc.Dest)
			dcdx[dc.Dest] = dc // append data client to index

			fmt.Printf("$")

		// cycle words at intervals
		case _ = <-cych:
			fmt.Printf("[")
			dm := wrdr.CycleWrd() // pick a new word & gen message
			bcastMsg(dm, dcdx)    // broadcast message to data clients
			fmt.Printf("]")
		}
	}
}

func bcastMsg(dm DataMsg, dcdx map[string]DataClient) {
	for dest, dc := range dcdx {
		select {
		case dc.MsgCh <- dm:
		default:
			log.Printf("ERROR: msgch for %v full!", dc.Dest)
			close(dc.MsgCh)
			delete(dcdx, dest)
		}
	}
}

// Metronome - send boolean true down channel at given interval
func Metronome(ch chan bool, ms int64) {
	beat := time.Duration(ms) * time.Millisecond
	for {
		time.Sleep(beat)
		ch <- true
	}
}
