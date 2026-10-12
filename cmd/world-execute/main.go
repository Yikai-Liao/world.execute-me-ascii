package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"time"

	assets "github.com/Yikai-Liao/world.execute-me-ascii"
	"github.com/Yikai-Liao/world.execute-me-ascii/internal/audio"
	"github.com/Yikai-Liao/world.execute-me-ascii/internal/film"
	"github.com/Yikai-Liao/world.execute-me-ascii/internal/playback"
	"github.com/gdamore/tcell/v2"
)

type options struct {
	audio, fps, report                                 string
	start, offset, snapshot, aspect, stopAfter, volume float64
	autoplay, demo                                     bool
	width, height                                      int
	latency                                            time.Duration
}

func main() {
	var o options
	flag.StringVar(&o.audio, "audio", "", "MP3 path; default: bundled music or media/song.mp3")
	flag.StringVar(&o.fps, "fps", "auto", "auto (15-60) or a fixed FPS from 5 to 240")
	flag.Float64Var(&o.start, "start", 0, "start position in seconds")
	flag.Float64Var(&o.offset, "offset", math.NaN(), "subtitle offset in seconds")
	flag.Float64Var(&o.snapshot, "snapshot", -1, "print a scene snapshot at this time and exit")
	flag.Float64Var(&o.aspect, "cell-aspect", 0, "cell width / height; 0 detects pixels or falls back to 0.5")
	flag.Float64Var(&o.stopAfter, "stop-after", -1, "exit when the audio/demo timeline reaches this time")
	flag.Float64Var(&o.volume, "volume", .75, "initial volume from 0 to 1")
	flag.BoolVar(&o.autoplay, "autoplay", false, "start immediately")
	flag.BoolVar(&o.demo, "demo", false, "silent preview with a monotonic timeline, no audio device needed")
	flag.IntVar(&o.width, "width", 120, "snapshot width in cells")
	flag.IntVar(&o.height, "height", 40, "snapshot height in cells")
	flag.DurationVar(&o.latency, "audio-latency", 80*time.Millisecond, "requested output buffer and estimated output latency")
	flag.StringVar(&o.report, "report", "", "write frame timing and allocation report as JSON on exit")
	flag.Parse()
	if err := run(o); err != nil {
		fmt.Fprintln(os.Stderr, "播放失败:", err)
		os.Exit(1)
	}
}

type transport interface {
	Position() float64
	Duration() float64
	Playing() bool
	Play()
	Pause() error
	Seek(float64) error
	SetVolume(float64)
	Err() error
	Close()
}

// A silent preview explicitly uses wall time; normal playback uses audio.Audio.
type demoClock struct {
	position, duration float64
	since              time.Time
	playing            bool
}

func (d *demoClock) Position() float64 {
	if d.playing {
		p := d.position + time.Since(d.since).Seconds()
		if p >= d.duration {
			d.position = d.duration
			d.playing = false
			return d.duration
		}
		return p
	}
	return d.position
}
func (d *demoClock) Duration() float64 { return d.duration }
func (d *demoClock) Playing() bool     { return d.playing }
func (d *demoClock) Play() {
	if !d.playing && d.position < d.duration {
		d.since = time.Now()
		d.playing = true
	}
}
func (d *demoClock) Pause() error { d.position = d.Position(); d.playing = false; return nil }
func (d *demoClock) Seek(p float64) error {
	d.position = max(0, min(d.duration, p))
	d.since = time.Now()
	return nil
}
func (d *demoClock) SetVolume(float64) {}
func (d *demoClock) Err() error        { return nil }
func (d *demoClock) Close()            {}

func run(o options) error {
	fps, auto := 60, o.fps == "auto"
	if !auto {
		var err error
		fps, err = strconv.Atoi(o.fps)
		if err != nil || fps < 5 || fps > 240 {
			return fmt.Errorf("--fps must be auto or an integer from 5 to 240")
		}
	}
	for _, n := range []float64{o.start, o.snapshot, o.aspect, o.stopAfter, o.volume} {
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return fmt.Errorf("time and aspect arguments must be finite")
		}
	}
	if math.IsInf(o.offset, 0) || o.start < 0 || o.aspect < 0 || o.aspect > 4 {
		return fmt.Errorf("invalid start, offset or cell aspect")
	}
	if o.width < 1 || o.height < 1 || o.width > 1000 || o.height > 500 {
		return fmt.Errorf("snapshot dimensions must be 1..1000 by 1..500")
	}
	if o.latency < time.Millisecond || o.latency > time.Second {
		return fmt.Errorf("--audio-latency must be between 1ms and 1s")
	}
	if o.volume < 0 || o.volume > 1 {
		return fmt.Errorf("--volume must be between 0 and 1")
	}
	f, err := film.Load(assets.FS)
	if err != nil {
		return err
	}
	if math.IsNaN(o.offset) {
		o.offset = f.Config.Offset
	}
	aspect := o.aspect
	if aspect == 0 {
		aspect = .5
	}
	if o.snapshot >= 0 {
		var canvas film.Canvas
		canvas.Resize(o.width, o.height)
		f.Render(&canvas, film.View{Time: o.snapshot, Offset: o.offset, Paused: true, FPS: fps, CellAspect: aspect})
		_, err := fmt.Print(canvas.Plain())
		return err
	}
	var clock transport
	if o.demo {
		clock = &demoClock{duration: f.Config.Duration}
	} else {
		var input io.ReadCloser
		if o.audio != "" {
			input, err = os.Open(o.audio)
		} else {
			input, err = assets.FS.Open("bundled/song.mp3")
			if err != nil {
				input, err = os.Open(f.Config.Audio)
			}
		}
		if err != nil {
			return fmt.Errorf("music unavailable; use --audio /path/to/song.mp3 or --demo: %w", err)
		}
		engine, openErr := audio.Open(input, o.latency)
		input.Close()
		if openErr != nil {
			return openErr
		}
		clock = engine
	}
	defer clock.Close()
	// Use the decoded duration in normal mode; the original metadata can differ
	// slightly due to MP3 encoder padding.
	f.Config.Duration = clock.Duration()
	if err := clock.Seek(o.start); err != nil {
		return err
	}
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()
	screen.HideCursor()
	screen.Clear()
	events := make(chan tcell.Event, 32)
	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			e := screen.PollEvent()
			if e == nil {
				return
			}
			select {
			case events <- e:
			case <-done:
				return
			}
		}
	}()
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)
	view := film.View{Offset: o.offset, Ready: !o.autoplay, Paused: !o.autoplay, CellAspect: aspect, FPS: fps}
	volume := o.volume
	clock.SetVolume(volume)
	if o.autoplay {
		clock.Play()
	}
	pacer := playback.NewPacer(fps, auto)
	var canvas film.Canvas
	stats := newReport(o.demo)
	if o.report != "" {
		defer func() {
			if e := stats.save(o.report); e != nil {
				fmt.Fprintln(os.Stderr, "report:", e)
			}
		}()
	}
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-interrupts:
			return nil
		case event := <-events:
			switch e := event.(type) {
			case *tcell.EventResize:
				w, h := e.Size()
				pw, ph := e.PixelSize()
				if o.aspect == 0 && w > 0 && h > 0 && pw > 0 && ph > 0 {
					view.CellAspect = float64(pw) * float64(h) / (float64(ph) * float64(w))
				}
			case *tcell.EventKey:
				quit, err := handleKey(e, clock, f, &view, &volume)
				if err != nil {
					return err
				}
				if quit {
					return nil
				}
			case *tcell.EventError:
				return e
			}
		case <-timer.C:
			start := time.Now()
			view.Time = clock.Position()
			view.Paused = !clock.Playing()
			view.FPS = pacer.FPS
			if err := clock.Err(); err != nil {
				return err
			}
			w, h := screen.Size()
			// A zero-size minimized terminal should not allocate or stop music.
			if w > 0 && h > 0 {
				canvas.Resize(w, h)
				f.Render(&canvas, view)
				canvas.Show(screen)
			}
			end := time.Now()
			stats.record(end.Sub(start), view.Time, pacer.FPS, w, h)
			if o.stopAfter >= 0 && view.Time >= o.stopAfter {
				return nil
			}
			timer.Reset(pacer.Next(start, end))
		}
	}
}

func handleKey(e *tcell.EventKey, clock transport, f *film.Film, v *film.View, volume *float64) (bool, error) {
	seekAndPlay := func(t float64) error {
		if err := clock.Seek(t); err != nil {
			return err
		}
		v.Ready = false
		clock.Play()
		return nil
	}
	switch e.Key() {
	case tcell.KeyEscape, tcell.KeyCtrlC:
		return true, nil
	case tcell.KeyLeft:
		return false, clock.Seek(clock.Position() - 5)
	case tcell.KeyRight:
		return false, clock.Seek(clock.Position() + 5)
	case tcell.KeyEnter:
		return false, toggle(clock, v)
	case tcell.KeyRune:
		r := e.Rune()
		switch r {
		case 'q', 'Q':
			return true, nil
		case ' ':
			return false, toggle(clock, v)
		case 'r', 'R':
			return false, seekAndPlay(0)
		case 'h', 'H':
			v.Help = !v.Help
		case '[':
			v.Offset = math.Round((v.Offset+.1)*10) / 10
		case ']':
			v.Offset = math.Round((v.Offset-.1)*10) / 10
		case ',':
			return false, seekAndPlay(f.AdjacentCue(clock.Position(), -1))
		case '.':
			return false, seekAndPlay(f.AdjacentCue(clock.Position(), 1))
		case '+', '=':
			*volume = min(1, *volume+.05)
			clock.SetVolume(*volume)
		case '-':
			*volume = max(0, *volume-.05)
			clock.SetVolume(*volume)
		default:
			if r >= '1' && r <= '5' {
				return false, seekAndPlay(film.Chapters[int(r-'1')].Time)
			}
		}
	}
	return false, nil
}

func toggle(clock transport, v *film.View) error {
	v.Ready = false
	if clock.Playing() {
		return clock.Pause()
	}
	if clock.Position() >= clock.Duration() {
		if err := clock.Seek(0); err != nil {
			return err
		}
	}
	clock.Play()
	return nil
}

type report struct {
	Frames         int     `json:"frames"`
	LastTime       float64 `json:"last_time"`
	FPS            int     `json:"target_fps"`
	Width, Height  int
	Demo           bool    `json:"demo"`
	MaxMS          float64 `json:"max_frame_ms"`
	P95MS          float64 `json:"p95_frame_ms"`
	AllocatedBytes uint64  `json:"allocated_bytes"`
	GCCycles       uint32  `json:"gc_cycles"`
	durations      []float64
	before         runtime.MemStats
}

func newReport(demo bool) *report {
	r := &report{Demo: demo, durations: make([]float64, 0, 4096)}
	runtime.ReadMemStats(&r.before)
	return r
}
func (r *report) record(cost time.Duration, t float64, fps, w, h int) {
	r.Frames++
	r.LastTime = t
	r.FPS = fps
	r.Width = w
	r.Height = h
	ms := float64(cost) / float64(time.Millisecond)
	r.MaxMS = max(r.MaxMS, ms)
	if len(r.durations) < cap(r.durations) {
		r.durations = append(r.durations, ms)
	} else {
		r.durations[(r.Frames-1)%len(r.durations)] = ms
	}
}
func (r *report) save(path string) error {
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	r.AllocatedBytes = after.TotalAlloc - r.before.TotalAlloc
	r.GCCycles = after.NumGC - r.before.NumGC
	sort.Float64s(r.durations)
	if len(r.durations) > 0 {
		r.P95MS = r.durations[min(len(r.durations)-1, int(float64(len(r.durations))*.95))]
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}
