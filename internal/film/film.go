package film

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"sort"
)

type Cue struct {
	Time float64 `json:"time"`
	End  float64 `json:"end"`
	EN   string  `json:"en"`
	ZH   string  `json:"zh"`
}

type Chapter struct {
	Time  float64
	Title string
}

var Chapters = []Chapter{
	{0, "01 / CREATION"}, {29.709, "02 / DEVOTION"},
	{110.9, "03 / ISOLATION"}, {125.708, "04 / EXECUTION"},
	{177.246, "05 / LOVE"},
}

type Film struct {
	Lyrics   []Cue
	Spectrum struct {
		FPS    float64     `json:"fps"`
		Bands  int         `json:"bands"`
		Frames [][]float64 `json:"frames"`
	}
	Config struct {
		Audio    string  `json:"audio"`
		Duration float64 `json:"duration"`
		Offset   float64 `json:"subtitle_offset"`
	}
}

func Load(files fs.FS) (*Film, error) {
	f := new(Film)
	for _, item := range []struct {
		name  string
		value any
	}{
		{"lyrics.json", &f.Lyrics}, {"spectrum.json", &f.Spectrum}, {"config.json", &f.Config},
	} {
		data, err := fs.ReadFile(files, item.name)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, item.value); err != nil {
			return nil, fmt.Errorf("%s: %w", item.name, err)
		}
	}
	if f.Config.Duration <= 0 || f.Spectrum.FPS <= 0 || len(f.Spectrum.Frames) == 0 {
		return nil, fmt.Errorf("invalid duration or spectrum metadata")
	}
	for i, cue := range f.Lyrics {
		if cue.End < cue.Time || (i > 0 && cue.Time < f.Lyrics[i-1].Time) {
			return nil, fmt.Errorf("invalid subtitle timeline at cue %d", i)
		}
	}
	for _, frame := range f.Spectrum.Frames {
		if len(frame) == 0 {
			return nil, fmt.Errorf("empty spectrum frame")
		}
	}
	return f, nil
}

func (f *Film) Cue(t float64) *Cue {
	i := sort.Search(len(f.Lyrics), func(i int) bool { return f.Lyrics[i].Time > t }) - 1
	if i >= 0 && t < f.Lyrics[i].End {
		return &f.Lyrics[i]
	}
	return nil
}

func (f *Film) AdjacentCue(t float64, direction int) float64 {
	i := sort.Search(len(f.Lyrics), func(i int) bool { return f.Lyrics[i].Time > t+.03 }) - 1
	i = max(0, min(len(f.Lyrics)-1, i+direction))
	if len(f.Lyrics) == 0 {
		return 0
	}
	return f.Lyrics[i].Time
}

func (f *Film) energy(t float64) []float64 {
	i := max(0, min(len(f.Spectrum.Frames)-1, int(t*f.Spectrum.FPS)))
	return f.Spectrum.Frames[i]
}

type View struct {
	Time, Offset        float64
	Ready, Paused, Help bool
	FPS                 int
	CellAspect          float64 // physical cell width / height
}

// Render reuses a caller-owned canvas. Scenes only receive their available area,
// so they cannot overwrite captions when the terminal becomes short or narrow.
func (f *Film) Render(c *Canvas, v View) {
	c.Clear()
	if c.W < 4 || c.H < 2 {
		c.Text(0, 0, "MV", Bright)
		return
	}
	state := "RUNNING"
	if v.Paused {
		state = "PAUSED"
	}
	if v.Ready {
		state = "READY"
	}
	clock := fmt.Sprintf("%02d:%02d / %02d:%02d %s", int(v.Time)/60, int(v.Time)%60, int(f.Config.Duration)/60, int(f.Config.Duration)%60, state)
	c.Text(1, 0, "WORLD.EXECUTE(ME);", Bright)
	if c.W >= 60 {
		c.Text(c.W-TextWidth(clock)-1, 0, clock, Dim)
	}
	c.Center(c.H-1, "SPACE play/pause  arrows seek  Q quit  H help", Dim)
	if c.H < 8 || c.W < 24 {
		c.Center(1, clock, Normal)
		if e := f.Cue(v.Time + v.Offset); e != nil && c.H >= 4 {
			c.Center(c.H-2, e.ZH, White)
		}
		return
	}
	chapter := 0
	for i, item := range Chapters {
		if v.Time >= item.Time {
			chapter = i
		}
	}
	c.Text(1, 1, Chapters[chapter].Title, Normal)
	if c.W >= 42 {
		c.Text(c.W-12, 1, fmt.Sprintf("%3d FPS", v.FPS), Dim)
	}
	// Caption rows expand for narrow windows, while leaving a usable scene area.
	maxLines := 1
	if c.H >= 18 {
		maxLines = 2
	}
	captionRows := 2 * maxLines
	spectrumY := c.H - captionRows - 2
	area := Rect{1, 3, c.W - 2, max(1, spectrumY-3)}
	e := f.Cue(v.Time + v.Offset)
	energy := f.energy(v.Time)
	pulse := 0.0
	for _, a := range energy[:min(10, len(energy))] {
		pulse += a
	}
	pulse /= float64(min(10, len(energy)))
	c.Clip = area
	if v.Ready {
		c.Center(area.Y+area.H/2-1, "EXECUTE(ME);", Bright)
		c.Center(area.Y+area.H/2+1, "[ SPACE / ENTER TO START ]", White)
	} else {
		drawScene(c, area, v.Time, v.CellAspect, pulse)
	}
	c.Clip = Rect{0, 0, c.W, c.H}
	bars := []rune("._:=|")
	for x := 1; x < c.W-1; x++ {
		a := energy[(x-1)*len(energy)/max(1, c.W-2)]
		c.Put(x, spectrumY, bars[max(0, min(4, int(math.Round(a*4))))], Dim)
	}
	captionY := spectrumY + 1
	if v.Ready {
		c.Center(captionY, "MILI / world.execute(me);", White)
	} else if e != nil {
		c.Wrapped(e.EN, captionY, maxLines, White)
		c.Wrapped(e.ZH, captionY+maxLines, maxLines, Bright)
	} else {
		c.Center(captionY, "[ instrumental / 间奏 ]", Dim)
	}
	if v.Help {
		drawHelp(c, v.Offset)
	}
}

func drawHelp(c *Canvas, offset float64) {
	lines := []string{"CONTROLS / 操作", "SPACE / ENTER: play / pause", "LEFT / RIGHT: seek 5 seconds", "R: restart    1-5: chapters", "[ / ]: subtitle offset", ", / .: previous / next cue", "+ / -: volume    Q / ESC: quit", fmt.Sprintf("Subtitle offset: %+.1fs", offset), "H: close help"}
	w, h := min(48, c.W), min(len(lines)+2, c.H)
	x, y := (c.W-w)/2, (c.H-h)/2
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			c.Put(xx, yy, ' ', Normal)
		}
	}
	for i := 0; i < min(len(lines), h-2); i++ {
		c.Text(x+1, y+i+1, Crop(lines[i], w-2), White)
	}
}
