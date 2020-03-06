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
	layout(local_size_x = 32, local_size_y = 8) in;
	
	// texture to write to
	layout(binding = 6, rgba32f) uniform image2D img_output;

	struct triangle 
	{
		vec3 a;
		vec3 b;
		vec3 c;
		vec3 color;
		vec3 intensity;
	};

	// triangles to render
	layout(std430, binding = 3) buffer model_ssbo
	{
		triangle triangles[];
	};

	// camera 
	uniform vec3 cam_origin_uniform = vec3(0, 300, 950);
	uniform vec2 cursor_pos = vec2(800/2, 600/2);
	
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

	mat4 rotationMatrix(vec3 axis, float angle) {
		axis = normalize(axis);
		float s = sin(angle);
		float c = cos(angle);
		float oc = 1.0 - c;
	    
		return mat4(oc * axis.x * axis.x + c,           oc * axis.x * axis.y - axis.z * s,  oc * axis.z * axis.x + axis.y * s,  0.0,
	                oc * axis.x * axis.y + axis.z * s,  oc * axis.y * axis.y + c,           oc * axis.y * axis.z - axis.x * s,  0.0,
	                oc * axis.z * axis.x - axis.y * s,  oc * axis.y * axis.z + axis.x * s,  oc * axis.z * axis.z + c,           0.0,
	                0.0,                                0.0,                                0.0,                                1.0);
	}

	vec3 rotate(vec3 v, vec3 axis, float angle) {
		mat4 m = rotationMatrix(axis, angle);
		return (m * vec4(v, 1.0)).xyz;
	}	

	void main() {
		// get index in global work group i.e x,y position
		ivec2 pixel_coord = ivec2(gl_GlobalInvocationID.xy);
		
		// image plane dimensions
		const float width = 800.0;
		const float height = 600.0;

		const float one_unit = 1;

		// rotate camera based on cursor position
		vec3 cam_origin = cam_origin_uniform;

		vec3 ray_dest = vec3(cam_origin.x - width/2 + pixel_coord.x, cam_origin.y - height/2 + pixel_coord.y, cam_origin.z - height);
		vec3 ray_dir = normalize(ray_dest - cam_origin);
		ray_dir = rotate(rotate(ray_dir, vec3(1,0,0), (2*cursor_pos.y/height) - 1), vec3(0,1,0), (2*cursor_pos.x/width) - 1);
		
		// (cam_origin+length(cam_origin_uniform)*vec3(2*cursor_pos.x/(1+(cursor_pos.x*cursor_pos.x)+(cursor_pos.y*cursor_pos.y)), 
		//                     2*cursor_pos.y/(1+(cursor_pos.x*cursor_pos.x)+(cursor_pos.y*cursor_pos.y)),
		//                     (-1+(cursor_pos.x*cursor_pos.x)+(cursor_pos.y*cursor_pos.y))/(1+(cursor_pos.x*cursor_pos.x)+(cursor_pos.y*cursor_pos.y))))

		// final pixel color
		vec3 pixel = vec3(0.0);
		float min_d = 999999.0;
		float d = 999999.0;

		// send camera ray
		for(int i = 0; i < triangles.length(); i++) {
			// 3 vertex components -> 1 vertex
			// 3 vertices		   -> 1 triangle
			// 9 vertex components -> 1 triangle
			vec3 v0 = (height/2)*one_unit*triangles[i].a;
			vec3 v1 = (height/2)*one_unit*triangles[i].b;
			vec3 v2 = (height/2)*one_unit*triangles[i].c;
			if (intersects(cam_origin, ray_dir, v0, v1, v2, d)) {
				if (d < min_d) {
					min_d = d;
					// TODO replace with actual triangle color
					vec3 u = v1 - v0;
					vec3 v = v2 - v0;
					vec3 normal = normalize(cross(u, v));

					// normal buffer
					pixel = vec3(normal + vec3(1))/2;

					// lambert shading
					//pixel = abs(vec3(triangles[i].color*dot(ray_dir, normal)));
				}
			}
		}
		
		// output to a specific pixel in the texture
		imageStore(img_output, pixel_coord, vec4(pixel, 1.0));
	}
	` + "\x00"
)