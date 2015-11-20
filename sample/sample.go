// +build !js

package main

import (
	"github.com/goxjs/gl"
	"github.com/goxjs/glfw"
	"github.com/shibukawa/nanovgo"
	"github.com/shibukawa/nanovgo/perfgraph"
	"log"
	//"time"
)

var blowup bool
var screenshot bool
var premult bool

func key(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if key == glfw.KeyEscape && action == glfw.Press {
		w.SetShouldClose(true)
	} else if key == glfw.KeySpace && action == glfw.Press {
		blowup = !blowup
	} else if key == glfw.KeyS && action == glfw.Press {
		screenshot = true
	} else if key == glfw.KeyP && action == glfw.Press {
		premult = !premult
	}
}

func renderDemo(ctx *nanovgo.Context, mx, my, width, height, t float32, data *DemoData) {
	drawEyes(ctx, width-250, 50, 150, 100, mx, my, t)
	drawLines(ctx, 120, height-50, 600, 50, t)
	drawWidths(ctx, 10, 50, 30)
	drawCaps(ctx, 10, 300, 30)
}

func main() {
	err := glfw.Init(gl.ContextWatcher)
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	// demo MSAA
	glfw.WindowHint(glfw.Samples, 4)

	window, err := glfw.CreateWindow(1000, 600, "NanoVGo", nil, nil)
	if err != nil {
		panic(err)
	}
	window.SetKeyCallback(key)
	window.MakeContextCurrent()

	ctx, err := nanovgo.NewContext( /*nanovgo.ANTIALIAS | */ nanovgo.STENCIL_STROKES | nanovgo.DEBUG)
	defer ctx.Delete()

	if err != nil {
		panic(err)
	}

	demoData := &DemoData{}
	demoData.loadData(ctx)

	glfw.SwapInterval(0)

	fps := perfgraph.NewPerfGraph(perfgraph.RENDER_FPS, "Frame Time", "sans")

	for !window.ShouldClose() {
		t, dt := fps.UpdateGraph()

		//time.Sleep(time.Second*time.Duration(0.016666 - dt))
		log.Println(t, 1.0/dt)

		fbWidth, fbHeight := window.GetFramebufferSize()
		winWidth, winHeight := window.GetSize()
		mx, my := window.GetCursorPos()

		pixelRatio := float32(fbWidth) / float32(winWidth)
		gl.Viewport(0, 0, fbWidth, fbHeight)
		if premult {
			gl.ClearColor(0, 0, 0, 0)
		} else {
			gl.ClearColor(0.3, 0.3, 0.32, 1.0)
		}
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		gl.Enable(gl.CULL_FACE)
		gl.Disable(gl.DEPTH_TEST)

		ctx.BeginFrame(winWidth, winHeight, pixelRatio)

		renderDemo(ctx, float32(mx), float32(my), float32(winWidth), float32(winHeight), t, demoData)
		fps.RenderGraph(ctx, 5, 5)

		ctx.EndFrame()

		gl.Enable(gl.DEPTH_TEST)
		window.SwapBuffers()
		glfw.PollEvents()
	}

	demoData.freeData(ctx)
}