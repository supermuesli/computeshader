package objparser

import (
	"bufio"
	"os"
	"log"
	"strconv"
	s "strings"
)

func Stream(path string) []float64 {
	vertices := []float64{}

	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cur := scanner.Text()
		if len(cur) > 0 {
			if string(cur[0]) == "v" {
				curVertex := s.Split(cur, " ")[1:]
				v0, err := strconv.ParseFloat(curVertex[0], 64)
				if err != nil {
					log.Fatal(err)
				}
				v1, err := strconv.ParseFloat(curVertex[1], 64)
				if err != nil {
					log.Fatal(err)
				}
				v2, err := strconv.ParseFloat(curVertex[2], 64)
				if err != nil {
					log.Fatal(err)
				}
				vertices = append(vertices, []float64{v0, v1, v2}...)
			} else {
				continue
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return vertices
}