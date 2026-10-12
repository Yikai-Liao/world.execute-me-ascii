package main

import (
	"math"
	"testing"

	assets "github.com/Yikai-Liao/world.execute-me-ascii"
	"github.com/Yikai-Liao/world.execute-me-ascii/internal/film"
	"github.com/gdamore/tcell/v2"
)

func TestTransportControls(t *testing.T) {
	f, err := film.Load(assets.FS)
	if err != nil {
		t.Fatal(err)
	}
	d := &demoClock{duration: f.Config.Duration}
	v := film.View{Ready: true}
	volume := .75
	key := func(k tcell.Key, r rune) {
		t.Helper()
		quit, err := handleKey(tcell.NewEventKey(k, r, 0), d, f, &v, &volume)
		if err != nil || quit {
			t.Fatalf("unexpected key result: quit=%t error=%v", quit, err)
		}
	}
	key(tcell.KeyRune, ' ')
	if v.Ready || !d.Playing() {
		t.Fatal("space did not leave the ready state")
	}
	key(tcell.KeyEnter, 0)
	if d.Playing() {
		t.Fatal("enter did not pause")
	}
	p := d.Position()
	key(tcell.KeyRight, 0)
	if math.Abs(d.Position()-p-5) > .001 || d.Playing() {
		t.Fatal("seek changed paused state or offset")
	}
	key(tcell.KeyRune, '4')
	if !d.Playing() || math.Abs(d.Position()-125.708) > .1 {
		t.Fatal("chapter jump did not play the correct chapter")
	}
	key(tcell.KeyRune, '[')
	if v.Offset != .1 {
		t.Fatal("subtitle offset direction changed")
	}
	key(tcell.KeyRune, 'h')
	if !v.Help {
		t.Fatal("help did not open")
	}
	key(tcell.KeyRune, 'r')
	if d.Position() > .1 {
		t.Fatal("restart did not reset timeline")
	}
	quit, err := handleKey(tcell.NewEventKey(tcell.KeyEscape, 0, 0), d, f, &v, &volume)
	if !quit || err != nil {
		t.Fatal("escape did not quit")
	}
}
