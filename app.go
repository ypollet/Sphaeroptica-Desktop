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
	imp "sphaeroptica.be/imports/imports"
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
	Commands         map[string]sph.Coordinates
	Intrinsics       sph.Intrinsics
	Extrinsics       map[string]sph.Extrinsics
	ThumbnailsWidth  int
	ThumbnailsHeight int
	Thumbnails       string
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

type Filter struct {
	Images     []VirtualCameraImage `json:"images"`
	Size       Size                 `json:"size"`
	Thumbnails bool                 `json:"thumbnails"`
}
type Type int
type ImportFile struct {
	Name    string
	Label   string
	Filters []runtime.FileFilter
	Type    Type
}

type ImportForm struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	File  bool   `json:"file"`
}

const (
	NONE = iota
	FILE
	FOLDER
)

var defaultDirectory = ""

type ImportTemplate struct {
	Files []ImportFile
}

var IMPORTS_FILES = map[string][]ImportFile{
	"Metashape": {
		{
			Name:    "Images",
			Label:   "Images Folder",
			Type:    FOLDER,
			Filters: []runtime.FileFilter{},
		},
		{
			Name:    "Thumbnails",
			Label:   "Thumbnails Folder",
			Type:    NONE,
			Filters: []runtime.FileFilter{},
		},
		{
			Name:  "Intrinsics",
			Label: "Intrinsics File OPENCV Format",
			Type:  FILE,
			Filters: []runtime.FileFilter{
				{
					DisplayName: "Intrinsic file (*.xml)",
					Pattern:     "*.xml",
				},
			},
		},
		{
			Name:  "Extrinsics",
			Label: "Extrinsics File ODK",
			Type:  FILE,
			Filters: []runtime.FileFilter{
				{
					DisplayName: "Extrinsic file (*.txt)",
					Pattern:     "*.txt",
				},
			},
		},
	},
}

var IMPORTS_READER = map[string]func(map[string]string) (*project, string, []imp.SaveThumbnail){
	"Metashape": ReadMetashape,
}

func ReadMetashape(files map[string]string) (*project, string, []imp.SaveThumbnail) {
	log.Println("Read Metashape Log")
	if len(files) != len(IMPORTS_FILES["Metashape"]) {
		log.Printf("File length incorrect\n")
		log.Printf("%d, %d\n", len(files), len(IMPORTS_FILES["Metashape"]))
		return nil, "", nil
	}
	for _, importFile := range IMPORTS_FILES["Metashape"] {
		if _, ok := files[importFile.Name]; !ok || len(files[importFile.Name]) == 0 {
			log.Printf("File Names incorrect\n")
			log.Printf("%sn", importFile.Name)
			log.Printf("%sn", files[importFile.Name])
			return nil, "", nil
		}
	}

	imagesDir := files["Images"]
	thumbnailsDir := files["Thumbnails"]

	thumbnails := fmt.Sprintf("%s/%s", imagesDir, thumbnailsDir)
	if err := os.MkdirAll(thumbnails, os.ModePerm); err != nil {
		log.Println(err)
		return nil, "", nil
	}

	images, thumbWidth, thumbHeight, thumbCreate, err := imp.ReadChildImages(imagesDir, thumbnailsDir)
	if err != nil {
		log.Println(err)
		return nil, "", nil
	}

	intrinsics, err := imp.ReadIntrinsicMetashape(files["Intrinsics"])
	if err != nil {
		log.Println(err)
		return nil, "", nil
	}

	extrinsics, latMin, latMax, err := imp.ReadExtrinsicMetashape(files["Extrinsics"], images)
	if err != nil {
		log.Println(err)
		return nil, "", nil
	}

	return &project{
		Commands: map[string]sph.Coordinates{
			"FRONT":    {Longitude: 0, Latitude: 0},
			"POST":     {Longitude: 180, Latitude: 0},
			"LEFT":     {Longitude: 90, Latitude: 0},
			"RIGHT":    {Longitude: -90, Latitude: 0},
			"SUPERIOR": {Longitude: 0, Latitude: latMin},
			"INFERIOR": {Longitude: 180, Latitude: latMax},
		},
		Intrinsics:       *intrinsics,
		Extrinsics:       extrinsics,
		Thumbnails:       thumbnailsDir,
		ThumbnailsWidth:  thumbWidth,
		ThumbnailsHeight: thumbHeight,
	}, imagesDir, thumbCreate
}

func (a *App) GetImportMethods() map[string][]ImportForm {
	imports := make(map[string][]ImportForm)
	for software, files := range IMPORTS_FILES {
		filenames := make([]ImportForm, len(files))
		for index, file := range files {
			filenames[index] = ImportForm{
				Name:  file.Name,
				Label: file.Label,
				File:  file.Type != NONE,
			}
		}
		imports[software] = filenames
	}
	return imports
}

func (a *App) ImportProject(software string, files map[string]string) string {
	log.Printf("Import Project from %s\n", software)
	project, imagesDir, thumbCreate := IMPORTS_READER[software](files)

	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		log.Println(err)
		return ""
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultDirectory: imagesDir,
		DefaultFilename:  "sphaeroptica.json",
		Filters: []runtime.FileFilter{{
			DisplayName: ".json",
			Pattern:     "*.json",
		}},
	})
	if err != nil {
		log.Println(err)
		return ""
	}
	thumbPath := fmt.Sprintf("%s/%s", imagesDir, project.Thumbnails)
	project.ThumbnailsWidth, project.ThumbnailsHeight, err = imp.CreateThumbnails(thumbPath, thumbCreate, project.ThumbnailsWidth, project.ThumbnailsHeight)
	if err != nil {
		log.Println("Error while creating thumbnails")
		log.Println(err)
		return ""
	}
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		log.Println(err)
		return ""
	}

	return path
}

func (a *App) OpenImportFile(software string, index int) string {
	importFile := IMPORTS_FILES[software][index]
	str := ""

	switch importFile.Type {
	case FILE:
		str = a.openFileDialog("Select "+importFile.Label, importFile.Filters)
	case FOLDER:
		str = a.openDirectoryDialog("Select "+importFile.Label, importFile.Filters)
	}

	return str
}

func (a *App) Reproject(projectFile string, imageName string, position []float64) sph.Pos {
	jsonFile, err := os.Open(projectFile)
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Println(err)
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
		log.Println(err)
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
		log.Println(err)
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
		log.Println(err)
		return nil
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var calibFile project
	json.Unmarshal([]byte(byteValue), &calibFile)

	log.Printf("Project : \n%v\n", calibFile)

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
	log.Printf("Center = %v\n\n", sph.FormatMatrixPrint(centerVec))
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
	str := a.openFileDialog("Select Project File", []runtime.FileFilter{
		{
			DisplayName: "Calibration File",
			Pattern:     "*.json",
		},
	},
	)

	return str
}

func (a *App) openFileDialog(title string, filters []runtime.FileFilter) string {
	str, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		DefaultDirectory: defaultDirectory,
		Title:            title,
		Filters:          filters,
	})
	defaultDirectory = filepath.Dir(str)
	if err != nil {
		return ""
	}
	return str
}

func (a *App) openDirectoryDialog(title string, filters []runtime.FileFilter) string {
	str, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		DefaultDirectory: defaultDirectory,
		Title:            title,
		Filters:          filters,
	})
	defaultDirectory = str
	if err != nil {
		return ""
	}
	return str
}
