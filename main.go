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

	// Configure the compute shader
	program, err := newProgram(computeShader)
	if err != nil {
		panic(err)
	}

	// define ssbo for triangles
	var ssbo uint32
	gl.GenBuffers(1, &ssbo)
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, ssbo)
	gl.BufferData(gl.SHADER_STORAGE_BUFFER, len(triVerts)*4, gl.Ptr(triVerts), gl.STATIC_COPY)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 3, ssbo);
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, 0); // unbind

	// pipe triangles from vao to uniform named "vert" in compute shader
	vertAttrib := uint32(gl.GetAttribLocation(program, gl.Str("vert\x00")))
	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointer(vertAttrib, 3, gl.FLOAT, false, 3*4, gl.PtrOffset(0))

	gl.UseProgram(program)

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)

	previousTime := glfw.GetTime()

	for !window.ShouldClose() {
		// dispatch shader
		gl.UseProgram(program)
		gl.DispatchCompute(windowWidth, windowHeight, 1)

		// make sure writing to image has finished before read
		gl.MemoryBarrier(gl.SHADER_IMAGE_ACCESS_BARRIER_BIT)

		// drawing pass
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		gl.DrawArrays(gl.TRIANGLES, 0, 3)
			
		// Update
		time := glfw.GetTime()
		elapsed := time - previousTime
		previousTime = time
		fmt.Println(1.0/elapsed, "fps")

		
		gl.DrawArrays(gl.TRIANGLES, 0, 6*2*3)

		// Maintenance
		window.SwapBuffers()
		glfw.PollEvents()
	}
}

func newProgram(computeShaderSource string) (uint32, error) {
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

var computeShader = `
#version 450 core
layout(local_size_x = 1, local_size_y = 1) in;
layout(std430, binding = 3) buffer triVerts
{
	vec3 vertex[];
};

// The camera specification
uniform vec3 eye;
uniform vec3 ray00;
uniform vec3 ray10;
uniform vec3 ray01;
uniform vec3 ray11;

struct tri {
	vec3 a;
	vec3 b;
	vec3 c;
};

ivec2 gid = ivec2(gl_GlobalInvocationID.xy);

void main() {
	
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