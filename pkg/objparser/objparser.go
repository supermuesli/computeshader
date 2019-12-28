package objparser

import (
	"fmt"
	"bufio"
	"os"
	"log"
	"strconv"
	s "strings"
)

func Stream(path string) []float32 {
	vertices := []float32{}
	faces := []float32{}

	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cur := scanner.Text()
		if len(cur) > 0 {
			cur = s.ReplaceAll(cur, "\t", "")
			for {
				if s.Contains(cur, "  ") {
					cur = s.ReplaceAll(cur, "  ", " ")
				} else {
					break
				}
			}
			fmt.Println(cur)
			if string(cur[0]) == "v" && string(cur[1]) == " " {
				curVertex := s.Split(cur, " ")
				fmt.Println(curVertex)
				curVertex = curVertex[len(curVertex)-3:]
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

				faces = append(faces, []float32 {
					vertices[i0], vertices[i0 + 1], vertices[i0 + 2], 
					vertices[i1], vertices[i1 + 1], vertices[i1 + 2], 
					vertices[i2], vertices[i2 + 1], vertices[i2 + 2]}...
				)
				if len(curFace) == 4 {	
					faces = append(faces, []float32 {
						vertices[i3], vertices[i3 + 1], vertices[i3 + 2],
						vertices[i0], vertices[i0 + 1], vertices[i0 + 2], 
						vertices[i2], vertices[i2 + 1], vertices[i2 + 2]}...
					)
				}
			} else {
				continue
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Println(faces)
	return faces
}