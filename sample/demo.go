package main

import (
	"fmt"
	"github.com/shibukawa/nanovgo"
	"math"

	"log"
)

type DemoData struct {
	fontNormal, fontBold, fontIcons int
	images                          [12]int
}

func (d *DemoData) loadData(vg *nanovgo.Context) {
	for i := 0; i < 12; i++ {
		path := fmt.Sprintf("images/image%d.jpg", i+1)
		d.images[i] = vg.CreateImage(path, 0)
		if d.images[i] == 0 {
			log.Fatalf("Could not load %s", path)
		}
	}

	d.fontIcons = vg.CreateFont("icons", "entypo.ttf")
	if d.fontIcons == -1 {
		log.Fatalln("Could not add font icons.")
	}
	d.fontNormal = vg.CreateFont("sans", "Roboto-Regular.ttf")
	if d.fontNormal == -1 {
		log.Fatalln("Could not add font italic.")
	}
	d.fontBold = vg.CreateFont("sans-bold", "Roboto-Bold.ttf")
	if d.fontBold == -1 {
		log.Fatalln("Could not add font bold.")
	}
}

func (d *DemoData) freeData(vg *nanovgo.Context) {
	for _, img := range d.images {
		vg.DeleteImage(img)
	}
}

func cosF(a float32) float32 {
	return float32(math.Cos(float64(a)))
}
func sinF(a float32) float32 {
	return float32(math.Sin(float64(a)))
}

func drawEyes(ctx *nanovgo.Context, x, y, w, h, mx, my, t float32) {
	ex := w * 0.23
	ey := h * 0.5
	lx := x + ex
	ly := y + ey
	rx := x + w - ex
	ry := y + ey
	var dx, dy, d, br float32
	if ex < ey {
		br = ex * 0.5
	} else {
		br = ey * 0.5
	}
	blink := float32(1 - math.Pow(math.Sqrt(float64(t)*0.5), 200)*0.8)

	bg1 := nanovgo.LinearGradient(x, y+h*0.5, x+w*0.1, y+h, nanovgo.RGBA(0, 0, 0, 32), nanovgo.RGBA(0, 0, 0, 16))
	ctx.BeginPath()
	ctx.Ellipse(lx+3.0, ly+16.0, ex, ey)
	ctx.Ellipse(rx+3.0, ry+16.0, ex, ey)
	ctx.SetFillPaint(bg1)
	ctx.Fill()

	bg2 := nanovgo.LinearGradient(x, y+h*0.25, x+w*0.1, y+h, nanovgo.RGBA(220, 220, 220, 255), nanovgo.RGBA(128, 128, 128, 255))
	ctx.BeginPath()
	ctx.Ellipse(lx, ly, ex, ey)
	ctx.Ellipse(rx, ry, ex, ey)
	ctx.SetFillPaint(bg2)
	ctx.Fill()

	dx = (mx - rx) / (ex * 10)
	dy = (my - ry) / (ey * 10)
	d = float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if d > 1.0 {
		dx /= d
		dy /= d
	}
	dx *= ex * 0.4
	dy *= ey * 0.5
	ctx.BeginPath()
	ctx.Ellipse(lx+dx, ly+dy+ey*0.25*(1-blink), br, br*blink)
	ctx.SetFillColor(nanovgo.RGBA(32, 32, 32, 255))
	ctx.Fill()

	dx = (mx - rx) / (ex * 10)
	dy = (my - ry) / (ey * 10)
	d = float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if d > 1.0 {
		dx /= d
		dy /= d
	}
	dx *= ex * 0.4
	dy *= ey * 0.5
	ctx.BeginPath()
	ctx.Ellipse(rx+dx, ry+dy+ey*0.25*(1-blink), br, br*blink)
	ctx.SetFillColor(nanovgo.RGBA(32, 32, 32, 255))
	ctx.Fill()
	dx = (mx - rx) / (ex * 10)
	dy = (my - ry) / (ey * 10)
	d = float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if d > 1.0 {
		dx /= d
		dy /= d
	}
	dx *= ex * 0.4
	dy *= ey * 0.5
	ctx.BeginPath()
	ctx.Ellipse(rx+dx, ry+dy+ey*0.25*(1-blink), br, br*blink)
	ctx.SetFillColor(nanovgo.RGBA(32, 32, 32, 255))
	ctx.Fill()

	gloss1 := nanovgo.RadialGradient(lx-ex*0.25, ly-ey*0.5, ex*0.1, ex*0.75, nanovgo.RGBA(255, 255, 255, 128), nanovgo.RGBA(255, 255, 255, 0))
	ctx.BeginPath()
	ctx.Ellipse(lx, ly, ex, ey)
	ctx.SetFillPaint(gloss1)
	ctx.Fill()

	gloss2 := nanovgo.RadialGradient(rx-ex*0.25, ry-ey*0.5, ex*0.1, ex*0.75, nanovgo.RGBA(255, 255, 255, 128), nanovgo.RGBA(255, 255, 255, 0))
	ctx.BeginPath()
	ctx.Ellipse(rx, ry, ex, ey)
	ctx.SetFillPaint(gloss2)
	ctx.Fill()

}

func drawLines(ctx *nanovgo.Context, x, y, w, h, t float32) {
	var pad float32 = 5.0
	s := w/9.0 - pad*2
	joins := []nanovgo.LineCap{nanovgo.MITER, nanovgo.ROUND, nanovgo.BEVEL}
	caps := []nanovgo.LineCap{nanovgo.BUTT, nanovgo.ROUND, nanovgo.SQUARE}

	ctx.Save()
	pts := []float32{
		-s*0.25 + cosF(t*0.3)*s*0.5,
		sinF(t*0.3) * s * 0.5,
		-s * 0.25,
		0,
		s * 0.25,
		0,
		s*0.25 + cosF(-t*0.3)*s*0.5,
		sinF(-t*0.3) * s * 0.5,
	}
	for i, cap := range caps {
		for j, join := range joins {
			fx := x + s*0.5 + float32(i*3+j)/9.0*w + pad
			fy := y - s*0.5 + pad

			ctx.SetLineCap(cap)
			ctx.SetLineJoin(join)

			ctx.SetStrokeWidth(s * 0.3)
			ctx.SetStrokeColor(nanovgo.RGBA(0, 0, 0, 160))
			ctx.BeginPath()
			ctx.MoveTo(fx+pts[0], fy+pts[1])
			ctx.LineTo(fx+pts[2], fy+pts[3])
			ctx.LineTo(fx+pts[4], fy+pts[5])
			ctx.LineTo(fx+pts[6], fy+pts[7])
			ctx.Stroke()

			ctx.SetLineCap(nanovgo.BUTT)
			ctx.SetLineJoin(nanovgo.BEVEL)

			ctx.SetStrokeWidth(1.0)
			ctx.SetStrokeColor(nanovgo.RGBA(0, 192, 255, 255))
			ctx.BeginPath()
			ctx.MoveTo(fx+pts[0], fy+pts[1])
			ctx.LineTo(fx+pts[2], fy+pts[3])
			ctx.LineTo(fx+pts[4], fy+pts[5])
			ctx.LineTo(fx+pts[6], fy+pts[7])
			ctx.Stroke()

		}
	}

	ctx.Restore()
}

func drawWidths(ctx *nanovgo.Context, x, y, width float32) {
	ctx.Save()
	ctx.SetStrokeColor(nanovgo.RGBA(255, 127, 255, 255))
	for i := 0; i < 20; i++ {
		w := (float32(i) + 0.5) * 0.1
		ctx.SetStrokeWidth(w)
		ctx.BeginPath()
		ctx.MoveTo(x, y)
		ctx.LineTo(x+width, y+width*0.3)
		ctx.Stroke()
		y += 10
	}
	ctx.Restore()
}

func drawCaps(ctx *nanovgo.Context, x, y, width float32) {
	caps := []nanovgo.LineCap{nanovgo.BUTT, nanovgo.ROUND, nanovgo.SQUARE}
	var lineWidth float32 = 0.8

	ctx.Save()
	ctx.BeginPath()
	ctx.Rect(x-lineWidth/2, y, width+lineWidth, 40)
	ctx.SetFillColor(nanovgo.RGBA(255, 255, 255, 32))
	ctx.Fill()

	ctx.BeginPath()
	ctx.Rect(x, y, width, 40)
	ctx.SetFillColor(nanovgo.RGBA(255, 255, 255, 32))
	ctx.Fill()

	ctx.SetStrokeWidth(lineWidth)

	for i, cap := range caps {
		ctx.SetLineCap(cap)
		ctx.SetStrokeColor(nanovgo.RGBA(0, 0, 0, 255))
		ctx.BeginPath()
		ctx.MoveTo(x, y+float32(i)*10+5)
		ctx.LineTo(x+width, y+float32(i)*10+5)
		ctx.Stroke()
	}
	ctx.Restore()
}