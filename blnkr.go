package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"log"
	"math"
	"net"

	"github.com/go-gl/mathgl/mgl64"
)

// LampSize - # of pnt (led) slots in each lamp
const LampSize = 16

// UDPPort - port # to send udp messages to
const UDPPort = "3333"

// UpdateDelay - delay between lamp udpcasts
const UpdateDelay = 33 // ~30hz

// WvDelay - delay between starting new waves
const WvDelay = 6000

// StrkThrsh - threshold for color streak to change inwaves
var StrkThrsh = int(7)

// WvClr - default inwv color
var WvClr = RGB{0x777, 0x777, 0x777}

// Led - type for reading led data from json file
type Led struct {
	IP    string  `json:"ip"`
	Index int     `json:"index"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Z     float64 `json:"z"`
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
func (rgb RGB) Dim(f float64) RGB {
	n := RGB{}
	for i := 0; i < 3; i++ {
		n[i] = int16(float64(rgb[i]) * f)
	}
	return n
}

// Pnt - position & color of an led in a lamp
type Pnt struct {
	Crds mgl64.Vec3
	Mres []float64 // min radius to epicenter
	Clr  RGB
}

// VtClr - station & color of vote
type VtClr struct {
	Stn int
	Clr RGB
}

// Lmp - a lamp with an ip and a list of child leds
type Lmp struct {
	IP   string
	Pnts [LampSize]Pnt
}

// Wv - wave pattern to render in lmp topos
type Wv struct {
	Mn   float64
	SD   float64
	Dlta float64
	Xs   float64 // xscale for gaussian
	Ys   float64 // yscale for gaussian
	Clr  RGB
}

// Blnkr - manages collection of leds sorted into topo buckets for wave anim
type Blnkr struct {
	Lmps    map[string]*Lmp
	Wvs     [][]Wv
	Epcntrs [][]mgl64.Vec3
	StnMp   map[int]int // map from vote station source to epicenter index
	Mxrs    []float64   // max radius from epicenter
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
		Epcntrs: [][]mgl64.Vec3{
			[]mgl64.Vec3{{0.0, 0.0, 0.0}, {300.0, 0.0, 0.0}},
			[]mgl64.Vec3{{80.0, 0.0, 0.0}},
			[]mgl64.Vec3{{170.0, 0.0, 0.0}},
			[]mgl64.Vec3{{260.0, 0.0, 0.0}},
		},
		StnMp: map[int]int{101: 1, 102: 2, 103: 3},
	}
	mxrs := []float64{0, 0, 0, 0} // get max (min) radius of leds from epicenters

	// create pnts & lmps from slice of leds
	lmps := make(map[string]*Lmp)
	for _, led := range leds {

		// get lmp for ip; if there is not already a lmp for this ip create it
		if _, has := lmps[led.IP]; !has {
			lmps[led.IP] = &Lmp{IP: led.IP}
		}
		lmp := lmps[led.IP]

		// create pnt and add it to lmp at index
		c := mgl64.Vec3{led.X, led.Y, led.Z}
		rs := []float64{-1, -1, -1, -1}
		for i := 0; i < 4; i++ {
			rs[i] = blnkr.mre(i, c)
			if rs[i] > mxrs[i] {
				mxrs[i] = rs[i]
			}
		}
		pnt := Pnt{Crds: c, Mres: rs}
		lmp.Pnts[led.Index] = pnt
	}
	blnkr.Lmps = lmps
	blnkr.Mxrs = mxrs
	blnkr.Wvs = make([][]Wv, 4)

	return &blnkr, nil
}

// calculate min radius to epicenter of given index for coordinate
func (blnkr *Blnkr) mre(dx int, crds mgl64.Vec3) float64 {
	var mndst = -1.0
	for _, ep := range blnkr.Epcntrs[dx] {
		dst := distance(ep, crds)
		if mndst < 0.0 || dst < mndst {
			mndst = dst
		}
	}
	return mndst
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
			// log.Printf("ERROR: failed to send udp message to lamp at %v: %v", dst, err)
		}

		// log.Printf("udpcast: %v", buf.Bytes())
	}
}

// Cast - routine to loop & update leds
func (blnkr *Blnkr) Cast(rgbch chan VtClr) {
	lastclr := RGB{}
	clrstrk := 0

	// trigger wave updates
	uch := make(chan bool)
	go Metronome(uch, UpdateDelay)

	// trigger new waves
	wch := make(chan bool)
	go Metronome(wch, WvDelay)

	for {
		select {

		// create new outwave in word color when vote received
		case vc := <-rgbch:

			c := vc.Clr
			edx := blnkr.StnMp[vc.Stn]
			blnkr.makeOutWv(edx, c)

			// track vote streaks
			if c == lastclr {
				clrstrk++
			} else {
				clrstrk = 1
				lastclr = c
			}

		// update waves & udpcast
		case _ = <-uch:
			blnkr.updateWvs()
			blnkr.UDPCast()

		// generate new inwaves
		case _ = <-wch:
			if clrstrk >= StrkThrsh {
				blnkr.makeInWv(lastclr)
			} else {
				blnkr.makeInWv(WvClr)
			}
		}
	}
}

func (blnkr *Blnkr) makeInWv(clr RGB) {
	wv := Wv{
		SD:   1.3,
		Dlta: -0.05,
		Xs:   20.0,
		Ys:   2.0,
		Clr:  clr,
	}
	wv.Mn = blnkr.Mxrs[0] + (wv.SD * 3.0 * wv.Xs)
	blnkr.Wvs[0] = append(blnkr.Wvs[0], wv)
}

func (blnkr *Blnkr) makeOutWv(edx int, clr RGB) {
	wv := Wv{
		SD:   1.0,
		Dlta: 0.02,
		Xs:   4.0,
		Ys:   1.0,
		Clr:  clr,
	}
	wv.Mn = -wv.SD * 3.0 * wv.Xs
	blnkr.Wvs[edx] = append(blnkr.Wvs[edx], wv)
}

// Pdf - the probability density function, which describes the probability
// of a random variable taking on the value x
func (wv *Wv) Pdf(x float64) float64 {
	m := wv.SD * math.Sqrt(2*math.Pi)
	e := math.Exp(-math.Pow(x-wv.Mn, 2) / (2 * wv.SD * wv.SD))
	return e / m
}

// ColorAt - get color of wave at given radius
func (wv *Wv) ColorAt(x float64) RGB {
	dlta := math.Abs(wv.Mn - x) // get distance from wave mean to position
	dlta = dlta / wv.Xs         // divide distance by wave xscale
	y := wv.Pdf(dlta) * wv.Ys

	nwclr := wv.Clr.Dim(y)

	// if y > 0.1 {
	// 	fmt.Printf("{%.2f %v}", y, nwclr)
	// }

	return nwclr
}

func (blnkr *Blnkr) updateWvs() {

	// update waves
	for edx, wvs := range blnkr.Wvs {
		nwwvs := []Wv{}
		for _, wv := range wvs {
			wv.Mn += wv.Dlta // update mean (position) by dlta

			// check if wv is still in range of lmps
			mn := -wv.SD * 4.0 * wv.Xs
			mx := blnkr.Mxrs[edx] + (wv.SD * 4.0 * wv.Xs)
			if wv.Mn > mn && wv.Mn < mx {
				nwwvs = append(nwwvs, wv)
			}

			// log.Printf("wv %v: %v", i, wv)
		}
		blnkr.Wvs[edx] = nwwvs
	}

	// apply waves to lamp points
	for _, lmp := range blnkr.Lmps {
		for i := 0; i < LampSize; i++ {
			clr := RGB{} // start with zeroed color
			for edx, wvs := range blnkr.Wvs {
				for _, wv := range wvs {
					r := float64(0)
					if len(lmp.Pnts[i].Mres) > 0 {
						r = lmp.Pnts[i].Mres[edx]
					}
					wvclr := wv.ColorAt(r)
					clr = clr.Add(wvclr)
				}
				lmp.Pnts[i].Clr = clr
			}
		}
	}
}

// calculate distance between two vec3s
func distance(p, q mgl64.Vec3) float64 {
	dlta := q.Sub(p)
	return dlta.Len()
}
