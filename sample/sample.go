package main

import (
	"runtime"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/JubbeArt/nanovgo"
)

func main() {
	runtime.LockOSThread()

	err := glfw.Init()
	check(err)
	defer glfw.Terminate()

	err = gl.Init()
	check(err)
	// demo MSAA
	glfw.WindowHint(glfw.Samples, 4)

	window, err := glfw.CreateWindow(1000, 600, "NanoVGo", nil, nil)
	check(err)
	window.MakeContextCurrent()

	ctx, err := nanovgo.NewContext(0 /*nanovgo.AntiAlias | nanovgo.StencilStrokes | nanovgo.Debug*/)
	check(err)
	defer ctx.Delete()

	glfw.SwapInterval(0)

	for !window.ShouldClose() {
		fbWidth, fbHeight := window.GetFramebufferSize()
		winWidth, winHeight := window.GetSize()

		pixelRatio := float32(fbWidth) / float32(winWidth)
		gl.Viewport(0, 0, int32(fbWidth), int32(fbHeight))
		gl.ClearColor(0, 0, 0, 0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		gl.Enable(gl.CULL_FACE)
		gl.Disable(gl.DEPTH_TEST)

		ctx.BeginFrame(winWidth, winHeight, pixelRatio)

		ctx.BeginPath()

		ctx.SetFillColor(nanovgo.RGB(255, 0, 0))
		//ctx.SetFillColor(colornames.Red)
		ctx.Rect(50, 50, 100, 100)
		ctx.Fill()

		ctx.EndFrame()

		gl.Enable(gl.DEPTH_TEST)
		window.SwapBuffers()
		glfw.PollEvents()
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}

}
