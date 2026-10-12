package film

import (
	"math"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

const (
	Dim = iota
	Normal
	Bright
	White
	Red
	Ghost
)

var styles = [...]tcell.Style{
	tcell.StyleDefault.Foreground(tcell.NewHexColor(0xaf875f)),
	tcell.StyleDefault.Foreground(tcell.NewHexColor(0xffaf5f)),
	tcell.StyleDefault.Foreground(tcell.NewHexColor(0xffd75f)).Bold(true),
	tcell.StyleDefault.Foreground(tcell.NewHexColor(0xffffd7)).Bold(true),
	tcell.StyleDefault.Foreground(tcell.NewHexColor(0xff5f5f)).Bold(true),
	tcell.StyleDefault.Foreground(tcell.NewHexColor(0x875f00)),
}

type Cell struct {
	Rune  rune
	Style uint8
}
type Rect struct{ X, Y, W, H int }
type Canvas struct {
	W, H  int
	Clip  Rect
	Cells []Cell
}

func (c *Canvas) Resize(w, h int) {
	if c.W == w && c.H == h {
		return
	}
	c.W, c.H = w, h
	c.Cells = make([]Cell, w*h)
	c.Clip = Rect{0, 0, w, h}
}

func (c *Canvas) Clear() {
	for i := range c.Cells {
		c.Cells[i] = Cell{' ', Dim}
	}
	c.Clip = Rect{0, 0, c.W, c.H}
}

func (c *Canvas) Put(x, y int, r rune, style int) {
	w := runeWidth(r)
	if w <= 0 || x < c.Clip.X || x+w > c.Clip.X+c.Clip.W || y < c.Clip.Y || y >= c.Clip.Y+c.Clip.H || x < 0 || y < 0 || x+w > c.W || y >= c.H {
		return
	}
	// Repair the other half if a scene overwrites a previous wide character.
	idx := y*c.W + x
	if c.Cells[idx].Rune == 0 && x > 0 {
		c.Cells[idx-1] = Cell{' ', Dim}
	}
	if runeWidth(c.Cells[idx].Rune) == 2 && x+1 < c.W {
		c.Cells[idx+1] = Cell{' ', Dim}
	}
	if w == 2 && runeWidth(c.Cells[idx+1].Rune) == 2 && x+2 < c.W {
		c.Cells[idx+2] = Cell{' ', Dim}
	}
	c.Cells[idx] = Cell{r, uint8(style)}
	if w == 2 {
		c.Cells[idx+1] = Cell{0, uint8(style)}
	}
}

func runeWidth(r rune) int {
	if r == 0 {
		return 0
	}
	return uniseg.StringWidth(string(r))
}
func TextWidth(s string) int { return uniseg.StringWidth(s) }
func Crop(s string, n int) string {
	used := 0
	for i, r := range s {
		used += runeWidth(r)
		if used > n {
			return s[:i]
		}
	}
	return s
}

func (c *Canvas) Text(x, y int, s string, style int) {
	for _, r := range s {
		c.Put(x, y, r, style)
		x += runeWidth(r)
	}
}

func (c *Canvas) Center(y int, s string, style int) {
	s = Crop(s, c.Clip.W)
	c.Text(c.Clip.X+(c.Clip.W-TextWidth(s))/2, y, s, style)
}

func (c *Canvas) Wrapped(s string, y, maxLines, style int) {
	w := max(1, c.W-2)
	for line := 0; line < maxLines && s != ""; line++ {
		part := Crop(s, w)
		if len(part) < len(s) {
			if at := strings.LastIndexByte(part, ' '); at > len(part)/2 {
				part = part[:at]
			}
		}
		c.Center(y+line, part, style)
		s = strings.TrimLeftFunc(s[len(part):], unicode.IsSpace)
	}
}

func (c *Canvas) Line(x0, y0, x1, y1 float64, ch rune, style int) {
	steps := max(1, int(math.Ceil(max(math.Abs(x1-x0), math.Abs(y1-y0)))))
	for i := 0; i <= steps; i++ {
		u := float64(i) / float64(steps)
		c.Put(int(math.Round(x0+(x1-x0)*u)), int(math.Round(y0+(y1-y0)*u)), ch, style)
	}
}

func (c *Canvas) Show(screen tcell.Screen) {
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			cell := c.Cells[y*c.W+x]
			if cell.Rune != 0 {
				screen.SetContent(x, y, cell.Rune, nil, styles[cell.Style])
			}
		}
	}
	screen.Show()
}

func (c *Canvas) Plain() string {
	var out strings.Builder
	out.Grow(len(c.Cells) + c.H)
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			if r := c.Cells[y*c.W+x].Rune; r != 0 {
				out.WriteRune(r)
			}
		}
		out.WriteByte('\n')
	}
	return out.String()
}
