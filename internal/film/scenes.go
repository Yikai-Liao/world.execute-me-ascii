package film

import (
	"fmt"
	"math"
)

const tau = 2 * math.Pi

func hash16(i uint32) uint32 {
	v := i + 0x9e3779b9
	v = (v ^ (v >> 16)) * 0x7feb352d
	v = (v ^ (v >> 15)) * 0x846ca68b
	return (v ^ (v >> 16)) & 65535
}

// Scene coordinates are measured in cell heights. This keeps geometric shapes
// proportional on wide, tall and square terminals instead of stretching axes.
type viewport struct{ cx, cy, sx, sy float64 }

func fit(area Rect, aspect float64) viewport {
	if aspect <= 0 {
		aspect = .5
	}
	scale := max(.1, min(float64(area.W)*aspect/2.4, float64(area.H)/2.4))
	return viewport{float64(area.X) + float64(area.W-1)/2, float64(area.Y) + float64(area.H-1)/2, scale / aspect, scale}
}

func (v viewport) point(x, y float64) (float64, float64) { return v.cx + x*v.sx, v.cy + y*v.sy }

func drawScene(c *Canvas, a Rect, t, aspect, pulse float64) {
	v := fit(a, aspect)
	switch {
	case t < 2.92:
		messages := []string{"BIOS v2.1.4 / SELF SYSTEM", "POWER ON SELF TEST", "Memory Test : 65536K OK", "Primary Master : SOUL.SYS", "Primary Slave : EMOTION.DAT", "IDENTITY : [YOU] detected", "RELATIONSHIP : initializing...", "SYSTEM READY"}
		for i, line := range messages {
			if t >= float64(i)*.17 && i < a.H {
				c.Text(a.X+1, a.Y+i, Crop(line, a.W-2), Normal)
			}
		}
	case t < 3.873:
		shield(c, v, t-2.92)
		c.Center(int(v.cy), "PROTECTION", White)
	case t < 6.38:
		pieces := []string{"[]", "{}", "<>", "//", "\\\\", "||", "==", "##"}
		for i, s := range pieces {
			u := max(0, min(1, (t-3.873-float64(i)*.15)/1.2))
			r := .7 + 1.4*math.Pow(1-u, 3)
			x, y := v.point(math.Cos(float64(i)*tau/8)*r, math.Sin(float64(i)*tau/8)*r)
			c.Text(int(x), int(y), s, Bright)
		}
	case t < 11.095:
		cube(c, v, t, min(1, (t-6.38)))
		c.Center(int(v.cy), "ENTITY: ME", White)
	case t < 15.8:
		cube(c, v, t, 1)
		c.Center(a.Y, "SIMULATION / INITIALIZED", Bright)
	case t < 29.709:
		title(c, v, a, t)
	case t < 33.412:
		cube(c, v, t, 1)
		c.Center(a.Y, "POINTS / DIMENSION", Bright)
	case t < 37.067:
		shield(c, v, 1)
		x, y := v.point(math.Cos(t)*.9, math.Sin(t)*.9)
		c.Line(v.cx, v.cy, x, y, '+', Bright)
		c.Center(a.Y, "CIRCLE / CIRCUMFERENCE", White)
	case t < 40.706:
		wave(c, v, t, pulse)
		c.Center(a.Y, "SINE WAVE / TANGENTS", White)
	case t < 44.452:
		infinity(c, v, t)
		c.Center(a.Y, "LIMIT -> INFINITY", White)
	case t < 47.672:
		wave(c, v, t, pulse)
		c.Center(a.Y, "AC -> DC", White)
	case t < 59.223:
		infinity(c, v, t)
	case t < 74.045:
		heart(c, v, t, pulse)
		c.Center(a.Y, "STIMULATION / SATISFACTION", White)
	case t < 85.078:
		// The full organic choreography remains in scenes.py. These native
		// silhouettes exercise scaling without baking fixed-size Python frames.
		symbol := "EGGPLANT"
		if t >= 77.5 {
			symbol = "TOMATO"
		}
		if t >= 81.4 {
			symbol = "CAT : /\\_/\\"
		}
		shield(c, v, 1)
		c.Center(int(v.cy), symbol, White)
	case t < 88.587:
		cube(c, v, t, 1)
		c.Center(int(v.cy), "GOD / EXISTENCE", White)
	case t < 103.489:
		shield(c, v, 1)
		c.Center(int(v.cy), "IDENTITY REWRITE", White)
		c.Center(int(v.cy)+2, "F -> M   AM -> PM   S -> M", Bright)
	case t < 110.9:
		wave(c, v, t, pulse)
		c.Center(a.Y, "VIBRATION / COMPLETION", White)
	case t < 118.333:
		isolation(c, v, t-110.9)
	case t < 125.708:
		cube(c, v, t, max(0, 1-(t-118.333)/7.375))
		c.Center(a.Y, "ERASING FRAGMENTS...", Red)
	case t < 147.66:
		execution(c, a, t, "ILLEGAL ARGUMENT")
	case t < 158.9:
		execution(c, a, t, "EXECUTION")
	case t < 162.632:
		counts := []string{"ONE / 一", "TWO / 二", "THREE / 三", "FOUR / 四", "FIVE / 五", "SIX / 六"}
		c.Center(int(v.cy), counts[min(5, int((t-158.9)*6/3.732))], White)
	case t < 177.246:
		execution(c, a, t, "ONLY EXECUTION")
	case t < 192.5:
		heart(c, v, t, pulse)
		c.Center(a.Y, "LOVE = YOU + ME", White)
		if t >= 188.483 {
			for x := a.X + 2; x < a.X+a.W; x += 6 {
				c.Line(float64(x), float64(a.Y), float64(x), float64(a.Y+a.H-1), '|', Ghost)
			}
			c.Center(int(v.cy), "TRAPPED IN LOVE", Bright)
		}
	default:
		infinity(c, v, t)
		c.Center(int(v.cy), "WAITING FOR YOU...", White)
		c.Center(a.Y+a.H-1, "PROCESS ENDED. THE LOOP REMAINS.", Dim)
	}
	if t >= 110.9 && t < 192.5 {
		glitch(c, a, t)
	}
}

func shield(c *Canvas, v viewport, elapsed float64) {
	for ring := 0; ring < 4; ring++ {
		r := (.4 + float64(ring)*.2) * max(0, min(1, elapsed/.95))
		for i := 0; i < 160; i++ {
			u := float64(i) * tau / 160
			x, y := v.point(math.Cos(u)*r, math.Sin(u)*r)
			c.Put(int(math.Round(x)), int(math.Round(y)), '.', Normal)
		}
	}
}

func cube(c *Canvas, v viewport, t, reveal float64) {
	var points [8][3]float64
	for i := range points {
		x, y, z := -.65, -.65, -.65
		if i&1 != 0 {
			x = .65
		}
		if i&2 != 0 {
			y = .65
		}
		if i&4 != 0 {
			z = .65
		}
		x, z = x*math.Cos(t*.37)+z*math.Sin(t*.37), z*math.Cos(t*.37)-x*math.Sin(t*.37)
		y, z = y*math.Cos(t*.23)-z*math.Sin(t*.23), y*math.Sin(t*.23)+z*math.Cos(t*.23)
		px, py := v.point(x*3.7/(3.7+z), y*3.7/(3.7+z))
		points[i] = [3]float64{px, py, z}
	}
	count := 0
	for i, p := range points {
		for j := i + 1; j < 8; j++ {
			k := i ^ j
			if k&(k-1) != 0 {
				continue
			}
			count++
			if float64(count) > reveal*12 {
				continue
			}
			q := points[j]
			style := Normal
			if p[2]+q[2] > 0 {
				style = Ghost
			}
			c.Line(p[0], p[1], q[0], q[1], '.', style)
		}
	}
}

func wave(c *Canvas, v viewport, t, pulse float64) {
	for i := 0; i < 240; i++ {
		u := float64(i)/239*2 - 1
		x, y := v.point(u, math.Sin(u*tau+t*3)*(.3+pulse*.4))
		c.Put(int(math.Round(x)), int(math.Round(y)), '~', Bright)
	}
	c.Line(v.cx-v.sx, v.cy, v.cx+v.sx, v.cy, '-', Ghost)
}

func infinity(c *Canvas, v viewport, t float64) {
	for i := 0; i < 260; i++ {
		u := float64(i) * tau / 260
		x, y := v.point(math.Cos(u), math.Sin(2*u)*.45)
		style := Normal
		if math.Sin(u-t*2) > .8 {
			style = White
		}
		c.Put(int(math.Round(x)), int(math.Round(y)), '*', style)
	}
}

func heart(c *Canvas, v viewport, t, pulse float64) {
	scale := .045 * (1 + pulse*.18 + math.Sin(t*4)*.035)
	for i := 0; i < 240; i++ {
		u := float64(i) * tau / 240
		x := 16 * math.Pow(math.Sin(u), 3) * scale
		y := -(13*math.Cos(u) - 5*math.Cos(2*u) - 2*math.Cos(3*u) - math.Cos(4*u)) * scale
		px, py := v.point(x, y)
		c.Put(int(math.Round(px)), int(math.Round(py)), '#', Bright)
	}
}

func isolation(c *Canvas, v viewport, elapsed float64) {
	d := min(.85, elapsed*.12)
	x0, y0 := v.point(-d, 0)
	x1, y1 := v.point(d, 0)
	if elapsed < 3 {
		c.Line(x0, y0, x1, y1, '-', Ghost)
	} else {
		c.Center(int(v.cy)+2, "CONNECTION LOST", Red)
	}
	c.Text(int(x0)-2, int(y0), "[ME]", Bright)
	c.Text(int(x1)-2, int(y1), "[YOU]", Dim)
}

func execution(c *Canvas, a Rect, t float64, label string) {
	for row := 0; row < a.H; row += 2 {
		c.Text(a.X+1, a.Y+row, Crop(fmt.Sprintf("%04d > %s();", int(t*12)+row, label), a.W-2), Red)
	}
}

func title(c *Canvas, v viewport, a Rect, t float64) {
	cube(c, v, t, 1)
	c.Center(int(v.cy)-1, "W O R L D", White)
	c.Center(int(v.cy)+1, "EXECUTE(ME);", Bright)
	c.Center(a.Y+a.H-1, "M I L I", White)
}

func glitch(c *Canvas, a Rect, t float64) {
	intensity := min(1, .26+(t-110.9)/90)
	tick := uint32(t * 24)
	for band := 0; band < int(intensity*6); band++ {
		y := a.Y + int(hash16(tick+uint32(band*41)))%max(1, a.H)
		x := a.X + int(hash16(tick+uint32(band*131)))%max(1, a.W)
		for dx := 0; dx < int(intensity*14); dx++ {
			chars := "01/:#_"
			c.Put(x+dx, y, rune(chars[hash16(tick+uint32(dx))%6]), Red)
		}
	}
}
