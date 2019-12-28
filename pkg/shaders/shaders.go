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
	layout(binding = 6, rgba32f) uniform image2D img_output;

	// triangles to render
	layout(std430, binding = 3) buffer model
	{
		float vertex_comp[];
	};

	// minimum "distance" to prevent self-intersection
	const float EPSILON = 0.0000001;

	// möller trombore triangle intersection
	bool intersects(vec3 ray_origin, vec3 ray_dir, vec3 p0, vec3 p1, vec3 p2, out float d) {
		vec3 edge1, edge2, h, s, q;
		float a, f, u, v;
		edge1 = p1 - p0;
		edge2 = p2 - p0;
		h = cross(ray_dir, edge2);
		a = dot(edge1, h);
		if (a > -EPSILON && a < EPSILON)
			// This ray is parallel to this triangle.
			return false; 
		f = 1.0/a;
		s = ray_origin - p0;
		u = f * dot(s, h);
		if (u < 0.0 || u > 1.0)
			return false;
		q = cross(s, edge1);
		v = f * dot(ray_dir, q);
		if (v < 0.0 || u + v > 1.0)
			return false;
		// At this stage we can compute d to find out where the intersection point is on the line.
		d = f * dot(edge2, q);
		if (d > EPSILON && d < 1/EPSILON) {
			// ray intersection
			return true;
		}
		else {
			// This means that there is a line intersection but not a ray intersection.
			return false;
		}
	}

	void main() {
		// get index in global work group i.e x,y position
		ivec2 pixel_coord = ivec2(gl_GlobalInvocationID.xy);
		
		const float width = 800.0;
		const float height = 600.0;
		const float one_unit = 1.0;

		// TODO dont hardcode camera
		vec3 cam_origin = vec3(0.5/width, 0.5/height, -one_unit);

		vec3 ray_dest = vec3(vec2(1/width, 1/height)*pixel_coord.xy - cam_origin.xy, cam_origin.z + one_unit);
		vec3 ray_dir = normalize(ray_dest - cam_origin);

		// final pixel color
		vec3 pixel = vec3(0.0);
		float min_d = 999999.0;
		float d = 999999.0;

		// send camera ray
		for(int i = 0; i < vertex_comp.length(); i = i+9) {
			// 3 vertex components -> 1 vertex
			// 3 vertices		   -> 1 triangle
			// 9 vertex components -> 1 triangle
			vec3 v0 = vec3(vertex_comp[i], vertex_comp[i+1], vertex_comp[i+2]);
			vec3 v1 = vec3(vertex_comp[i+3], vertex_comp[i+4], vertex_comp[i+5]);
			vec3 v2 = vec3(vertex_comp[i+6], vertex_comp[i+7], vertex_comp[i+8]);
			if (intersects(cam_origin, ray_dir, v0, v1, v2, d)) {
				if (d < min_d) {
					min_d = d;
					// TODO replace with actual triangle color
					pixel = vec3(0.0, 1.0, 0.0);
				}
			}
		}
		
		// output to a specific pixel in the texture
		imageStore(img_output, pixel_coord, vec4(pixel, 1.0));
	}
	` + "\x00"
)