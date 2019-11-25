package shaders

const (
	VertexSrc = `
	#version 450 core
	in vec2 pos;
	out vec2 coord;
	void main() {
		gl_Position = vec4(pos, 0.0, 1.0);
		coord = 0.5 * pos + vec2(0.5, 0.5);
	}
	` + "\x00"

	FragmentSrc = `
	#version 450 core
	in vec2 coord;
	out vec4 finalCol;
	layout(binding = 1) uniform sampler2D img_output;
	void main() {
		finalCol = texture(img_output, coord);
	}
	` + "\x00"

	ComputeSrc = `
	#version 450 core
	layout(local_size_x = 1, local_size_y = 1) in;
	layout(rgba32f, binding = 1) uniform image2D img_output;

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
)