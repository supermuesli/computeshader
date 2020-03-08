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
	
	// image plane dimensions
	uniform float width = 800.0;
	uniform float height = 600.0;

	uniform int samples = 1;

	// texture to write to
	layout(binding = 6, rgba32f) uniform image2D img_output;
	
	struct Triangle 
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
		Triangle triangles[];
	};

	// camera 
	uniform vec3 cam_origin_uniform = vec3(0, 300, 950);
	uniform vec2 cursor_pos;
	
	// minimum "distance" to prevent self-intersection
	const float EPSILON = 0.000001;


	const float one_unit = 1;

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

	vec3 rotate_rand(vec3 v, float angle_1, float angle_2, float angle_3) {
		return rotate(rotate(rotate(v, vec3(1,0,0), angle_1), vec3(0,1,0), angle_2), vec3(0,0,1), angle_3);
	}

	highp float rand(vec2 co)
	{
		highp float a = 12.9898;
		highp float b = 78.233;
		highp float c = 43758.5453;
		highp float dt= dot(co.xy ,vec2(a,b));
		highp float sn= mod(dt,3.14);
		return 2*fract(sin(sn) * c)-1;
	}

	vec2 csh(float u, float v) {
		float m = 1;
		float theta = acos(pow(1-u, 1/(1+m)));
		float phi = 2 * 3.1415926535897932 * v;

		return vec2(sin(theta) * cos(phi), sin(theta) * sin(phi));
	}

	vec3 trace(vec3 ray_origin, vec3 ray_dir, int hops) {
		vec3 col = vec3(1);
		vec3 inten = vec3(0);
		bool left_the_scene = true;
		for (int hop = 0; hop < hops; ++hop) {
			float min_d = 999999.0;
			float d = 999999.0;
			int closest_tri;
			vec3 normal;
			for (int i = 0; i < triangles.length(); i++) {
				vec3 v0 = (height/2)*one_unit*triangles[i].a;
				vec3 v1 = (height/2)*one_unit*triangles[i].b;
				vec3 v2 = (height/2)*one_unit*triangles[i].c;
				if (intersects(ray_origin, ray_dir, v0, v1, v2, d)) {
					left_the_scene = false;
					if (d < min_d) {
						min_d = d;
						normal = normalize(cross(v1 - v0, v2 - v0));

						// normal buffer
						//col = vec3(normal + vec3(1))/2;

						closest_tri = i;
					}
				}
			}

			if (left_the_scene) {
				break;
			}

			inten = inten + triangles[closest_tri].intensity*abs(dot(ray_dir, normal));
			col = col * triangles[closest_tri].color;
			ray_origin = ray_origin + min_d*ray_dir;
			float rand_1 = rand(ray_dir.xy*samples*2.23234899874);
			float rand_2 = rand(ray_dir.xz*samples*rand_1);
			float rand_3 = rand(ray_dir.yz*samples*rand_2);
			ray_dir = normalize(rotate_rand(normal, rand_1*3.1415/2, rand_2*3.1415/2, rand_3*3.1415/2));
			
			// account for self interesction
			ray_origin = ray_origin + ray_dir*0.001;
		}

		return col * inten *10;
	}

	void main() {
		// get index in global work group i.e x,y position
		ivec2 pixel_coord = ivec2(gl_GlobalInvocationID.xy);

		// rotate camera based on cursor position
		vec3 cam_origin = cam_origin_uniform;

		vec3 ray_dest = vec3(cam_origin.x - width/2 + pixel_coord.x, cam_origin.y - height/2 + pixel_coord.y, cam_origin.z - height);
		vec3 ray_dir = normalize(ray_dest - cam_origin);
		ray_dir = rotate(rotate(ray_dir, vec3(1,0,0), (2*cursor_pos.y/height) - 1), vec3(0,1,0), (2*cursor_pos.x/width) - 1);

		// send camera ray
		vec3 pixel = trace(cam_origin, ray_dir, 3) + imageLoad(img_output, pixel_coord).xyz * (samples-1); 
		
		// gamma correction
		//pixel = vec3(pow(pixel.x, 1.22), pow(pixel.y, 1.22), pow(pixel.z, 1.22));

		// output to a specific pixel in the texture
		imageStore(img_output, pixel_coord, vec4(pixel/samples, 1.0));
	}
	` + "\x00"
)