package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"

	"gonum.org/v1/gonum/mat"
	"sphaeroptica.be/photogrammetry/photogrammetry"
	sph "sphaeroptica.be/photogrammetry/photogrammetry"
)

//go:embed all:frontend/dist
var assets embed.FS

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

func main() {
	projectFile := "/home/psadmin/Numerisation/Sphaeroptica/desktop/data/papillon_big/calibration.json"
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

	keys := make([]string, 0)

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
		encodedImages = append(encodedImages, virtualCameraImage{name: image, image: image})

		extrinsics := calibFile.Extrinsics[image].Matrix

		extrinsicsMat := mat.NewDense(extrinsics.Shape.Row, extrinsics.Shape.Col, extrinsics.Matrix)
		rotationMat := mat.DenseCopyOf(extrinsicsMat.Slice(0, 3, 0, 3))
		transMat := mat.DenseCopyOf(extrinsicsMat.Slice(0, 3, 3, 4))
		worldCoord := sph.GetCameraWorldsCoordinates(rotationMat, transMat)

		centers[image] = worldCoord
		centersX = append(centersX, worldCoord.AtVec(0))
		centersY = append(centersY, worldCoord.AtVec(1))
		centersZ = append(centersZ, worldCoord.AtVec(2))
	}

	radius, center := sph.SphereFit(centersX, centersY, centersZ)

	var centerVecDense mat.VecDense
	centerVecDense.CloneFromVec(center)
	centerVec := centerVecDense.SliceVec(0, 3)
	fmt.Printf("Radius = %f\n", radius)
	fmt.Printf("Center = %v\n", sph.FormatMatrixPrint(center))

	for index, imageData := range encodedImages {
		imageName := imageData.name
		C := centers[imageName]
		var vector mat.VecDense
		fmt.Printf("Center = %v\n", sph.FormatMatrixPrint(centerVec))
		fmt.Printf("C = %v\n", sph.FormatMatrixPrint(C))

		vector.SubVec(C, centerVec)
		long, lat := photogrammetry.GetLongLat(vector)
		imageData.longitude = photogrammetry.Rad2Degrees(long)
		imageData.latitude = photogrammetry.Rad2Degrees(lat)
		encodedImages[index] = imageData
	}
	fmt.Printf("%v\n\n", encodedImages)

	/*
		// Create an instance of the app structure
		app := NewApp()

		// Create application with options
		err := wails.Run(&options.App{
			Title:  "sphaeroptica",
			Width:  1024,
			Height: 768,
			AssetServer: &assetserver.Options{
				Assets: assets,
			},
			BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
			OnStartup:        app.startup,
			Bind: []interface{}{
				app,
			},
		})

		if err != nil {
			println("Error:", err.Error())
		}
	*/
}
