package main

import (
	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/supermuesli/computeshader/pkg/shaders"
	"github.com/supermuesli/computeshader/internal/shaderutils"
	"fmt"
	_ "image/png"
	"log"
	"runtime"
	"unsafe"
)

const (
	windowWidth = 800
	windowHeight = 600
)

func init() {
	// glfw event handling must run on the main OS thread
	runtime.LockOSThread()
}

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 5)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	window, err := glfw.CreateWindow(windowWidth, windowHeight, "compute shady boi", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()
	glfw.SwapInterval(0)

	// init glow
	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)
	gl.Enable(gl.DEBUG_OUTPUT)
	gl.DebugMessageCallback(
		func (
			source uint32,
			gltype uint32,
			id uint32,
			severity uint32,
			length int32,
			message string,
			userParam unsafe.Pointer,
		){
			fmt.Println(source, gltype, id, severity, length, message, userParam)
		}, nil,
	)

	// print max workgroup count/size/invocations
	fmt.Println("***-----------------------------------------------------------------------------***")
	var workGroupCount [3]int32
	gl.GetIntegeri_v(gl.MAX_COMPUTE_WORK_GROUP_COUNT, 0, &workGroupCount[0]);
	gl.GetIntegeri_v(gl.MAX_COMPUTE_WORK_GROUP_COUNT, 1, &workGroupCount[1]);
	gl.GetIntegeri_v(gl.MAX_COMPUTE_WORK_GROUP_COUNT, 2, &workGroupCount[2]);
	fmt.Printf("max global (total) work group size x:%i y:%i z:%i\n", workGroupCount[0], workGroupCount[1], workGroupCount[2])

	var workGroupSize [3]int32
	gl.GetIntegeri_v(gl.MAX_COMPUTE_WORK_GROUP_SIZE, 0, &workGroupSize[0]);
	gl.GetIntegeri_v(gl.MAX_COMPUTE_WORK_GROUP_SIZE, 1, &workGroupSize[1]);
	gl.GetIntegeri_v(gl.MAX_COMPUTE_WORK_GROUP_SIZE, 2, &workGroupSize[2]);
	fmt.Printf("max global (in one shader) work group sizes x:%i y:%i z:%i\n", workGroupSize[0], workGroupSize[1], workGroupSize[2])

	var workGroupInv int32
	gl.GetIntegerv(gl.MAX_COMPUTE_WORK_GROUP_INVOCATIONS, &workGroupInv);
	fmt.Printf("max local work group invocations %i\n", workGroupInv);
	fmt.Println("***-----------------------------------------------------------------------------***")

	// configure compute shader
	computeShader, err := shaderutils.NewComputeShader(shaders.ComputeSrc)
	if err != nil {
		panic(err)
	}

	// configure fullscreen quad shader
	quadShader, err := shaderutils.NewQuadShader(shaders.VertexSrc, shaders.FragmentSrc)
	if err != nil {
		panic(err)
	}

	// define quad texture to draw framebuffer onto
	var quadTexture uint32
	gl.GenTextures(1, &quadTexture)
	gl.BindTexture(gl.TEXTURE_2D, quadTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA32F, windowWidth, windowHeight, 0, gl.RGBA, gl.FLOAT, nil)
	
	// define quad vao
	var quadVao uint32
	gl.GenVertexArrays(1, &quadVao)
	gl.BindVertexArray(quadVao)

	// define quad vbo
	var quadVbo uint32
	gl.GenBuffers(1, &quadVbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, quadVbo)
	quadVertices := [8]int8{-1, -1, -1, 1, 1, -1, 1, 1}
	gl.BufferData(gl.ARRAY_BUFFER, 8, unsafe.Pointer(&quadVertices[0]), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.BYTE, false, 0, nil)

	// define 3d model vertices
	var model uint32
	gl.GenBuffers(1, &model)
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, model)
	modelVertices := []float32{130, 130, 200, 130, 30, 200, 30, 30, 200}
	gl.BufferData(gl.SHADER_STORAGE_BUFFER, 9, unsafe.Pointer(&modelVertices[0]), gl.STATIC_DRAW)
	// bound to binding point 3
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 3, model)

	// color (black) that gl.Clear() is going to use
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)

	previousTime := glfw.GetTime()

	for !window.ShouldClose() {
		// dispatch shader
		gl.UseProgram(computeShader)

		// https://stackoverflow.com/questions/37136813/what-is-the-difference-between-glbindimagetexture-and-glbindtexture
		// binds a single level of a texture to an image unit for the purpose of reading and writing it from shaders. 
		gl.BindImageTexture(11, quadTexture, 0, false, 0, gl.READ_ONLY, gl.RGBA32F)

		gl.DispatchCompute(windowWidth, windowHeight, 1)

		// make sure writing to image has finished before read
		gl.MemoryBarrier(gl.SHADER_IMAGE_ACCESS_BARRIER_BIT)

		gl.Clear(gl.COLOR_BUFFER_BIT)
		
		// render compute shader output (texture) onto screen quad
		gl.UseProgram(quadShader)
		
		// https://community.khronos.org/t/when-to-use-glactivetexture/64913/2
		gl.ActiveTexture(gl.TEXTURE12)
		
		// calling glBindTexture binds the texture name
		// to the target. When a texture is bound to a target, the previous binding for that target is automatically broken.
		gl.BindTexture(gl.TEXTURE_2D, quadTexture)
		
		gl.BindVertexArray(quadVao)
		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

		time := glfw.GetTime()
		elapsed := time - previousTime
		fmt.Println(int(1.0/elapsed), "FPS")
		previousTime = time

		// poll keyboard/mouse events
		glfw.PollEvents()
		
		window.SwapBuffers()
	}
}