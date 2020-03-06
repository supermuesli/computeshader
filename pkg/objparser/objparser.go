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
	Color     []float32 
	Intensity float32
}

type Triangle struct {
	A         []float32
	B         []float32
	C         []float32
	Color     []float32
	Intensity float32
}

func GetTriangles(path string) []Triangle {
	vertices := []float32{}
	parsedMtls := []Material{}
	triangles := []Triangle{}

	curColor := []float32{0, 0, 0}
	curIntensity := float32(0.0)

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
	ScanLoop:
	for scanner.Scan() {
		cur := scanner.Text()
		if len(cur) > 1 {
			cur = s.ReplaceAll(cur, "\t", "")
			for {
				cur = s.ReplaceAll(cur, "  ", " ")
				if !s.Contains(cur, "  ") {
					break
				}
			}
			for {
				if cur == " " {
					continue ScanLoop
				}
				if string(cur[0]) == " " {
					cur = cur[1:]
				} else {
					break
				}
			}
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

				curTriangle := Triangle {
					A: []float32{vertices[i0], vertices[i0 + 1], vertices[i0 + 2]},
					B: []float32{vertices[i1], vertices[i1 + 1], vertices[i1 + 2]},
					C: []float32{vertices[i2], vertices[i2 + 1], vertices[i2 + 2]},
					Color: curColor,
					Intensity: curIntensity,
				}
				triangles = append(triangles, curTriangle)
				
				// triangulate quad
				if len(curFace) == 4 {	
					curTriangle := Triangle {
						A: []float32{vertices[i0], vertices[i0 + 1], vertices[i0 + 2]},
						B: []float32{vertices[i2], vertices[i2 + 1], vertices[i2 + 2]},
						C: []float32{vertices[i3], vertices[i3 + 1], vertices[i3 + 2]},
						Color: curColor,
						Intensity: curIntensity,
					}
					triangles = append(triangles, curTriangle)
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
	ScanLoop:
	for scanner.Scan() {
		cur := scanner.Text()
		if len(cur) > 1 {
			cur = s.ReplaceAll(cur, "\t", "")
			for {
				cur = s.ReplaceAll(cur, "  ", " ")
				if !s.Contains(cur, "  ") {
					break
				}
			}
			for {
				if cur == " " {
					continue ScanLoop
				}
				if string(cur[0]) == " " {
					cur = cur[1:]
				} else {
					break
				}
			}
			if len(cur) > 5 {
				if cur[:6] == "newmtl" {
					curMaterial := Material {
						Name: s.Split(cur, " ")[1],
						Color: []float32{0, 0, 0},
						Intensity: 0,
					}
					materials = append(materials, curMaterial)
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
				materials[len(materials)-1].Color = []float32{float32(r),float32(g),float32(b)}
			}
		}
	}

	return materials
}