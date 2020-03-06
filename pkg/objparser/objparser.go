package objparser

import (
	"bufio"
	"os"
	"log"
	"strconv"
	s "strings"
	"fmt"
)

type Material struct {
	Name      string
	Color     [3]float32 
	Intensity [3]float32
}

type Triangle struct {
	A         [3]float32
	B         [3]float32
	C         [3]float32
	Color     [3]float32
	Intensity [3]float32
}

func GetTriangles(path string) []Triangle {
	vertices   := []float32{}
	parsedMtls := []Material{}
	triangles  := []Triangle{}

	curColor     := [3]float32{0, 0, 0}
	curIntensity := [3]float32{0, 0, 0}

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		cur := scanner.Text()
		//fmt.Println("cur at start", cur)
		if len(cur) > 1 {
			// delete all tabs
			cur = s.ReplaceAll(cur, "\t", " ")
			//fmt.Println("cut after tabs", cur)
			// delete all double spaces
			for {
				cur = s.ReplaceAll(cur, "  ", " ")
				if !s.Contains(cur, "  ") {
					break
				}
			}
			//fmt.Println("cur after double spaces", cur)
			// delete all prefix spaces
			for {
				if cur == " " {
					break
				}
				if string(cur[0]) == " " {
					cur = cur[1:]
				} else {
					break
				}
			}
			//fmt.Println("cur after prefix spaces", cur)
			if len(cur) > 5  {
				if cur[:6] == "mtllib" {
					fmt.Println("parsing", cwd + "/" + s.Split(cur, " ")[1])
					parsedMtls = append(parsedMtls, parseMtl(cwd + "/" + s.Split(cur, " ")[1])...)

				} else if cur[:6] == "usemtl" {
					materialName := s.Split(cur, " ")[1]
					for _, pm := range parsedMtls {
						if pm.Name == materialName {
							curColor = pm.Color
							curIntensity = pm.Intensity
							break
						}
					}

				}
			} 
			if string(cur[0]) == "v" && string(cur[1]) == " " {
				curVertex := s.Split(cur, " ")
				curVertex = curVertex[1:]
				//fmt.Println("cur at v", cur)
				v0, err := strconv.ParseFloat(curVertex[0], 32)
				if err != nil {
					log.Fatal(err)
				}
				v1, err := strconv.ParseFloat(curVertex[1], 32)
				if err != nil {
					log.Fatal(err)
				}
				v2, err := strconv.ParseFloat(curVertex[2], 32)
				if err != nil {
					log.Fatal(err)
				}
				vertices = append(vertices, []float32{float32(v0), float32(v1), float32(v2)}...)
			} else if string(cur[0]) == "f" {
				curFace := s.Split(cur, " ")[1:]
				i0, err := strconv.Atoi(s.Split(curFace[0], "/")[0])
				if err != nil {
					log.Fatal(err)
				}
				i1, err := strconv.Atoi(s.Split(curFace[1], "/")[0])
				if err != nil {
					log.Fatal(err)
				}
				i2, err := strconv.Atoi(s.Split(curFace[2], "/")[0])
				if err != nil {
					log.Fatal(err)
				}
				i3 := 99999999
				if len(curFace) == 4 {
					i3, err = strconv.Atoi(s.Split(curFace[3], "/")[0])
					if err != nil {
						log.Fatal(err)
					}
				}

				if i0 < 1 {
					i0 = len(vertices) + 3*i0
				}
				if i1 < 1 {
					i1 = len(vertices) + 3*i1
				}
				if i2 < 1 {
					i2 = len(vertices) + 3*i2
				}
				if i3 < 1 {
					i3 = len(vertices) + 3*i3
				}

				triangles = append(triangles, Triangle {
					A: [3]float32{vertices[i0], vertices[i0 + 1], vertices[i0 + 2]},
					B: [3]float32{vertices[i1], vertices[i1 + 1], vertices[i1 + 2]},
					C: [3]float32{vertices[i2], vertices[i2 + 1], vertices[i2 + 2]},
					Color: curColor,
					Intensity: curIntensity,
				})
				
				// triangulate quad
				if len(curFace) == 4 {	
					triangles = append(triangles, Triangle {
						A: [3]float32{vertices[i0], vertices[i0 + 1], vertices[i0 + 2]},
						B: [3]float32{vertices[i2], vertices[i2 + 1], vertices[i2 + 2]},
						C: [3]float32{vertices[i3], vertices[i3 + 1], vertices[i3 + 2]},
						Color: curColor,
						Intensity: curIntensity,
					})
				}
			} 
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return triangles
}

func parseMtl(path string) []Material {
	materials := []Material{} 
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		cur := scanner.Text()
		if len(cur) > 1 {
			cur = s.ReplaceAll(cur, "\t", " ")
			for {
				cur = s.ReplaceAll(cur, "  ", " ")
				if !s.Contains(cur, "  ") {
					break
				}
			}
			for {
				if cur == " " {
					break
				}
				if string(cur[0]) == " " {
					cur = cur[1:]
				} else {
					break
				}
			}
			if len(cur) > 5 {
				if cur[:6] == "newmtl" {
					materials = append(materials, Material {
						Name: s.Split(cur, " ")[1],
						Color: [3]float32{0, 0, 0},
						Intensity: [3]float32{0, 0, 0},
					})
				} 
			} 
			if string(cur[0]) == "K" && string(cur[1]) == "a" {
				curVertex := s.Split(cur, " ")[1:]
				r, err := strconv.ParseFloat(curVertex[0], 32)
				if err != nil {
					log.Fatal(err)
				}
				g, err := strconv.ParseFloat(curVertex[1], 32)
				if err != nil {
					log.Fatal(err)
				}
				b, err := strconv.ParseFloat(curVertex[2], 32)
				if err != nil {
					log.Fatal(err)
				}
				materials[len(materials)-1].Color = [3]float32{float32(r),float32(g),float32(b)}
			} else if string(cur[0]) == "K" && string(cur[1]) == "e" {
				curVertex := s.Split(cur, " ")[1:]
				r, err := strconv.ParseFloat(curVertex[0], 32)
				if err != nil {
					log.Fatal(err)
				}
				g, err := strconv.ParseFloat(curVertex[1], 32)
				if err != nil {
					log.Fatal(err)
				}
				b, err := strconv.ParseFloat(curVertex[2], 32)
				if err != nil {
					log.Fatal(err)
				}
				materials[len(materials)-1].Intensity = [3]float32{float32(r),float32(g),float32(b)}
			}
		}
	}

	return materials
}