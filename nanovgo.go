package nanovgo

import (
	"image"
	"image/color"
	"image/draw"
	"log"

	"github.com/shibukawa/nanovgo/fontstashmini"
)

type Context struct {
	gl             *glContext
	commands       []float32
	commandX       float32
	commandY       float32
	states         []nvgState
	cache          nvgPathCache
	tessTol        float32
	distTol        float32
	fringeWidth    float32
	devicePxRatio  float32
	fs             *fontstashmini.FontStash
	fontImages     []int
	fontImageIdx   int
	drawCallCount  int
	fillTriCount   int
	strokeTriCount int
	textTriCount   int
}

// Delete is called when tearing down NanoVGo context
func (c *Context) Delete() {
	for i, fontImage := range c.fontImages {
		if fontImage != 0 {
			c.DeleteImage(fontImage)
			c.fontImages[i] = 0
		}
	}
	c.gl.renderDelete()
	c.gl = nil
}

// BeginFrame begins drawing a new frame
// Calls to NanoVGo drawing API should be wrapped in Context.BeginFrame() & Context.EndFrame()
// Context.BeginFrame() defines the size of the window to render to in relation currently
// set viewport (i.e. glViewport on GL backends). Device pixel ration allows to
// control the rendering on Hi-DPI devices.
// For example, GLFW returns two dimension for an opened window: window size and
// frame buffer size. In that case you would set windowWidth/Height to the window size
// devicePixelRatio to: frameBufferWidth / windowWidth.
func (c *Context) BeginFrame(windowWidth, windowHeight int, devicePixelRatio float32) {
	c.states = c.states[:0]
	c.Save()
	c.getState().reset()

	c.setDevicePixelRatio(devicePixelRatio)
	c.gl.renderViewport(windowWidth, windowHeight)

	c.drawCallCount = 0
	c.fillTriCount = 0
	c.strokeTriCount = 0
	c.textTriCount = 0
}

// EndFrame ends drawing flushing remaining render state.
func (c *Context) EndFrame() {
	c.gl.renderFlush()
	if c.fontImageIdx != 0 {
		fontImage := c.fontImages[c.fontImageIdx]
		if fontImage == 0 {
			return
		}
		iw, ih, _ := c.ImageSize(fontImage)
		j := 0
		for i := 0; i < c.fontImageIdx; i++ {
			nw, nh, _ := c.ImageSize(c.fontImages[i])
			if nw < iw || nh < ih {
				c.DeleteImage(c.fontImages[i])
			} else {
				c.fontImages[j] = c.fontImages[i]
				j++
			}
		}
		// make current font image to first
		c.fontImages[j] = c.fontImages[0]
		j++
		c.fontImages[0] = fontImage
		c.fontImageIdx = 0
		// clear all image after j
		for i := j; i < nvgMaxFontImages; i++ {
			c.fontImages[i] = 0
		}
	}
}

// Save pushes and saves the current render state into a state stack.
// A matching Restore() must be used to restore the state.
func (c *Context) Save() {
	if len(c.states) >= nvgMaxStates {
		return
	}
	if len(c.states) > 0 {
		c.states = append(c.states, c.states[len(c.states)-1])
	} else {
		c.states = append(c.states, nvgState{})
	}
}

// Restore pops and restores current render state.
func (c *Context) Restore() {
	nStates := len(c.states)
	if nStates > 1 {
		c.states = c.states[:nStates-1]
	}
}

// Block makes Save/Restore block.
func (c *Context) Block(block func()) {
	c.Save()
	defer c.Restore()
	block()
}

// SetStrokeWidth sets the stroke width of the stroke style.
func (c *Context) SetStrokeWidth(width float32) { c.getState().strokeWidth = width }

// SetTransformByValue premultiplies current coordinate system by specified matrix.
// The parameters are interpreted as matrix as follows:
//   [a c e]
//   [b d f]
//   [0 0 1]
func (c *Context) SetTransformByValue(a, b, cc, d, e, f float32) {
	t := TransformMatrix{a, b, cc, d, e, f}
	state := c.getState()
	state.xform = state.xform.PreMultiply(t)
}

// ResetTransform resets current transform to a identity matrix.
func (c *Context) ResetTransform() {
	state := c.getState()
	state.xform = IdentityMatrix()
}

// Translate translates current coordinate system.
func (c *Context) Translate(x, y float32) {
	state := c.getState()
	state.xform = state.xform.PreMultiply(TranslateMatrix(x, y))
}

// CurrentTransform returns the top part (a-f) of the current transformation matrix.
//   [a c e]
//   [b d f]
//   [0 0 1]
// There should be space for 6 floats in the return buffer for the values a-f.
func (c *Context) CurrentTransform() TransformMatrix {
	return c.getState().xform
}

// SetStrokeColor sets current stroke style to a solid color.
func (c *Context) SetStrokeColor(color color.Color) {
	c.getState().stroke.setPaintColor(color)
}

// SetFillColor sets current fill style to a solid color.
func (c *Context) SetFillColor(color color.Color) {
	c.getState().fill.setPaintColor(color)
}

func (c *Context) SetFillImage() {
	//c.getState().fill.image =
}

// CreateImageFromGoImage creates image by loading it from the specified image.Image object.
// Returns handle to the image.
func (c *Context) CreateImage(img image.Image) int {
	bounds := img.Bounds()
	size := bounds.Size()

	var rgba *image.RGBA

	switch i := img.(type) {
	case *image.RGBA:
		rgba = i
	default:
		rgba = image.NewRGBA(bounds)
		draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	}
	return c.gl.renderCreateTexture(nvgTextureRGBA, size.X, size.Y, rgba.Pix)
}

// ImageSize returns the dimensions of a created image.
func (c *Context) ImageSize(img int) (int, int, error) {
	return c.gl.renderGetTextureSize(img)
}

// DeleteImage deletes created image.
func (c *Context) DeleteImage(img int) {
	c.gl.renderDeleteTexture(img)
}

// Scissor sets the current scissor rectangle.
// The scissor rectangle is transformed by the current transform.
func (c *Context) Scissor(x, y, w, h float32) {
	state := c.getState()

	w = maxF(0.0, w)
	h = maxF(0.0, h)

	state.scissor.xform = TranslateMatrix(x+w*0.5, y+h*0.5).Multiply(state.xform)
	state.scissor.extent = [2]float32{w * 0.5, h * 0.5}
}

// IntersectScissor calculates intersects current scissor rectangle with the specified rectangle.
// The scissor rectangle is transformed by the current transform.
// Note: in case the rotation of previous scissor rect differs from
// the current one, the intersection will be done between the specified
// rectangle and the previous scissor rectangle transformed in the current
// transform space. The resulting shape is always rectangle.
func (c *Context) IntersectScissor(x, y, w, h float32) {
	state := c.getState()

	if state.scissor.extent[0] < 0 {
		c.Scissor(x, y, w, h)
		return
	}

	pXform := state.scissor.xform.Multiply(state.xform.Inverse())
	ex := state.scissor.extent[0]
	ey := state.scissor.extent[1]

	teX := ex * absF(pXform[0]) * ey * absF(pXform[2])
	teY := ex * absF(pXform[1]) * ey * absF(pXform[3])
	rect := intersectRects(pXform[4]-teX, pXform[5]-teY, teX*2, teY*2, x, y, w, h)
	c.Scissor(rect[0], rect[1], rect[2], rect[3])
}

// ResetScissor resets and disables scissoring.
func (c *Context) ResetScissor() {
	state := c.getState()

	state.scissor.xform = TransformMatrix{0, 0, 0, 0, 0, 0}
	state.scissor.extent = [2]float32{-1.0, -1.0}
}

// BeginPath clears the current path and sub-paths.
func (c *Context) BeginPath() {
	c.commands = c.commands[:0]
	c.cache.clearPathCache()
}

// Rect creates new rectangle shaped sub-path.
func (c *Context) Rect(x, y, w, h float32) {
	c.appendCommand([]float32{
		float32(nvgMOVETO), x, y,
		float32(nvgLINETO), x, y + h,
		float32(nvgLINETO), x + w, y + h,
		float32(nvgLINETO), x + w, y,
		float32(nvgCLOSE),
	})
}

// RoundedRect creates new rounded rectangle shaped sub-path.
func (c *Context) RoundedRect(x, y, w, h, r float32) {
	if r < 0.1 {
		c.Rect(x, y, w, h)
	} else {
		rx := minF(r, absF(w)*0.5) * signF(w)
		ry := minF(r, absF(h)*0.5) * signF(h)
		c.appendCommand([]float32{
			float32(nvgMOVETO), x, y + ry,
			float32(nvgLINETO), x, y + h - ry,
			float32(nvgBEZIERTO), x, y + h - ry*(1-Kappa90), x + rx*(1-Kappa90), y + h, x + rx, y + h,
			float32(nvgLINETO), x + w - rx, y + h,
			float32(nvgBEZIERTO), x + w - rx*(1-Kappa90), y + h, x + w, y + h - ry*(1-Kappa90), x + w, y + h - ry,
			float32(nvgLINETO), x + w, y + ry,
			float32(nvgBEZIERTO), x + w, y + ry*(1-Kappa90), x + w - rx*(1-Kappa90), y, x + w - rx, y,
			float32(nvgLINETO), x + rx, y,
			float32(nvgBEZIERTO), x + rx*(1-Kappa90), y, x, y + ry*(1-Kappa90), x, y + ry,
			float32(nvgCLOSE),
		})
	}
}

// Ellipse creates new ellipse shaped sub-path.
func (c *Context) Ellipse(cx, cy, rx, ry float32) {
	c.appendCommand([]float32{
		float32(nvgMOVETO), cx - rx, cy,
		float32(nvgBEZIERTO), cx - rx, cy + ry*Kappa90, cx - rx*Kappa90, cy + ry, cx, cy + ry,
		float32(nvgBEZIERTO), cx + rx*Kappa90, cy + ry, cx + rx, cy + ry*Kappa90, cx + rx, cy,
		float32(nvgBEZIERTO), cx + rx, cy - ry*Kappa90, cx + rx*Kappa90, cy - ry, cx, cy - ry,
		float32(nvgBEZIERTO), cx - rx*Kappa90, cy - ry, cx - rx, cy - ry*Kappa90, cx - rx, cy,
		float32(nvgCLOSE),
	})
}

// Circle creates new circle shaped sub-path.
func (c *Context) Circle(cx, cy, r float32) {
	c.Ellipse(cx, cy, r, r)
}

// ClosePath closes current sub-path with a line segment.
func (c *Context) ClosePath() {
	c.appendCommand([]float32{float32(nvgCLOSE)})
}

// PathWinding sets the current sub-path winding, see Winding.
func (c *Context) PathWinding(winding Winding) {
	c.appendCommand([]float32{float32(nvgWINDING), float32(winding)})
}

// DebugDumpPathCache prints cached path information to console
func (c *Context) DebugDumpPathCache() {
	log.Printf("Dumping %d cached paths\n", len(c.cache.paths))
	for i := 0; i < len(c.cache.paths); i++ {
		path := &c.cache.paths[i]
		log.Printf(" - Path %d\n", i)
		if len(path.fills) > 0 {
			log.Printf("   - fill: %d\n", len(path.fills))
			for _, fill := range path.fills {
				log.Printf("%f\t%f\n", fill.x, fill.y)
			}
		}
		if len(path.strokes) > 0 {
			log.Printf("   - strokes: %d\n", len(path.strokes))
			for _, stroke := range path.strokes {
				log.Printf("%f\t%f\n", stroke.x, stroke.y)
			}
		}
	}
}

// Fill fills the current path with current fill style.
func (c *Context) Fill() {
	state := c.getState()
	fillPaint := state.fill
	c.flattenPaths()

	if c.gl.edgeAntiAlias() {
		c.cache.expandFill(c.fringeWidth, Miter, 2.4, c.fringeWidth)
	} else {
		c.cache.expandFill(0.0, Miter, 2.4, c.fringeWidth)
	}

	c.gl.renderFill(&fillPaint, &state.scissor, c.fringeWidth, c.cache.bounds, c.cache.paths)

	// Count triangles
	for i := 0; i < len(c.cache.paths); i++ {
		path := &c.cache.paths[i]
		c.fillTriCount += len(path.fills) - 2
		c.strokeTriCount += len(path.strokes) - 2
		c.drawCallCount += 2
	}
}

// Stroke draws the current path with current stroke style.
func (c *Context) Stroke() {
	state := c.getState()
	scale := state.xform.getAverageScale()
	strokeWidth := clampF(state.strokeWidth*scale, 0.0, 200.0)
	strokePaint := state.stroke

	if strokeWidth < c.fringeWidth {
		// If the stroke width is less than pixel size, use alpha to emulate coverage.
		strokeWidth = c.fringeWidth
	}

	c.flattenPaths()
	for _, path := range c.cache.paths {
		if path.count == 1 {
			panic("")
		}
	}
	const miterLimit = 10 // TODO: remove
	const lineCap = Butt
	const lineJoin = Miter // or Round

	if c.gl.edgeAntiAlias() {
		c.cache.expandStroke(strokeWidth*0.5+c.fringeWidth*0.5, lineCap, lineJoin, miterLimit, c.fringeWidth, c.tessTol)
	} else {
		c.cache.expandStroke(strokeWidth*0.5, lineCap, lineJoin, miterLimit, c.fringeWidth, c.tessTol)
	}
	c.gl.renderStroke(&strokePaint, &state.scissor, c.fringeWidth, strokeWidth, c.cache.paths)

	// Count triangles
	for i := 0; i < len(c.cache.paths); i++ {
		path := &c.cache.paths[i]
		c.strokeTriCount += len(path.strokes) - 2
		c.drawCallCount += 2
	}
}

// CreateFont creates font by loading it from the disk from specified file name.
// Returns handle to the font.
func (c *Context) CreateFont(name, filePath string) int {
	return c.fs.AddFont(name, filePath)
}

// CreateFontFromMemory creates image by loading it from the specified memory chunk.
// Returns handle to the font.
func (c *Context) CreateFontFromMemory(name string, data []byte, freeData uint8) int {
	return c.fs.AddFontFromMemory(name, data, freeData)
}

// FindFont finds a loaded font of specified name, and returns handle to it, or -1 if the font is not found.
func (c *Context) FindFont(name string) int {
	return c.fs.GetFontByName(name)
}

// SetFontSize sets the font size of current text style.
func (c *Context) SetFontSize(size float32) { c.getState().fontSize = size }

// SetTextLetterSpacing sets the letter spacing of current text style.
func (c *Context) SetTextLetterSpacing(spacing float32) { c.getState().letterSpacing = spacing }

// SetTextLineHeight sets the line height of current text style.
func (c *Context) SetTextLineHeight(lineHeight float32) {
	c.getState().lineHeight = lineHeight
}

// SetTextAlign sets the text align of current text style.
func (c *Context) SetTextAlign(align Align) { c.getState().textAlign = align }

// SetFontFaceID sets the font face based on specified id of current text style.
func (c *Context) SetFontFaceID(font int) { c.getState().fontID = font }

// SetFontFace sets the font face based on specified name of current text style.
func (c *Context) SetFontFace(font string) { c.getState().fontID = c.fs.GetFontByName(font) }

// Text draws text string at specified location. If end is specified only the sub-string up to the end is drawn.
func (c *Context) Text(x, y float32, str string) float32 {
	return c.TextRune(x, y, []rune(str))
}

// TextRune is an alternate version of Text that accepts rune slice.
func (c *Context) TextRune(x, y float32, runes []rune) float32 {
	state := c.getState()
	scale := state.getFontScale() * c.devicePxRatio
	invScale := 1.0 / scale
	if state.fontID == fontstashmini.INVALID {
		return 0
	}

	c.fs.SetSize(state.fontSize * scale)
	c.fs.SetSpacing(state.letterSpacing * scale)
	c.fs.SetBlur(0)
	c.fs.SetAlign(fontstashmini.FONSAlign(state.textAlign))
	c.fs.SetFont(state.fontID)

	vertexCount := maxI(2, len(runes)) * 4 // conservative estimate.
	vertexes := c.cache.allocVertexes(vertexCount)

	iter := c.fs.TextIterForRunes(x*scale, y*scale, runes)
	prevIter := iter
	index := 0

	for {
		quad, ok := iter.Next()
		if !ok {
			break
		}
		if iter.PrevGlyph == nil || iter.PrevGlyph.Index == -1 {
			if !c.allocTextAtlas() {
				break // no memory :(
			}
			if index != 0 {
				c.renderText(vertexes[:index])
				index = 0
			}
			iter = prevIter
			quad, _ = iter.Next() // try again
			if iter.PrevGlyph == nil || iter.PrevGlyph.Index == -1 {
				// still can not find glyph?
				break
			}
		}
		prevIter = iter
		// Transform corners.
		c0, c1 := state.xform.TransformPoint(quad.X0*invScale, quad.Y0*invScale)
		c2, c3 := state.xform.TransformPoint(quad.X1*invScale, quad.Y0*invScale)
		c4, c5 := state.xform.TransformPoint(quad.X1*invScale, quad.Y1*invScale)
		c6, c7 := state.xform.TransformPoint(quad.X0*invScale, quad.Y1*invScale)
		//log.Printf("quad(%c) x0=%d, x1=%d, y0=%d, y1=%d, s0=%d, s1=%d, t0=%d, t1=%d\n", iter.CodePoint, int(quad.X0), int(quad.X1), int(quad.Y0), int(quad.Y1), int(1024*quad.S0), int(quad.S1*1024), int(quad.T0*1024), int(quad.T1*1024))
		// Create triangles
		if index+4 <= vertexCount {
			(&vertexes[index]).set(c2, c3, quad.S1, quad.T0)
			(&vertexes[index+1]).set(c0, c1, quad.S0, quad.T0)
			(&vertexes[index+2]).set(c4, c5, quad.S1, quad.T1)
			(&vertexes[index+3]).set(c6, c7, quad.S0, quad.T1)
			index += 4
		}
	}
	c.flushTextTexture()
	c.renderText(vertexes[:index])
	return iter.X
}

// TextBounds measures the specified text string. Parameter bounds should be a pointer to float[4],
// if the bounding box of the text should be returned. The bounds value are [xmin,ymin, xmax,ymax]
// Returns the horizontal advance of the measured text (i.e. where the next character should drawn).
// Measured values are returned in local coordinate space.
func (c *Context) TextBounds(x, y float32, str string) (float32, []float32) {
	state := c.getState()
	scale := state.getFontScale() * c.devicePxRatio
	invScale := 1.0 / scale
	if state.fontID == fontstashmini.INVALID {
		return 0, nil
	}

	c.fs.SetSize(state.fontSize * scale)
	c.fs.SetSpacing(state.letterSpacing * scale)
	c.fs.SetBlur(0)
	c.fs.SetAlign(fontstashmini.FONSAlign(state.textAlign))
	c.fs.SetFont(state.fontID)

	width, bounds := c.fs.TextBounds(x*scale, y*scale, str)
	if bounds != nil {
		bounds[1], bounds[3] = c.fs.LineBounds(y * scale)
		bounds[0] *= invScale
		bounds[1] *= invScale
		bounds[2] *= invScale
		bounds[3] *= invScale
	}
	return width * invScale, bounds
}

// TextMetrics returns the vertical metrics based on the current text style.
// Measured values are returned in local coordinate space.
func (c *Context) TextMetrics() (float32, float32, float32) {
	state := c.getState()
	scale := state.getFontScale() * c.devicePxRatio
	invScale := 1.0 / scale
	if state.fontID == fontstashmini.INVALID {
		return 0, 0, 0
	}

	c.fs.SetSize(state.fontSize * scale)
	c.fs.SetSpacing(state.letterSpacing * scale)
	c.fs.SetBlur(0)
	c.fs.SetAlign(fontstashmini.FONSAlign(state.textAlign))
	c.fs.SetFont(state.fontID)

	ascender, descender, lineH := c.fs.VerticalMetrics()
	return ascender * invScale, descender * invScale, lineH * invScale
}

func (c *Context) setDevicePixelRatio(ratio float32) {
	c.tessTol = 0.25 / ratio
	c.distTol = 0.01 / ratio
	c.fringeWidth = 1.0 / ratio
	c.devicePxRatio = ratio
}

func (c *Context) getState() *nvgState {
	return &c.states[len(c.states)-1]
}

func (c *Context) appendCommand(vals []float32) {
	xForm := c.getState().xform

	if nvgCommands(vals[0]) != nvgCLOSE && nvgCommands(vals[0]) != nvgWINDING {
		c.commandX = vals[len(vals)-2]
		c.commandY = vals[len(vals)-1]
	}

	i := 0
	for i < len(vals) {
		switch nvgCommands(vals[i]) {
		case nvgMOVETO:
			vals[i+1], vals[i+2] = xForm.TransformPoint(vals[i+1], vals[i+2])
			i += 3
		case nvgLINETO:
			vals[i+1], vals[i+2] = xForm.TransformPoint(vals[i+1], vals[i+2])
			i += 3
		case nvgBEZIERTO:
			vals[i+1], vals[i+2] = xForm.TransformPoint(vals[i+1], vals[i+2])
			vals[i+3], vals[i+4] = xForm.TransformPoint(vals[i+3], vals[i+4])
			vals[i+5], vals[i+6] = xForm.TransformPoint(vals[i+5], vals[i+6])
			i += 7
		case nvgCLOSE:
			i++
		case nvgWINDING:
			i += 2
		default:
			i++
		}
	}
	c.commands = append(c.commands, vals...)
}

func (c *Context) flattenPaths() {
	cache := &c.cache
	if len(cache.paths) > 0 {
		return
	}
	// Flatten
	i := 0
	for i < len(c.commands) {
		switch nvgCommands(c.commands[i]) {
		case nvgMOVETO:
			cache.addPath()
			cache.addPoint(c.commands[i+1], c.commands[i+2], nvgPtCORNER, c.distTol)
			i += 3
		case nvgLINETO:
			cache.addPoint(c.commands[i+1], c.commands[i+2], nvgPtCORNER, c.distTol)
			i += 3
		case nvgBEZIERTO:
			last := cache.lastPoint()
			if last != nil {
				cache.tesselateBezier(
					last.x, last.y,
					c.commands[i+1], c.commands[i+2],
					c.commands[i+3], c.commands[i+4],
					c.commands[i+5], c.commands[i+6], 0, nvgPtCORNER, c.tessTol, c.distTol)
			}
			i += 7
		case nvgCLOSE:
			cache.closePath()
			i++
		case nvgWINDING:
			cache.pathWinding(Winding(c.commands[i+1]))
			i += 2
		default:
			i++
		}
	}

	cache.bounds = [4]float32{1e6, 1e6, -1e6, -1e6}

	// Calculate the direction and length of line segments.
	for j := 0; j < len(cache.paths); j++ {
		path := &cache.paths[j]
		points := cache.points[path.first:]
		p0 := &points[path.count-1]
		p1Index := 0
		p1 := &points[p1Index]
		if ptEquals(p0.x, p0.y, p1.x, p1.y, c.distTol) && path.count > 2 {
			path.count--
			p0 = &points[path.count-1]
			path.closed = true
		}

		// Enforce winding.
		if path.count > 2 {
			area := polyArea(points, path.count)
			if path.winding == Solid && area < 0.0 {
				polyReverse(points, path.count)
			} else if path.winding == Hole && area > 0.0 {
				polyReverse(points, path.count)
			}
		}
		for i := 0; i < path.count; i++ {
			// Calculate segment direction and length
			p0.len, p0.dx, p0.dy = normalize(p1.x-p0.x, p1.y-p0.y)
			// Update bounds
			cache.bounds = [4]float32{
				minF(cache.bounds[0], p0.x),
				minF(cache.bounds[1], p0.y),
				maxF(cache.bounds[2], p0.x),
				maxF(cache.bounds[3], p0.y),
			}
			// Advance
			p1Index++
			p0 = p1
			if len(points) != p1Index {
				p1 = &points[p1Index]
			}
		}
	}
}

func (c *Context) flushTextTexture() {
	dirty := c.fs.ValidateTexture()
	if dirty != nil {
		fontImage := c.fontImages[c.fontImageIdx]
		// Update texture
		if fontImage != 0 {
			data, _, _ := c.fs.GetTextureData()
			x := dirty[0]
			y := dirty[1]
			w := dirty[2] - x
			h := dirty[3] - y
			c.gl.renderUpdateTexture(fontImage, x, y, w, h, data)
		}
	}
}

func (c *Context) allocTextAtlas() bool {
	c.flushTextTexture()
	if c.fontImageIdx >= nvgMaxFontImages-1 {
		return false
	}
	var iw, ih int
	// if next fontImage already have a texture
	if c.fontImages[c.fontImageIdx+1] != 0 {
		iw, ih, _ = c.ImageSize(c.fontImages[c.fontImageIdx+1])
	} else { // calculate the new font image size and create it.
		iw, ih, _ = c.ImageSize(c.fontImages[c.fontImageIdx])
		if iw > ih {
			ih *= 2
		} else {
			iw *= 2
		}
		if iw > nvgMaxFontImageSize || ih > nvgMaxFontImageSize {
			iw = nvgMaxFontImageSize
			ih = nvgMaxFontImageSize
		}
		c.fontImages[c.fontImageIdx+1] = c.gl.renderCreateTexture(nvgTextureALPHA, iw, ih, nil)
	}
	c.fontImageIdx++
	c.fs.ResetAtlas(iw, ih)
	return true
}

func (c *Context) renderText(vertexes []nvgVertex) {
	state := c.getState()
	paint := state.fill

	// Render triangles
	paint.image = c.fontImages[c.fontImageIdx]

	c.gl.renderTriangleStrip(&paint, &state.scissor, vertexes)

	c.drawCallCount++
	c.textTriCount += len(vertexes) / 3
}
