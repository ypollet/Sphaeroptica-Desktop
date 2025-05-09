package photogrammetry

import (
	"encoding/xml"

	"gonum.org/v1/gonum/mat"
)

type Shape struct {
	Row int
	Col int
}

type MatrixInfo struct {
	Shape Shape
	Data  []float64
}

type Extrinsics struct {
	Matrix MatrixInfo
}

type Intrinsics struct {
	Height           int
	Width            int
	CameraMatrix     MatrixInfo
	DistortionMatrix MatrixInfo
}

type Coordinates struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

type Pos struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type ProjPoint struct {
	Mat   mat.Matrix
	Point mat.Vector
}

// Import Files Types struct

type IntrinsicsXML struct {
	XMLName                 xml.Name   `xml:"opencv_storage"`
	Image_Width             int        `xml:"image_Width"`
	Image_Height            int        `xml:"image_Height"`
	Camera_Matrix           MatrixData `xml:"Camera_Matrix"`
	Distortion_Coefficients MatrixData `xml:"Distortion_Coefficients"`
}

type MatrixData struct {
	Rows int    `xml:"rows"`
	Cols int    `xml:"cols"`
	Dt   string `xml:"dt"`
	Data string `xml:"data"`
}
