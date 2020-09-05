package nanovgo

import (
	"image/color"
)

type Paint struct {
	color color.Color
	image int
}

func (p *Paint) setPaintColor(color color.Color) {
	p.color = color
	p.image = 0
}
