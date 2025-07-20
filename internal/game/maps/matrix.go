package maps

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var Tmap Map
var Matrix [][]bool

func getAppDir() string {
	exe, _ := os.Executable()
	return filepath.Dir(exe)
}

func ReadMap(path string) (Map, error) {
	//pathDef := filepath.Join("map.json")
	pathDef := filepath.Join(getAppDir(), "map.json")
	file, errd := os.Open(pathDef)
	if errd != nil {
		panic(errd)
	}
	defer file.Close()

	var tileMap Map
	if err := json.NewDecoder(file).Decode(&tileMap); err != nil {
		panic(err)
	}

	if len(tileMap.Layers) == 0 {
		return tileMap, fmt.Errorf("map file %s has no layers", path)
	}

	return tileMap, nil
}

func GenerateCollisionMatrix(path string) error {
	Tmap, err := ReadMap(path)
	if err != nil {
		fmt.Printf("Error reading map: %v\n", err)
		return err
	}
	Matrix = make([][]bool, Tmap.Height)
	for i := range Matrix {
		Matrix[i] = make([]bool, Tmap.Width)
	}

	for _, layer := range Tmap.Layers {
		if layer.Name == "Objects" {
			for i, obj := range layer.Data {
				posX := i % Tmap.Width
				posY := i / Tmap.Width

				if obj != 0 {
					Matrix[posY][posX] = true
				} else {
					Matrix[posY][posX] = false
				}
			}
		}
	}
	fmt.Println(Matrix)
	return nil
}
