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
	const float EPSILON = 0.0001;

	// used by smartDeNoise()
	const float INV_SQRT_OF_2PI = 0.39894228040143267793994605993439;
	const float INV_PI = 0.31830988618379067153776752674503;

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
		
		return mat4(oc * axis.x * axis.x + c,		   oc * axis.x * axis.y - axis.z * s,  oc * axis.z * axis.x + axis.y * s,  0.0,
					oc * axis.x * axis.y + axis.z * s,  oc * axis.y * axis.y + c,		   oc * axis.y * axis.z - axis.x * s,  0.0,
					oc * axis.z * axis.x - axis.y * s,  oc * axis.y * axis.z + axis.x * s,  oc * axis.z * axis.z + c,		   0.0,
					0.0,								0.0,								0.0,								1.0);
	}

	vec3 rotate(vec3 v, vec3 axis, float angle) {
		mat4 m = rotationMatrix(axis, angle);
		return (m * vec4(v, 1.0)).xyz;
	}	

	vec3 rotate_rand(vec3 v, float angle_1, float angle_2, float angle_3) {
		return rotate(rotate(rotate(v, vec3(1,0,0), angle_1), vec3(0,1,0), angle_2), vec3(0,0,1), angle_3);
	}


	// A single iteration of Bob Jenkins' One-At-A-Time hashing algorithm.
	uint hash( uint x ) {
		x += ( x << 10u );
		x ^= ( x >>  6u );
		x += ( x <<  3u );
		x ^= ( x >> 11u );
		x += ( x << 15u );
		return x;
	}



	// Compound versions of the hashing algorithm I whipped together.
	uint hash( uvec2 v ) { return hash( v.x ^ hash(v.y)						 ); }
	uint hash( uvec3 v ) { return hash( v.x ^ hash(v.y) ^ hash(v.z)			 ); }
	uint hash( uvec4 v ) { return hash( v.x ^ hash(v.y) ^ hash(v.z) ^ hash(v.w) ); }



	// Construct a float with half-open range [0:1] using low 23 bits.
	// All zeroes yields 0.0, all ones yields the next smallest representable value below 1.0.
	float floatConstruct( uint m ) {
		const uint ieeeMantissa = 0x007FFFFFu; // binary32 mantissa bitmask
		const uint ieeeOne	  = 0x3F800000u; // 1.0 in IEEE binary32

		m &= ieeeMantissa;					 // Keep only mantissa bits (fractional part)
		m |= ieeeOne;						  // Add fractional part to 1.0

		float  f = uintBitsToFloat( m );	   // Range [1:2]
	return f - 1.0;						// Range [0:1]
	}



	// Pseudo-random value in half-open range [0:1].
	float rand( float x ) { return 2*floatConstruct(hash(floatBitsToUint(x)))-1; }
	float rand( vec2  v ) { return 2*floatConstruct(hash(floatBitsToUint(v)))-1; }
	float rand( vec3  v ) { return 2*floatConstruct(hash(floatBitsToUint(v)))-1; }
	float rand( vec4  v ) { return 2*floatConstruct(hash(floatBitsToUint(v)))-1; }

	vec2 csh(float u, float v) {
		float m = 1;
		float theta = acos(pow(1-u, 1/(1+m)));
		float phi = 2 * 3.1415926535897932 * v;

		return vec2(sin(theta) * cos(phi), sin(theta) * sin(phi));
	}


	// https://github.com/BrutPitt/glslSmartDeNoise
	//  smartDeNoise - parameters
	//~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	//
	//  sampler2D tex	  - sampler image / texture
	//  vec2 uv			  - actual fragment coord
	//  float sigma  >  0 - sigma Standard Deviation
	//  float kSigma >= 0 - sigma coefficient 
	//	  kSigma * sigma  -->  radius of the circular kernel
	//  float threshold   - edge sharpening threshold 

	vec4 smartDeNoise(image2D img, ivec2 uv, float sigma, float kSigma, float threshold)
	{
		float radius = round(kSigma*sigma);
		float radQ = radius * radius;
		
		float invSigmaQx2 = .5 / (sigma * sigma);	  
		float invSigmaQx2PI = INV_PI * invSigmaQx2;
		
		float invThresholdSqx2 = .5 / (threshold * threshold);
		float invThresholdSqrt2PI = INV_SQRT_OF_2PI / threshold;
		
		vec4 centrPx = imageLoad(img, uv); 
		
		float zBuff = 0.0;
		vec4 aBuff = vec4(0.0);
		vec2 size = vec2(width, height);
		
		for(float x=-radius; x <= radius; x++) {
			float pt = sqrt(radQ-x*x);  // pt = yRadius: have circular trend
			for(float y=-pt; y <= pt; y++) {
				vec2 d = vec2(x,y)/size;

				float blurFactor = exp( -dot(d , d) * invSigmaQx2 ) * invSigmaQx2;
				
				vec4 walkPx =  imageLoad(img, uv+ivec2(d.x*width, d.y*height));

				vec4 dC = walkPx-centrPx;
				float deltaFactor = exp( -dot(dC, dC) * invThresholdSqx2) * invThresholdSqrt2PI * blurFactor;
									 
				zBuff += deltaFactor;
				aBuff += deltaFactor*walkPx;
			}
		}
		return aBuff/zBuff;
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
			float rand_1 = rand(ray_dir.xy*samples);
			float rand_2 = rand(ray_dir.xz*samples);
			float rand_3 = rand(ray_dir.yz*samples);
			ray_dir = normalize(rotate_rand(normal, rand_1*3.1415/2, rand_2*3.1415/2, rand_3*3.1415/2));
			
			// account for self interesction
			ray_origin = ray_origin + ray_dir*0.001;
		}

		return col * inten * 5;
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

		// denoise
		//pixel = smartDeNoise(img_output, pixel_coord, 3.0, 7.0, 0.15).xyz;

		// and output to the specific pixel in the texture again
		//imageStore(img_output, pixel_coord, vec4(pixel, 1.0));
	}
	` + "\x00"
)