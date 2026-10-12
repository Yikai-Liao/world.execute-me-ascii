package film

import (
	"strings"
	"testing"

	assets "github.com/Yikai-Liao/world.execute-me-ascii"
	"github.com/gdamore/tcell/v2"
)

func TestOriginalSubtitleBoundaries(t *testing.T) {
	f, err := Load(assets.FS)
	if err != nil {
		t.Fatal(err)
	}
	if f.Cue(0) != nil {
		t.Fatal("subtitle before the first cue")
	}
	for i := range f.Lyrics {
		cue := &f.Lyrics[i]
		if got := f.Cue(cue.Time); got != cue {
			t.Fatalf("cue %d does not start at its timestamp", i)
		}
		if got := f.Cue(cue.End); got == cue {
			t.Fatalf("cue %d remains visible at its exclusive end", i)
		}
	}
}

func TestSceneAndCaptionLayoutAtExtremeSizes(t *testing.T) {
	f, err := Load(assets.FS)
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range [][2]int{{1, 1}, {3, 2}, {20, 6}, {24, 8}, {40, 100}, {240, 12}, {128, 44}, {300, 100}} {
		for _, seconds := range []float64{0, 3.2, 5, 7, 17, 30, 35, 38, 42, 60, 80, 90, 115, 123, 135, 150, 160, 170, 180, 190, 210} {
			var c Canvas
			c.Resize(size[0], size[1])
			f.Render(&c, View{Time: seconds, CellAspect: .5, FPS: 60})
			rows := strings.Split(strings.TrimSuffix(c.Plain(), "\n"), "\n")
			if len(rows) != size[1] {
				t.Fatalf("%v at %g: wrong number of rows", size, seconds)
			}
			for i, row := range rows {
				if TextWidth(row) != size[0] {
					t.Fatalf("%v at %g row %d has width %d", size, seconds, i, TextWidth(row))
				}
			}
			if size[0] >= 24 && size[1] >= 18 {
				cue := f.Cue(seconds)
				if cue != nil {
					captions := strings.Join(rows[size[1]-5:size[1]-1], "\n")
					if !strings.Contains(captions, Crop(cue.ZH, size[0]-2)) && TextWidth(cue.ZH) <= size[0]-2 {
						t.Fatalf("scene overwrote Chinese caption at %g", seconds)
					}
				}
			}
		}
	}
}

func TestWideCharacterOverwrite(t *testing.T) {
	var c Canvas
	c.Resize(8, 1)
	c.Clear()
	c.Text(1, 0, "中文", White)
	c.Put(2, 0, 'x', Normal)
	c.Put(4, 0, 'y', Normal)
	if got := c.Plain(); got != "  x y   \n" {
		t.Fatalf("orphaned wide-character half: %q", got)
	}
	c.Clear()
	c.Text(1, 0, "中文", White)
	c.Put(2, 0, '好', Bright)
	if got := c.Plain(); got != "  好    \n" {
		t.Fatalf("overlapping wide characters: %q", got)
	}
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(8, 1)
	c.Show(screen)
	if r, _, _, w := screen.GetContent(2, 0); r != '好' || w != 2 {
		t.Fatalf("wide glyph lost at screen boundary: %q, width %d", r, w)
	}
}

func TestResizeReusesCanvas(t *testing.T) {
	var c Canvas
	c.Resize(120, 40)
	first := &c.Cells[0]
	c.Resize(120, 40)
	if &c.Cells[0] != first {
		t.Fatal("same-size frame allocated a new canvas")
	}
	c.Resize(40, 120)
	if c.W != 40 || c.H != 120 || len(c.Cells) != 4800 {
		t.Fatal("portrait resize lost dimensions")
	}
}

func BenchmarkRender(b *testing.B) {
	f, err := Load(assets.FS)
	if err != nil {
		b.Fatal(err)
	}
	var c Canvas
	c.Resize(240, 85)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Render(&c, View{Time: float64(i%12720) / 60, CellAspect: .5, FPS: 60})
	}
}
