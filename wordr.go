package main

import (
  "log"
  "time"
  "os"
  "encoding/json"
  "math/rand"
)

var WRD_POOL = []string{
  "analytical",
  "inquisitive",
  "fearless",
  "open-minded",
  "creative",
  "balanced",
  "experiential",
  "adventurous",
  "inclusive",
  "present",
  "disruptive",
  "thoughtful",
  "curious",
  "critical",
}

var POST_DELAY = int64(10000) // dataclient word cycle timeout

type Wrdr struct {
  Srcs []int
  Wrds []string
  LstWrds []string
  Stmps []int64
  Lgr *json.Encoder
}

// json record for logging word posts & touches
type WrdLg struct {
  Word    string  `json:"word"`
  Flavor  string  `json:"flavor"`
  Source  int     `json:"source"`
  Choice  string  `json:"choice"`
  Time    int64     `json:"time"`
}

func NewWrdr(srcs []int, lgf *os.File) Wrdr {
  wrdln := len(srcs) * 2
  w := Wrdr {
    Srcs: srcs[:],
    Wrds: make([]string, wrdln),
    LstWrds: make([]string, wrdln),
    Stmps: make([]int64, wrdln),
    Lgr:  json.NewEncoder(lgf),
  }

  for i := 0; i < wrdln; i++ {

    // set initial word values and time stamps
    wrd := w.PickWrd()
    stmp := NowMs()
    w.Wrds[i] = wrd
    w.Stmps[i] = stmp

    // log posted words
    w.LogPost(i, wrd, stmp)
  }

  return w
}

// return randomish word not in current words list from pool
func (w Wrdr) PickWrd() string {
  for {
    nwwrd := WRD_POOL[rand.Intn(len(WRD_POOL))] // pick random word from pool
    isnw := bool(true)

    for _, wrd := range w.Wrds { // check if word is displayed now
      if nwwrd == wrd {
        isnw = false
      }
    }
    if isnw {
      return nwwrd
    }
  }
}

func (w Wrdr) CycleWrd() DataMsg {
  wrddx := rand.Intn(len(w.Wrds)) // pick random vote station to cycle word for

  nwwrd := w.PickWrd()
  stmp := NowMs() + POST_DELAY // stamp in future after post delay
  w.LstWrds[wrddx] = w.Wrds[wrddx]
  w.Wrds[wrddx] = nwwrd
  w.Stmps[wrddx] = stmp
  w.LogPost(wrddx, nwwrd, stmp)

  nwsrc, nwchc := w.DeDex(wrddx)
  return DataMsg{
    Source: nwsrc, Flavor: "new_word", Choice: nwchc, Word: nwwrd}
}

func (w Wrdr) LogPost(wrddx int, nwwrd string, stmp int64) {
  src, chc := w.DeDex(wrddx)
  lg := WrdLg{
    Word: nwwrd,
    Flavor: "post",
    Source: src,
    Choice: chc,
    Time: stmp,
  }
  w.Lgr.Encode(&lg)
}

func (w Wrdr) LogTouch(src int, flvr string, chc string) {
  stmp := NowMs()
  wrddx := w.Dex(src, chc)
  if wrddx < 0 {
    log.Printf(
      "ERROR: failed to log touch from unexpected source '%v' '%v'", src, chc)
    return
  }
  wrd := w.Wrds[wrddx]
  if stmp < w.Stmps[wrddx] { // check if word has loaded yet
    wrd = w.LstWrds[wrddx] // if not register vote for last word
  }

  lg := WrdLg{
    Word: wrd,
    Flavor: flvr,
    Source: src,
    Choice: chc,
    Time: stmp,
  }
  w.Lgr.Encode(&lg)
}

func (w Wrdr) DeDex(wrddx int) (int, string) {
  src := w.Srcs[wrddx/2]
  chc := "right"
  if wrddx % 2 == 0 {
    chc = "left"
  }
  return src, chc
}

func (w Wrdr) Dex(src int, chc string) int {
  for i, s := range w.Srcs {
    if s == src {
      if chc == "left" {
        return 2 * i
      } else {
        return (2 * i) + 1
      }
    }
  }
  return -1
}

// current unix epoch time in ms
func NowMs() int64 {
  return time.Now().UnixNano() / 1000000
}
