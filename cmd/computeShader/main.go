package main

import (
	"fmt"
	_ "image/png"
	"log"
	"runtime"
	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"unsafe"
	"github.com/supermuesli/computeshader/pkg/shaders"
	"github.com/supermuesli/computeshader/internal/shaderutils"
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
	var work_grp_cnt [3]int32
	gl.GetIntegeri_v(gl.MAX_COMPUTE_WORK_GROUP_COUNT, 0, &work_grp_cnt[0]);
	gl.GetIntegeri_v(gl.MAX_COMPUTE_WORK_GROUP_COUNT, 1, &work_grp_cnt[1]);
	gl.GetIntegeri_v(gl.MAX_COMPUTE_WORK_GROUP_COUNT, 2, &work_grp_cnt[2]);
	fmt.Printf("max global (total) work group size x:%i y:%i z:%i\n", work_grp_cnt[0], work_grp_cnt[1], work_grp_cnt[2])

	var work_grp_size [3]int32
	gl.GetIntegeri_v(gl.MAX_COMPUTE_WORK_GROUP_SIZE, 0, &work_grp_size[0]);
	gl.GetIntegeri_v(gl.MAX_COMPUTE_WORK_GROUP_SIZE, 1, &work_grp_size[1]);
	gl.GetIntegeri_v(gl.MAX_COMPUTE_WORK_GROUP_SIZE, 2, &work_grp_size[2]);
	fmt.Printf("max global (in one shader) work group sizes x:%i y:%i z:%i\n", work_grp_size[0], work_grp_size[1], work_grp_size[2])

	var work_grp_inv int32
	gl.GetIntegerv(gl.MAX_COMPUTE_WORK_GROUP_INVOCATIONS, &work_grp_inv);
	fmt.Printf("max local work group invocations %i\n", work_grp_inv);
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

	// define texture to draw framebuffer onto
	var texOutput uint32
	gl.GenTextures(1, &texOutput)
	gl.BindTexture(gl.TEXTURE_2D, texOutput)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA32F, windowWidth, windowHeight, 0, gl.RGBA, gl.FLOAT, nil)
	gl.BindImageTexture(0, texOutput, 0, false, 0, gl.WRITE_ONLY, gl.RGBA32F)

	// define quad vao
	var quadVao uint32
	gl.GenVertexArrays(1, &quadVao)
	gl.BindVertexArray(quadVao)

	// define quad vbo
	var quadVbo uint32
	gl.GenBuffers(1, &quadVbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, quadVbo)
	buf := [12]int8{-1, -1, 1, -1, 1, 1, 1, 1, -1, 1, -1, -1}
	gl.BufferData(gl.ARRAY_BUFFER, 12, unsafe.Pointer(&buf[0]), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.BYTE, false, 0, nil)

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)

	previousTime := glfw.GetTime()

	for !window.ShouldClose() {
		// dispatch shader
		gl.UseProgram(computeShader)

		// binds a single level of a texture to an image unit for the purpose of reading and writing it from shaders. 
		// unit specifies the zero-based index of the image unit to which to bind the texture level. texture specifies 
		// the name of an existing texture object to bind to the image unit. If texture is zero, then any existing 
		// binding to the image unit is broken. level specifies the level of the texture to bind to the image unit.
		gl.BindImageTexture(0, texOutput, 0, false, 0, gl.WRITE_ONLY, gl.RGBA32F)
		
		gl.DispatchCompute(windowWidth, windowHeight, 1)

		// make sure writing to image has finished before read
		gl.MemoryBarrier(gl.SHADER_IMAGE_ACCESS_BARRIER_BIT)

		// render to screen
		gl.Clear(gl.COLOR_BUFFER_BIT)
		gl.UseProgram(quadShader)
		gl.BindVertexArray(quadVao)
		
		// lets you create or use a named texture. Calling glBindTexture with target set to GL_TEXTURE_1D, 
		// GL_TEXTURE_2D, GL_TEXTURE_3D, GL_TEXTURE_1D_ARRAY, GL_TEXTURE_2D_ARRAY, GL_TEXTURE_RECTANGLE, 
		// GL_TEXTURE_CUBE_MAP, GL_TEXTURE_CUBE_MAP_ARRAY, GL_TEXTURE_BUFFER, GL_TEXTURE_2D_MULTISAMPLE or 
		// GL_TEXTURE_2D_MULTISAMPLE_ARRAY and texture set to the name of the new texture binds the texture 
		// name to the target. When a texture is bound to a target, the previous binding for that target is
		// automatically broken.
		gl.BindTexture(gl.TEXTURE_2D, texOutput)
		
		gl.DrawArrays(gl.TRIANGLES, 0, 6)

		time := glfw.GetTime()
		elapsed := time - previousTime
		//fmt.Println(int(1.0/elapsed), "FPS")
		_ = elapsed
		previousTime = time

		// poll keyboard/mouse events
		glfw.PollEvents()
		
		window.SwapBuffers()
	}
}