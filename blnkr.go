package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/go-gl/mathgl/mgl32"
)

// LampSize - # of pnt (led) slots in each lamp
const LampSize = 16

// UDPPort - port # to send udp messages to
const UDPPort = "3333"

// Led - type for reading led data from json file
type Led struct {
	IP    string  `json:"ip"`
	Index int     `json:"index"`
	X     float32 `json:"x"`
	Y     float32 `json:"y"`
	Z     float32 `json:"z"`
}

// RGB - holds 16 bit rgb color
type RGB [3]int16

// Pnt - position & color of an led in a lamp
type Pnt struct {
	Crds mgl32.Vec3
	Clr  RGB
}

// Lmp - a lamp with an ip and a list of child leds
type Lmp struct {
	IP   string
	Pnts [LampSize]Pnt
}

// Topo - bucket holding leds within certain dist from epicenters
type Topo []*Pnt

// Blnkr - manages collection of leds sorted into topo buckets for wave anim
type Blnkr struct {
	Lmps     map[string]Lmp
	Topos    []Topo
	Epcntrs  []mgl32.Vec3
	TopoStep float32
}

// NewBlnkr - init blnkr with given json file
func NewBlnkr(jsond []byte) (*Blnkr, error) {

	// unmarshal json data into a slice of Leds
	var leds []Led
	err := json.Unmarshal(jsond, &leds)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	fmt.Println(len(leds))
	fmt.Println(leds[0])

	// create blinkr to hold led data
	blnkr := Blnkr{
		TopoStep: 10.0,
		Epcntrs:  []mgl32.Vec3{{0.0, 0.0, 0.0}, {100.0, 0.0, 0.0}, {200.0, 0.0, 0.0}},
	}

	// create pnts & lmps & topos from slice of leds
	lmps := make(map[string]Lmp)
	topos := make([]Topo, 1)
	for _, led := range leds {

		// get lmp for ip; if there is not already a lmp for this ip create it
		if _, has := lmps[led.IP]; !has {
			lmps[led.IP] = Lmp{IP: led.IP}
		}
		lmp := lmps[led.IP]

		// create pnt and add it to lmp at index
		pnt := Pnt{Crds: mgl32.Vec3{led.X, led.Y, led.Z}}
		lmp.Pnts[led.Index] = pnt
		lmps[led.IP] = lmp

		// add pnt to topo by min distance to epicenter
		var mndst float32 = -1.0
		for _, ep := range blnkr.Epcntrs {
			dst := distance(ep, pnt.Crds)
			if mndst < 0.0 || dst < mndst {
				mndst = dst
			}
		}
		if mndst >= 0.0 {
			tdx := int(mndst / blnkr.TopoStep) // topo bucket index

			// extend list of topo buckets if not already big enough
			if len(topos) <= tdx {
				topos = append(topos, make([]Topo, tdx-len(topos)+1)...)
			}

			// add pointer to point in lamp to topo bucket so we can access lamp color
			topos[tdx] = append(topos[tdx], &lmp.Pnts[led.Index])
		}

	}
	blnkr.Lmps = lmps
	blnkr.Topos = topos

	return &blnkr, nil
}

// UDPCast - send udp packet with current colors to each lamp
func (blnkr *Blnkr) UDPCast() {
	for ip, lmp := range blnkr.Lmps {

		// write color values for lamps leds to buffer
		buf := new(bytes.Buffer)
		for i := 0; i < LampSize; i++ {
			err := binary.Write(buf, binary.BigEndian, lmp.Pnts[i].Clr)
			if err != nil {
				fmt.Println("ERROR: binary write of color to buffer failed:", err)
			}
		}

		// connect to local udp server on port 3333
		dst := ip + ":" + UDPPort
		conn, err := net.Dial("udp", dst)
		if err != nil {
			log.Printf("ERROR: failed get udp conn for lamp at %v: %v", dst, err)
		}
		defer conn.Close()

		// write buffer to server
		_, err = conn.Write(buf.Bytes())
		if err != nil {
			log.Printf("ERROR: failed to send udp message to lamp at %v: %v", dst, err)
		}
	}
}

// calculate distance between two vec3s
func distance(p, q mgl32.Vec3) float32 {
	dlta := q.Sub(p)
	return dlta.Len()
}
