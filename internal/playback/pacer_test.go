package playback

import (
	"testing"
	"time"
)

func TestPacerDropsExpiredDeadlines(t *testing.T) {
	p := NewPacer(60, false)
	start := time.Unix(100, 0)
	end := start.Add(200 * time.Millisecond)
	wait := p.Next(start, end)
	if wait <= 0 || wait > time.Second/60 {
		t.Fatalf("expired frame produced catch-up burst: %s", wait)
	}
	if p.FPS != 60 {
		t.Fatal("fixed FPS changed")
	}
	// A slight overrun should wait until the next cadence slot, without adding
	// an entire new period after the work that already ran.
	p = NewPacer(60, false)
	wait = p.Next(start, start.Add(20*time.Millisecond))
	if wait >= 15*time.Millisecond {
		t.Fatalf("overrun added unnecessary full-period delay: %s", wait)
	}
}

func TestAutoFPSNeedsSustainedPressureAndRecovers(t *testing.T) {
	p := NewPacer(60, true)
	start := time.Unix(100, 0)
	p.Next(start, start.Add(100*time.Millisecond))
	if p.FPS != 60 {
		t.Fatal("single slow frame changed the FPS")
	}
	for i := 0; i < 11; i++ {
		p.Next(start, start.Add(20*time.Millisecond))
	}
	if p.FPS != 45 {
		t.Fatalf("sustained slow frames: got %d FPS", p.FPS)
	}
	for i := 0; i < 225; i++ {
		p.Next(start, start.Add(time.Millisecond))
	}
	if p.FPS != 50 {
		t.Fatalf("stable fast frames did not recover gradually: %d", p.FPS)
	}
	for i := 0; i < 120; i++ {
		p.Next(start, start.Add(time.Second))
	}
	if p.FPS != 15 {
		t.Fatalf("auto FPS minimum: %d", p.FPS)
	}
}
