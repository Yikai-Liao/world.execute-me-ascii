// Package audio owns decoding, transport controls and the estimated playback
// timeline. Oto exposes queued bytes, not a hardware presentation timestamp.
package audio

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"sync/atomic"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

type pcmSource struct {
	*bytes.Reader
	position atomic.Int64
}

func (s *pcmSource) Read(p []byte) (int, error) {
	n, err := s.Reader.Read(p)
	s.position.Add(int64(n))
	return n, err
}

func (s *pcmSource) Seek(offset int64, whence int) (int64, error) {
	n, err := s.Reader.Seek(offset, whence)
	if err == nil {
		s.position.Store(n)
	}
	return n, err
}

// Transport controls and Position are called by the UI goroutine. Only the PCM
// cursor is shared with Oto; all seeks go through Player.Seek for serialization.
type Audio struct {
	context            *oto.Context
	player             *oto.Player
	source             *pcmSource
	bytesPerSecond     int64
	duration           float64
	latency            float64
	playing            bool
	position, seekBase float64
	drainAt            time.Time
}

func Open(input io.Reader, latency time.Duration) (*Audio, error) {
	decoder, err := mp3.NewDecoder(input)
	if err != nil {
		return nil, fmt.Errorf("decode MP3: %w", err)
	}
	// This MV is short. Predecoding avoids decoder work and allocation during
	// playback, and makes chapter/caption seeks constant-time PCM offsets.
	pcm, err := io.ReadAll(io.LimitReader(decoder, 256<<20))
	if err != nil {
		return nil, fmt.Errorf("read PCM: %w", err)
	}
	if len(pcm) == 256<<20 {
		return nil, fmt.Errorf("decoded audio exceeds 256 MiB limit")
	}
	if len(pcm) < 4 {
		return nil, fmt.Errorf("empty audio")
	}
	pcm = pcm[:len(pcm)/4*4]
	rate := decoder.SampleRate()
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate: rate, ChannelCount: 2, Format: oto.FormatSignedInt16LE,
		BufferSize: latency, ApplicationName: "world.execute(me);",
	})
	if err != nil {
		return nil, fmt.Errorf("audio output: %w", err)
	}
	select {
	case <-ready:
	case <-time.After(8 * time.Second):
		return nil, fmt.Errorf("audio device initialization timed out")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	source := &pcmSource{Reader: bytes.NewReader(pcm)}
	player := ctx.NewPlayer(source)
	player.SetBufferSize(rate * 4 / 25) // 40 ms player buffer, separate from device buffer.
	player.SetVolume(.75)
	return &Audio{context: ctx, player: player, source: source, bytesPerSecond: int64(rate * 4), duration: float64(len(pcm)) / float64(rate*4), latency: latency.Seconds()}, nil
}

func (a *Audio) Duration() float64 { return a.duration }
func (a *Audio) Playing() bool     { return a.playing }

func (a *Audio) Position() float64 {
	if !a.playing {
		return a.position
	}
	// BufferedSize excludes the device/server queue. Requested device latency
	// is only an estimate; --audio-latency and subtitle offset allow calibration.
	read := a.source.position.Load()
	queued := a.player.BufferedSize()
	p := float64(read-int64(queued))/float64(a.bytesPerSecond) - a.latency
	a.position = max(a.position, max(a.seekBase, min(a.duration, p)))
	if !a.player.IsPlaying() {
		if a.drainAt.IsZero() {
			a.drainAt = time.Now().Add(time.Duration(a.latency * float64(time.Second)))
		}
		if !time.Now().Before(a.drainAt) {
			a.position = a.duration
			a.playing = false
		}
	} else {
		a.drainAt = time.Time{}
	}
	return a.position
}

func (a *Audio) Play() {
	if a.position >= a.duration {
		return
	}
	a.drainAt = time.Time{}
	a.player.Play()
	a.playing = true
}

func (a *Audio) Pause() error {
	p := a.Position()
	a.player.PauseAndStopReading()
	a.playing = false
	// Requeue from the estimated audible position so resuming doesn't skip
	// bytes previously handed to the backend. Accuracy depends on its latency.
	return a.Seek(p)
}

func (a *Audio) Seek(seconds float64) error {
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return fmt.Errorf("invalid seek time")
	}
	seconds = max(0, min(a.duration, seconds))
	wasPlaying := a.playing
	a.player.PauseAndStopReading()
	offset := int64(seconds*float64(a.bytesPerSecond)) / 4 * 4
	if _, err := a.player.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	a.position = float64(offset) / float64(a.bytesPerSecond)
	a.seekBase = a.position
	a.drainAt = time.Time{}
	a.playing = false
	if wasPlaying && a.position < a.duration {
		a.Play()
	}
	return nil
}

func (a *Audio) SetVolume(v float64) { a.player.SetVolume(max(0, min(1, v))) }
func (a *Audio) Err() error {
	if err := a.context.Err(); err != nil {
		return err
	}
	return a.player.Err()
}
func (a *Audio) Close() { a.player.PauseAndStopReading() }
