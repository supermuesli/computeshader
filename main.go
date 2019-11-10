package main

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
	computeShaderProgram, err := newComputeShaderProgram(computeShader)
	if err != nil {
		panic(err)
	}

	// configure fullscreen quad shader
	quadProgram, err := newQuadProgram(vertexShader, fragmentShader)
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
		gl.UseProgram(computeShaderProgram)
		gl.BindImageTexture(0, texOutput, 0, false, 0, gl.WRITE_ONLY, gl.RGBA32F)
		gl.DispatchCompute(windowWidth, windowHeight, 1)

		// make sure writing to image has finished before read
		gl.MemoryBarrier(gl.SHADER_IMAGE_ACCESS_BARRIER_BIT)

		// render to screen
		gl.Clear(gl.COLOR_BUFFER_BIT)
		gl.UseProgram(quadProgram)
		gl.BindVertexArray(quadVao)
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

func newQuadProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
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
	gl.BindAttribLocation(program, 0, gl.Str("pos\x00"))
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
in vec2 pos;
out vec2 coord;
void main() {
	gl_Position = vec4(pos, 0.0, 1.0);
	coord = 0.5 * pos + vec2(0.5, 0.5);
}
` + "\x00"

var fragmentShader = `
#version 450 core
in vec2 coord;
out vec4 finalCol;
uniform sampler2D img_output;
void main() {
	finalCol = texture(img_output, coord);
}
` + "\x00"

var computeShader = `
#version 450 core
layout(local_size_x = 1, local_size_y = 1) in;
layout(rgba32f, binding = 0) uniform image2D img_output;

void main() {
	// get index in global work group i.e x,y position
	ivec2 pixel_coords = ivec2(gl_GlobalInvocationID.xy);
	
	// base pixel color for image
	vec4 pixel = vec4(float(pixel_coords[0])/800, 0.0, 0.0, 1.0);

	//
	// interesting stuff happens here later
	//
	
	// output to a specific pixel in the image
	imageStore(img_output, pixel_coords, pixel);
}
` + "\x00"