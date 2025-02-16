package photogrammetry

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
