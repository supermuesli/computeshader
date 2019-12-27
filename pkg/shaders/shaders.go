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
	out vec4 final_col;
	layout(binding = 12) uniform sampler2D img_output;
	void main() {
		final_col = texture(img_output, coord);
	}
	` + "\x00"

	ComputeSrc = `
	#version 450 core
	layout(local_size_x = 1, local_size_y = 1) in;
	
	// texture to write to
	layout(binding = 11, rgba32f) uniform image2D img_output;

	// triangles to render
	layout(std430, binding = 3) buffer model
	{
		float vertex_comp[];
	};

	// minimum "distance" to prevent self-intersection
	const float EPSILON = 0.0000001;

	// möller trombore triangle intersection
	bool intersects(vec3 ray_origin, vec3 ray_dir, vec3 p0, vec3 p1, vec3 p2, out float d, out vec3 tri_normal) {
		const vec3 e0 = p1 - p0;
		const vec3 e1 = p0 - p2;
		tri_normal = cross(e1, e0);

		const vec3 e2 = (1.0/dot(tri_normal, ray_dir)) * (p0 - ray_origin);
		const vec3 i  = cross(ray_dir, e2);

		d = dot(tri_normal, e2);

		if (d > EPSILON) {
			return true;
		} 
		return false;
	}

	void main() {
		// get index in global work group i.e x,y position
		ivec2 pixel_coord = ivec2(gl_GlobalInvocationID.xy);
		
		// TODO dont hardcode camera
		vec3 cam_origin = vec3(400.0, 300.0, -600.0);

		vec3 ray_dir = normalize(vec3(pixel_coord, 0.0) - cam_origin);

		// final pixel color
		vec4 pixel = vec4(0.0, 0.0, 0.0, 1.0);
		float min_d = 999999.0;
		float d;
		vec3 min_tri_normal;
		vec3 tri_normal;

		// send camera ray
		for(int i = 0; i < vertex_comp.length(); i = i+9) {
			// 3 vertex components -> 1 vertex
			// 3 vertices          -> 1 triangle
			// 9 vertex components -> 1 triangle
			if (intersects(cam_origin, ray_dir, vec3(vertex_comp[i], vertex_comp[i+1], vertex_comp[i+2]), vec3(vertex_comp[i+3], vertex_comp[i+4], vertex_comp[i+5]), vec3(vertex_comp[i+6], vertex_comp[i+7], vertex_comp[i+8]), d, tri_normal)) {
				if (d < min_d) {
					min_d = d;
					min_tri_normal = tri_normal;
					// TODO replace with actual triangle color
					pixel = vec4(normalize(vec3(d)), 1.0);
				}
			}
		}
		
		// output to a specific pixel in the texture
		imageStore(img_output, pixel_coord, pixel);
	}
	` + "\x00"
)