package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"

	"gonum.org/v1/gonum/mat"
	sph "sphaeroptica.be/photogrammetry/photogrammetry"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

type project struct {
	Commands          map[string]string
	Intrinsics        sph.Intrinsics
	Extrinsics        map[string]sph.MatrixInfo
	Thumbnails_width  int
	Thumbnails_height int
	Thumbnails        string
}

type virtualCameraImage struct {
	Name      string  `json:"name"`
	FullImage string  `json:"fullImage"`
	Thumbnail string  `json:"thumbnail"`
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

type cameraViewer struct {
	Images []virtualCameraImage `json:"images"`
}

// Get images
func (a *App) Shortcuts(projectFile string) map[string]string {
	jsonFile, err := os.Open(projectFile)
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)
	var calibFile project
	json.Unmarshal([]byte(byteValue), &calibFile)

	fmt.Printf("%v\n", calibFile.Commands)

	return calibFile.Commands
}

func (a *App) Greet() string {
	return "Hello World!"
}

// Get images
func (a *App) Images(projectFile string) *cameraViewer {
	fmt.Printf("Checking : %s\n", projectFile)

	// Open our jsonFile
	jsonFile, err := os.Open(projectFile)
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var calibFile project
	json.Unmarshal([]byte(byteValue), &calibFile)

	fmt.Printf("Intrinsics : %+v\n", calibFile.Intrinsics)

	keys := make([]string, 0, len(calibFile.Extrinsics))

	for k := range calibFile.Extrinsics {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Printf("Number Images = %d\n", len(keys))

	centers := make(map[string]mat.Vector)

	var centersX []float64
	var centersY []float64
	var centersZ []float64

	encodedImages := make([]virtualCameraImage, 0)

	for _, image := range keys {
		file := fmt.Sprintf("%s/%s", filepath.Dir(projectFile), image)
		thumbnail := fmt.Sprintf("%s/%s/%s", filepath.Dir(projectFile), calibFile.Thumbnails, image)
		encodedImages = append(encodedImages, virtualCameraImage{Name: image, FullImage: file, Thumbnail: thumbnail})

		extrinsics := calibFile.Extrinsics[image]

		extrinsicsMat := mat.NewDense(extrinsics.Shape.Row, extrinsics.Shape.Col, extrinsics.Matrix)
		rotationMat := mat.DenseCopyOf(extrinsicsMat.Slice(0, 3, 0, 3))
		transMat := mat.DenseCopyOf(extrinsicsMat.Slice(0, 3, 3, 4))
		worldCoord := sph.GetCameraWorldsCoordinates(rotationMat, transMat)
		centersX = append(centersX, worldCoord.AtVec(0))
		centersY = append(centersY, worldCoord.AtVec(1))
		centersZ = append(centersZ, worldCoord.AtVec(2))
		centers[image] = worldCoord
	}

	_, center := sph.SphereFit(centersX, centersY, centersZ)

	var centerVecDense mat.VecDense
	centerVecDense.CloneFromVec(center)
	centerVec := centerVecDense.SliceVec(0, 3)

	for index, imageData := range encodedImages {
		imageName := imageData.Name
		C := centers[imageName]
		var vector mat.VecDense

		vector.SubVec(C, centerVec)
		long, lat := sph.GetLongLat(vector)
		imageData.Longitude = sph.Rad2Degrees(long)
		imageData.Latitude = sph.Rad2Degrees(lat)
		encodedImages[index] = imageData
	}

	camViewer := cameraViewer{Images: encodedImages}
	fmt.Printf("%v\n", camViewer)
	return &camViewer
}
