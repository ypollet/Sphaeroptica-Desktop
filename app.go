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
	Extrinsics        map[string]sph.Extrinsics
	Thumbnails_width  int
	Thumbnails_height int
	Thumbnails        int
}

type virtualCameraImage struct {
	name      string
	image     string
	longitude float64
	latitude  float64
}

type cameraViewer struct {
	images []virtualCameraImage
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

// Get images
func (a *App) Images(projectFile string) cameraViewer {
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
		encodedImages = append(encodedImages, virtualCameraImage{name: image, image: file})

		extrinsics := calibFile.Extrinsics[image].Matrix

		extrinsicsMat := mat.NewDense(extrinsics.Shape.Row, extrinsics.Shape.Col, extrinsics.Matrix)
		rotationMat := mat.DenseCopyOf(extrinsicsMat.Slice(0, 3, 0, 3))
		transMat := mat.DenseCopyOf(extrinsicsMat.Slice(0, 3, 3, 4))
		worldCoord := sph.GetCameraWorldsCoordinates(rotationMat, transMat)
		centersX = append(centersX, worldCoord.At(0, 0))
		centersY = append(centersY, worldCoord.At(1, 0))
		centersZ = append(centersZ, worldCoord.At(2, 0))
	}

	_, center := sph.SphereFit(centersX, centersY, centersZ)

	var centerVecDense mat.VecDense
	centerVecDense.CloneFromVec(center)
	centerVec := centerVecDense.SliceVec(0, 3)

	for index, imageData := range encodedImages {
		imageName := imageData.name
		C := centers[imageName]
		var vector mat.VecDense
		fmt.Printf("Center = %v\n", sph.FormatMatrixPrint(centerVec))
		fmt.Printf("C = %v\n", sph.FormatMatrixPrint(C))

		vector.SubVec(C, centerVec)
		long, lat := sph.GetLongLat(vector)
		imageData.longitude = sph.Rad2Degrees(long)
		imageData.latitude = sph.Rad2Degrees(lat)
		encodedImages[index] = imageData
	}

	return cameraViewer{images: encodedImages}
}
