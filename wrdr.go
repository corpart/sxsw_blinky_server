package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// Wrd - holds string and color for a word
type Wrd struct {
	Str string
	Clr RGB
}

// WrdPool - words to use for dataviz
var WrdPool = []Wrd{
	{"analytical", RGB{0xc90, 0x910, 0xd30}},
	{"inquisitive", RGB{0xc86, 0x4b0, 0xff0}},
	{"fearless", RGB{0xea0, 0x300, 0x400}},
	{"open-minded", RGB{0xf60, 0xe90, 0x370}},
	{"creative", RGB{0xff0, 0xaa0, 0x110}},
	{"balanced", RGB{0x220, 0xaa0, 0xdd0}},
	{"experiential", RGB{0x990, 0xbb0, 0xee0}},
	{"adventurous", RGB{0xff0, 0x550, 0x330}},
	{"inclusive", RGB{0xd90, 0x480, 0xd60}},
	{"present", RGB{0x000, 0xff0, 0x880}},
	{"disruptive", RGB{0xff0, 0x8d0, 0x8d0}},
	{"thoughtful", RGB{0x8f0, 0x310, 0x9a0}},
	{"curious", RGB{0x5c0, 0x330, 0xfb0}},
	{"critical", RGB{0x2c0, 0xfc0, 0xfd0}},
}

// PostDelay - dataclient word cycle timeout
var PostDelay = int64(10000)

// Wrdr - manages word cycling & vote logging
type Wrdr struct {
	Srcs    []int
	Wrds    []Wrd
	LstWrds []Wrd
	Stmps   []int64
	Lgr     *json.Encoder
}

// WrdLg - json record for logging word posts & touches
type WrdLg struct {
	Word   string `json:"word"`
	Flavor string `json:"flavor"`
	Source int    `json:"source"`
	Choice string `json:"choice"`
	Time   int64  `json:"time"`
}

// NewWrdr - init wrdr with list of vote station sources & logfile
func NewWrdr(srcs []int, lgf *os.File) Wrdr {
	wrdln := len(srcs) * 2
	w := Wrdr{
		Srcs:    srcs[:],
		Wrds:    make([]Wrd, wrdln),
		LstWrds: make([]Wrd, wrdln),
		Stmps:   make([]int64, wrdln),
		Lgr:     json.NewEncoder(lgf),
	}

	for i := 0; i < wrdln; i++ {

		// set initial word values and time stamps
		wrd := w.PickWrd()
		stmp := NowMs()
		w.Wrds[i] = wrd
		w.Stmps[i] = stmp

		// log posted words
		w.LogPost(i, wrd.Str, stmp)
	}

	return w
}

// PickWrd - return randomish word not in current words list from pool
func (w Wrdr) PickWrd() Wrd {
	for {
		nwwrd := WrdPool[rand.Intn(len(WrdPool))] // pick random word from pool
		isnw := bool(true)

		for _, wrd := range w.Wrds { // check if word is displayed now
			if nwwrd.Str == wrd.Str {
				isnw = false
			}
		}
		if isnw {
			return nwwrd
		}
	}
}

// CycleWrd - randomly change one of the current words & log change
func (w Wrdr) CycleWrd() DataMsg {
	wrddx := rand.Intn(len(w.Wrds)) // pick random vote station to cycle word for

	nwwrd := w.PickWrd()
	stmp := NowMs() + PostDelay // stamp in future after post delay
	w.LstWrds[wrddx] = w.Wrds[wrddx]
	w.Wrds[wrddx] = nwwrd
	w.Stmps[wrddx] = stmp
	w.LogPost(wrddx, nwwrd.Str, stmp)

	nwsrc, nwchc := w.DeDex(wrddx)
	return DataMsg{
		Source: nwsrc,
		Flavor: "new_word",
		Choice: nwchc,
		Word:   nwwrd.Str,
		Color:  []int{int(nwwrd.Clr[0]), int(nwwrd.Clr[1]), int(nwwrd.Clr[2])},
	}
}

// LogPost - write post event to json log file
func (w Wrdr) LogPost(wrddx int, nwwrd string, stmp int64) {
	src, chc := w.DeDex(wrddx)
	lg := WrdLg{
		Word:   nwwrd,
		Flavor: "post",
		Source: src,
		Choice: chc,
		Time:   stmp,
	}
	w.Lgr.Encode(&lg)
}

// LogTouch - write touch event to json log file
func (w Wrdr) LogTouch(src int, flvr string, chc string) (*Wrd, error) {
	stmp := NowMs()
	wrddx := w.Dex(src, chc)
	if wrddx < 0 {
		emsg := fmt.Sprintf(
			"failed to log touch from unexpected source '%v' '%v'", src, chc)
		return nil, errors.New(emsg)
	}
	wrd := w.Wrds[wrddx]
	if stmp < w.Stmps[wrddx] { // check if word has loaded yet
		wrd = w.LstWrds[wrddx] // if not register vote for last word
	}

	lg := WrdLg{
		Word:   wrd.Str,
		Flavor: flvr,
		Source: src,
		Choice: chc,
		Time:   stmp,
	}
	w.Lgr.Encode(&lg)

	return &wrd, nil
}

// DeDex - get vote station source address & side from index
func (w Wrdr) DeDex(wrddx int) (int, string) {
	src := w.Srcs[wrddx/2]
	chc := "right"
	if wrddx%2 == 0 {
		chc = "left"
	}
	return src, chc
}

// Dex - get vote station index from source address & side
func (w Wrdr) Dex(src int, chc string) int {
	for i, s := range w.Srcs {
		if s == src {
			if chc == "left" {
				return 2 * i
			}
			return (2 * i) + 1
		}
	}
	return -1
}

// NowMs - current unix epoch time in ms
func NowMs() int64 {
	return time.Now().UnixNano() / 1000000
}
