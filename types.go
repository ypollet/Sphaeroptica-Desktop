package main

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"
	sph "sphaeroptica.be/photogrammetry/photogrammetry"
)

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

// Import File structs

type Filter struct {
	Images     []VirtualCameraImage `json:"images"`
	Size       Size                 `json:"size"`
	Thumbnails bool                 `json:"thumbnails"`
}
type Type int

const (
	NONE = iota
	FILE
	FOLDER
)

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

type ImportTemplate struct {
	Files []ImportFile
}

// Import landmarks JSON
type ExportJSON struct {
	ScaleFactor float64                 `json:"scaleFactor"`
	Landmarks   map[string]LandmarkJSON `json:"landmarks"`
	Distances   []DistanceJSON          `json:"distances"`
}

type LandmarkJSON struct {
	Label    string              `json:"label"`
	Color    string              `json:"color"`
	Position []float64           `json:"position"`
	Poses    map[string]PoseJSON `json:"poses"`
}

type PoseJSON struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type DistanceJSON struct {
	Label string `json:"label"`
	Left  string `json:"left"`
	Right string `json:"right"`
}

// Import landmarks CSV
type LandmarkCSV struct {
	Label     string `json:"label"`
	Color     string `json:"color"`
	X         string `json:"x"`
	Y         string `json:"y"`
	Z         string `json:"z"`
	XAdjusted string `json:"x_adjusted"`
	YAdjusted string `json:"y_adjusted"`
	ZAdjusted string `json:"z_adjusted"`
}
