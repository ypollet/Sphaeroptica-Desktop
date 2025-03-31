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

	"github.com/wailsapp/wails/v2/pkg/runtime"
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
	Commands          map[string]sph.Coordinates
	Intrinsics        sph.Intrinsics
	Extrinsics        map[string]sph.Extrinsics
	Thumbnails_width  int
	Thumbnails_height int
	Thumbnails        string
}

type VirtualCameraImage struct {
	Name        string          `json:"name"`
	FullImage   string          `json:"fullImage"`
	Thumbnail   string          `json:"thumbnail"`
	Coordinates sph.Coordinates `json:"coordinates"`
}

type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type CameraViewer struct {
	Images     []VirtualCameraImage `json:"images"`
	Size       Size                 `json:"size"`
	Thumbnails bool                 `json:"thumbnails"`
}

/*
   pose = reconstruction.project_points(position, intrinsics, extrinsics, dist_coeffs)

   return {
     "pose": {"x": pose.item(0), "y": pose.item(1)}
           }
*/

func (a *App) Reproject(projectFile string, imageName string, position []float64) sph.Pos {
	jsonFile, err := os.Open(projectFile)
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Print(err)
		return sph.Pos{X: -1, Y: -1}
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)
	var calibFile project
	json.Unmarshal([]byte(byteValue), &calibFile)

	vectorPos := mat.NewVecDense(4, position)

	intrinsics := mat.NewDense(calibFile.Intrinsics.CameraMatrix.Shape.Row, calibFile.Intrinsics.CameraMatrix.Shape.Col, calibFile.Intrinsics.CameraMatrix.Data)
	distCoeffs := mat.NewDense(calibFile.Intrinsics.DistortionMatrix.Shape.Row, calibFile.Intrinsics.DistortionMatrix.Shape.Col, calibFile.Intrinsics.DistortionMatrix.Data)
	extrinsics := mat.NewDense(calibFile.Extrinsics[imageName].Matrix.Shape.Row, calibFile.Extrinsics[imageName].Matrix.Shape.Col, calibFile.Extrinsics[imageName].Matrix.Data)

	return sph.ProjectPoints(vectorPos, intrinsics, extrinsics, distCoeffs)
}

func (a *App) Triangulate(projectFile string, poses map[string]sph.Pos) []float64 {
	jsonFile, err := os.Open(projectFile)
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Print(err)
		return []float64{}
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)
	var calibFile project
	json.Unmarshal([]byte(byteValue), &calibFile)

	intrinsics := mat.NewDense(calibFile.Intrinsics.CameraMatrix.Shape.Row, calibFile.Intrinsics.CameraMatrix.Shape.Col, calibFile.Intrinsics.CameraMatrix.Data)
	distCoeffs := mat.NewDense(calibFile.Intrinsics.DistortionMatrix.Shape.Row, calibFile.Intrinsics.DistortionMatrix.Shape.Col, calibFile.Intrinsics.DistortionMatrix.Data)

	projPoints := make([]sph.ProjPoint, 0)

	for image, pos := range poses {
		extrinsics := mat.NewDense(calibFile.Extrinsics[image].Matrix.Shape.Row, calibFile.Extrinsics[image].Matrix.Shape.Col, calibFile.Extrinsics[image].Matrix.Data)
		projMat := sph.ProjectionMatrix(intrinsics, extrinsics)
		pose := mat.NewVecDense(2, []float64{pos.X, pos.Y})
		undistortedPos := sph.UndistortIter(pose, intrinsics, distCoeffs)

		projPoints = append(projPoints, sph.ProjPoint{Mat: projMat, Point: undistortedPos})
	}

	landmarkPos := sph.TriangulatePoint(projPoints)
	return landmarkPos
}

// Get shortcuts
func (a *App) Shortcuts(projectFile string) map[string]sph.Coordinates {
	jsonFile, err := os.Open(projectFile)
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Print(err)
		return map[string]sph.Coordinates{}
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)
	var calibFile project
	json.Unmarshal([]byte(byteValue), &calibFile)

	return calibFile.Commands
}

// Get images
func (a *App) Images(projectFile string) *CameraViewer {
	// Open our jsonFile
	jsonFile, err := os.Open(projectFile)
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Print(err)
		return nil
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var calibFile project
	json.Unmarshal([]byte(byteValue), &calibFile)

	keys := make([]string, 0, len(calibFile.Extrinsics))

	for k := range calibFile.Extrinsics {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	centers := make(map[string]mat.Vector)

	var centersX []float64
	var centersY []float64
	var centersZ []float64

	encodedImages := make([]VirtualCameraImage, 0)

	thumbnails := false
	for _, image := range keys {
		projectDirAbs, _ := filepath.Abs(filepath.Dir(projectFile))
		file := fmt.Sprintf("%s/%s", projectDirAbs, image)
		thumbnail := ""
		if calibFile.Thumbnails != "" {
			thumbnail = fmt.Sprintf("%s/%s/%s", projectDirAbs, calibFile.Thumbnails, image)
			thumbnails = true
		}
		encodedImages = append(encodedImages, VirtualCameraImage{Name: image, FullImage: file, Thumbnail: thumbnail})

		extrinsics := calibFile.Extrinsics[image]

		extrinsicsMat := mat.NewDense(extrinsics.Matrix.Shape.Row, extrinsics.Matrix.Shape.Col, extrinsics.Matrix.Data)
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
	fmt.Printf("Center = %v\n\n", sph.FormatMatrixPrint(centerVec))
	for index, imageData := range encodedImages {
		imageName := imageData.Name
		C := centers[imageName]
		var vector mat.VecDense

		vector.SubVec(C, centerVec)
		long, lat := sph.GetLongLat(vector)
		imageData.Coordinates = sph.Coordinates{
			Longitude: sph.Rad2Degrees(long),
			Latitude:  sph.Rad2Degrees(lat)}
		encodedImages[index] = imageData
	}

	camViewer := CameraViewer{Images: encodedImages, Thumbnails: thumbnails, Size: Size{Width: calibFile.Intrinsics.Width, Height: calibFile.Intrinsics.Height}}
	return &camViewer
}

func (a *App) ImportNewFile() string {
	fmt.Printf("Importing New File\n")
	str, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select File",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Calibration File",
				Pattern:     "*.json",
			},
		},
	})
	if err != nil {
		return ""
	}
	return str
}
