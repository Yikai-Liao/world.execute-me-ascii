// Package playback schedules drawing independently of the audio timeline.
package playback

import "time"

type Pacer struct {
	FPS        int
	auto       bool
	slow, fast int
	deadline   time.Time
}

func NewPacer(fps int, auto bool) *Pacer { return &Pacer{FPS: fps, auto: auto} }

// Next drops expired deadlines rather than trying to render a backlog. The
// measured cost includes terminal writes, but cannot measure display presentation.
func (p *Pacer) Next(start, end time.Time) time.Duration {
	period := time.Second / time.Duration(p.FPS)
	if p.auto {
		cost := end.Sub(start)
		if cost > period*9/10 {
			p.slow++
			p.fast = 0
		} else if cost < period/2 {
			p.fast++
			p.slow = 0
		} else {
			p.slow = 0
			p.fast = 0
		}
		if p.slow >= 12 && p.FPS > 15 {
			p.FPS = max(15, p.FPS*3/4)
			p.slow = 0
			p.deadline = time.Time{}
		} else if p.fast >= p.FPS*5 && p.FPS < 60 {
			p.FPS = min(60, p.FPS+5)
			p.fast = 0
			p.deadline = time.Time{}
		}
		period = time.Second / time.Duration(p.FPS)
	}
	if p.deadline.IsZero() {
		p.deadline = start
	}
	p.deadline = p.deadline.Add(period)
	if !p.deadline.After(end) {
		missed := end.Sub(p.deadline)/period + 1
		p.deadline = p.deadline.Add(missed * period)
	}
	return p.deadline.Sub(end)
}
