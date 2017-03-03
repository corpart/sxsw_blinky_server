package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"log"
	"net"

	"github.com/go-gl/mathgl/mgl32"
)

// LampSize - # of pnt (led) slots in each lamp
const LampSize = 16

// UDPPort - port # to send udp messages to
const UDPPort = "3333"

// UpdateDelay - delay between lamp udpcasts
var UpdateDelay = int64(200)

// WvDelay - delay between
var WvDelay = int64(4000)

// StrkThrsh - threshold for color streak to change inwaves
var StrkThrsh = int(7)

// WvClr - default inwv color
var WvClr = RGB{0x777, 0x777, 0x777}

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

// Add - add two colors & return result
func (rgb RGB) Add(o RGB) RGB {
	n := RGB{}
	for i := 0; i < 3; i++ {
		n[i] = rgb[i] + o[i]
	}
	return n
}

// Dim - multiply color by float & return result
func (rgb RGB) Dim(f float32) RGB {
	n := RGB{}
	for i := 0; i < 3; i++ {
		n[i] = int16(float32(rgb[i]) * f)
	}
	return n
}

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

// PntRf - reference to point in lamp by ip & index
type PntRf struct {
	IP string
	Dx int
}

// Topo - bucket holding leds within certain dist from epicenters
type Topo []PntRf

// Wv - wave pattern to render in lmp topos
type Wv struct {
	Dx   int
	Dlta int
	Clr  RGB
	Shp  []float32
}

// Blnkr - manages collection of leds sorted into topo buckets for wave anim
type Blnkr struct {
	Lmps     map[string]*Lmp
	Topos    []Topo
	Wvs      []Wv
	Epcntrs  []mgl32.Vec3
	TopoStep float32
}

// NewBlnkr - init blnkr with given json file
func NewBlnkr(jsond []byte) (*Blnkr, error) {

	// unmarshal json data into a slice of Leds
	var leds []Led
	err := json.Unmarshal(jsond, &leds)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// create blinkr to hold led data
	blnkr := Blnkr{
		TopoStep: 10.0,
		Epcntrs:  []mgl32.Vec3{{0.0, 0.0, 0.0}, {100.0, 0.0, 0.0}, {200.0, 0.0, 0.0}},
	}

	// create pnts & lmps & topos from slice of leds
	lmps := make(map[string]*Lmp)
	topos := make([]Topo, 1)
	for _, led := range leds {

		// get lmp for ip; if there is not already a lmp for this ip create it
		if _, has := lmps[led.IP]; !has {
			lmps[led.IP] = &Lmp{IP: led.IP}
		}
		lmp := lmps[led.IP]

		// create pnt and add it to lmp at index
		pnt := Pnt{Crds: mgl32.Vec3{led.X, led.Y, led.Z}}
		lmp.Pnts[led.Index] = pnt
		//lmps[led.IP] = lmp

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

			// add ref to point in lamp to topo bucket so we can access lamp color
			topos[tdx] = append(topos[tdx], PntRf{led.IP, led.Index})
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
				log.Println("ERROR: binary write of color to buffer failed:", err)
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

// Cast - routine to loop & update leds
func (blnkr *Blnkr) Cast(rgbch chan RGB) {
	lastupdate := NowMs()
	lastwv := NowMs()
	lastclr := RGB{}
	clrstrk := 0

	for {
		select {

		// create new outwave in word color when vote received
		case c := <-rgbch:
			blnkr.makeOutWv(c)

			// track vote streaks
			if c == lastclr {
				clrstrk++
			} else {
				clrstrk = 1
				lastclr = c
			}

		default:
			nw := NowMs()

			// update waves & udpcast
			if lastupdate+UpdateDelay < nw {
				lastupdate = nw
				blnkr.updateWvs()
				blnkr.UDPCast()
			}

			// generate new inwaves
			if lastwv+WvDelay < nw {
				lastwv = nw
				if clrstrk >= StrkThrsh {
					blnkr.makeInWv(lastclr)
				} else {
					blnkr.makeInWv(WvClr)
				}
			}
		}
	}
}

func (blnkr *Blnkr) makeInWv(clr RGB) {
	wv := Wv{
		Dx:   len(blnkr.Topos) - 1,
		Dlta: -1,
		Clr:  clr,
		Shp:  []float32{0.2, 0.7, 1.0, 0.8, 0.5, 0.3, 0.1},
	}
	blnkr.Wvs = append(blnkr.Wvs, wv)
}

func (blnkr *Blnkr) makeOutWv(clr RGB) {
	wv := Wv{
		Dx:   -3,
		Dlta: 2,
		Clr:  clr,
		Shp:  []float32{0.1, 0.3, 0.5, 0.2},
	}
	blnkr.Wvs = append(blnkr.Wvs, wv)
}

func (blnkr *Blnkr) updateWvs() {

	// zero lmp colors
	for _, lmp := range blnkr.Lmps {
		for i := 0; i < LampSize; i++ {
			lmp.Pnts[i].Clr = RGB{}
		}
	}

	// update waves & apply to topos
	wvs := []Wv{}
	for _, wv := range blnkr.Wvs {
		wv.Dx += wv.Dlta // update position index by dlta

		// check if wv is past topos
		if !((wv.Dlta < 0 && wv.Dx+len(wv.Shp) < 0) || (wv.Dlta > 0 && wv.Dx > len(blnkr.Topos))) {
			wvs = append(wvs, wv)

			// apply wv to topos
			for i, k := range wv.Shp {
				dx := i + wv.Dx
				if dx >= 0 && dx < len(blnkr.Topos) {
					for _, pr := range blnkr.Topos[dx] {
						clr := blnkr.Lmps[pr.IP].Pnts[pr.Dx].Clr
						clr = clr.Add(wv.Clr.Dim(k))
						blnkr.Lmps[pr.IP].Pnts[pr.Dx].Clr = clr
					}
				}
			}
		}
	}
	blnkr.Wvs = wvs
}

// calculate distance between two vec3s
func distance(p, q mgl32.Vec3) float32 {
	dlta := q.Sub(p)
	return dlta.Len()
}
