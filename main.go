// Copyright 2014 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Renders a textured spinning cube using GLFW 3 and OpenGL 4.1 core forward-compatible profile.
package main // import "github.com/go-gl/example/gl41core-cube"

import (
	"fmt"
	_ "image/png"
	"log"
	"runtime"
	"strings"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"unsafe"
)

const windowWidth = 800
const windowHeight = 600

func init() {
	// GLFW event handling must run on the main OS thread
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

	// Initialize Glow
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

	// define ssbo for triangles
	var ssbo uint32
	gl.GenBuffers(1, &ssbo)
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, ssbo)
	gl.BufferData(gl.SHADER_STORAGE_BUFFER, len(triVerts)*4, gl.Ptr(triVerts), gl.STATIC_COPY)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 3, ssbo)
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, 0) // unbind

	// define texture to draw framebuffer onto
	var tex uint32
	gl.GenTextures(1, &tex);
	gl.BindTexture(gl.TEXTURE_2D, tex);
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA32F, windowWidth, windowHeight, 0, gl.RGBA, gl.FLOAT, nil);
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST);
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST);
	gl.GenerateMipmap(gl.TEXTURE_2D);

	// define vao, vbo for full screen quad to render 
	// texture onto
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 2*4, gl.Ptr([]float32{
		-1.0, -1.0, 0.0,
		-1.0, 1.0, 0.0,
		1.0, -1.0, 0.0,
		1.0, 1.0, 0.0,
	}), gl.STATIC_DRAW)

	/* Specify that our coordinate data is going into attribute index 1, and contains two floats per vertex */
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 0, gl.Ptr(nil));
	/* Enable attribute index 1 as being used */
	gl.EnableVertexAttribArray(1);

	// Configure the compute shader
	computeShaderProgram, err := newComputeShaderProgram(computeShader)
	if err != nil {
		panic(err)
	}

	// Configure the vao quad shader
	vaoProgram, err := newVaoProgram(vertexShader, fragmentShader)
	if err != nil {
		panic(err)
	}

	// pipe triangles from vao to uniform named "vert" in compute shader
	vertAttrib := uint32(gl.GetAttribLocation(computeShaderProgram, gl.Str("vert\x00")))
	gl.VertexAttribPointer(vertAttrib, 3, gl.FLOAT, false, 3*4, gl.Ptr(&triVerts[0]))
	gl.EnableVertexAttribArray(vertAttrib)

	// pipe texcoord to uniform named "texcoord" in vertex shader
	texcoordAttrib := uint32(gl.GetAttribLocation(vaoProgram, gl.Str("texcoord\x00")))
	gl.VertexAttribPointer(texcoordAttrib, 2, gl.FLOAT, false, 2*4, gl.Ptr(&texCoords[0]))
	gl.EnableVertexAttribArray(texcoordAttrib)

	//texAttrib := uint32(gl.GetAttribLocation(computeShaderProgram, gl.Str("tex\x00")))

	//ourtexAttrib := uint32(gl.GetAttribLocation(vaoProgram, gl.Str("ourTexture\x00")))
	//gl.VertexAttribPointer(ourtexAttrib, 4, gl.FLOAT, false, 4*4, gl.Ptr(texAttrib))
	//gl.EnableVertexAttribArray(ourtexAttrib)

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)

	previousTime := glfw.GetTime()

	for !window.ShouldClose() {
		// dispatch shader
		gl.BindImageTexture(0, tex, 0, false, 0, gl.READ_WRITE, gl.RGBA32F);
		gl.UseProgram(computeShaderProgram)
		gl.Uniform1i(gl.GetUniformLocation(computeShaderProgram, gl.Str("tex\x00")), 0);

		gl.DispatchCompute(windowWidth, windowHeight, 1)

		// make sure writing to image has finished before read
		gl.MemoryBarrier(gl.SHADER_IMAGE_ACCESS_BARRIER_BIT)

		gl.UseProgram(vaoProgram)
		
		// drawing pass
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
			
		// Update
		time := glfw.GetTime()
		elapsed := time - previousTime
		previousTime = time
		fmt.Println(1.0/elapsed, "fps")

		// Maintenance
		window.SwapBuffers()
		glfw.PollEvents()
	}
}

func newComputeShaderProgram(computeShaderSource string) (uint32, error) {
	computeShader, err := compileShader(computeShaderSource, gl.COMPUTE_SHADER)
	if err != nil {
		return 0, err
	}

	program := gl.CreateProgram()

	gl.AttachShader(program, computeShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	gl.DeleteShader(computeShader)

	return program, nil
}

func newVaoProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}

	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	program := gl.CreateProgram()

	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)
	
	return program, nil
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

var vertexShader = `
#version 450 core
in vec3 pos;
out vec4 col;
out vec2 texcoord;
uniform sampler2D ourTexture;
void main() {
	gl_Position = vec4(pos, 1.0);
	col = vec4(pos, 1.0);
}
` + "\x00"

var fragmentShader = `
#version 450 core
in vec4 col;
in vec2 texcoord;
out vec4 finalCol;
uniform sampler2D ourTexture;
void main() {
	finalCol = texture(ourTexture, texcoord);
}
` + "\x00"

var computeShader = `
#version 450 core
layout(std430, binding = 3) buffer triVerts
{
	vec3 vertex[];
};
layout(local_size_x = 1, local_size_y = 1) in;
layout (rgba32f)  uniform image2D tex;


void main() {
	ivec2 gid = ivec2(gl_GlobalInvocationID.xy);
	imageStore(tex, gid, vec4(1.0, 1.0, 1.0, 1.0));
}
` + "\x00"

var triVerts = []float32 {
	-0.5, -0.5, 0.0,
	 0.0,  0.5, 0.0,
	 0.5, -0.5, 0.0,
	-1.0, -1.0, 0.0,
	-0.7, -0.7, 0.0,
	-0.7,  1.0, 0.0,
}

var texCoords = []float32 {
	1.0, 1.0,
	 -1.0, 1.0,
	 1.0,  -1.0,
	-1.0, -1.0,
}