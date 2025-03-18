package photogrammetry

import "gonum.org/v1/gonum/mat"

type Shape struct {
	Row int
	Col int
}
type MatrixInfo struct {
	Shape  Shape
	Matrix []float64
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
