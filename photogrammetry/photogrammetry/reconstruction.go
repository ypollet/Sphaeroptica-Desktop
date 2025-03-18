package photogrammetry

import (
	"log"
	"math"
	"slices"

	"gonum.org/v1/gonum/mat"
)

const OPENCV_DISTORT_VALUES = 8
const MAX_ITER = 100

func scaleHomogeonousPoint(point mat.Vector) mat.Vector {
	var vector mat.VecDense
	vector.ScaleVec((1 / point.AtVec(point.Len()-1)), point)
	return &vector
}

func normalizePixel(point mat.Vector, intrinsics mat.Matrix) (float64, float64) {
	x_u, y_u := point.AtVec(0), point.AtVec(1)

	fx, fy := intrinsics.At(0, 0), intrinsics.At(1, 1)
	cx, cy := intrinsics.At(0, 2), intrinsics.At(1, 2)

	x_u = (x_u - cx) / fx
	y_u = (y_u - cy) / fy

	return x_u, y_u
}

func denormalizePixel(normPoint mat.Vector, intrinsics mat.Matrix) mat.Vector {
	x, y := normPoint.AtVec(0), normPoint.AtVec(1)
	fx, fy := intrinsics.At(0, 0), intrinsics.At(1, 1)
	cx, cy := intrinsics.At(0, 2), intrinsics.At(1, 2)

	return mat.NewVecDense(2, []float64{x*fx + cx, y*fy + cy})
}

func ProjectionMatrix(intrinsics mat.Matrix, extrinsics mat.Matrix) mat.Matrix {
	extrinsicsVecMat := mat.DenseCopyOf(extrinsics)
	extrinsics = extrinsicsVecMat.Slice(0, 3, 0, 4)
	var projMat mat.Dense
	projMat.Mul(intrinsics, extrinsics)

	return &projMat
}

func UndistortIter(point mat.Vector, intrinsics mat.Matrix, distCoeffs mat.Matrix) mat.Vector {
	allDistCoeffs := make([]float64, OPENCV_DISTORT_VALUES)
	distCoeffsDense := mat.DenseCopyOf(distCoeffs)
	for index, coeff := range distCoeffsDense.RawMatrix().Data {
		allDistCoeffs[index] = coeff
	}
	k1, k2, p1, p2, k3, k4, k5, k6 := allDistCoeffs[0], allDistCoeffs[1], allDistCoeffs[2], allDistCoeffs[3], allDistCoeffs[4], allDistCoeffs[5], allDistCoeffs[6], allDistCoeffs[7]

	x, y := normalizePixel(point, intrinsics)
	x0 := x
	y0 := y

	for _ = range MAX_ITER {
		r2 := math.Pow(x, 2) + math.Pow(y, 2)
		k_inv := (1 + k4*r2 + k5*math.Pow(r2, 2) + k6*math.Pow(r2, 3)) / (1 + k1*r2 + k2*math.Pow(r2, 2) + k3*math.Pow(r2, 3))
		delta_x := 2*p1*x*y + p2*(r2+2*math.Pow(x, 2))
		delta_y := p1*(r2+2*math.Pow(y, 2)) + 2*p2*x*y
		xant := x
		yant := y
		x = (x0 - delta_x) * k_inv
		y = (y0 - delta_y) * k_inv
		e := math.Pow((xant-x), 2) + math.Pow((yant-y), 2)
		if e == 0 {
			break
		}
	}
	vec := mat.NewVecDense(2, []float64{x, y})
	return denormalizePixel(vec, intrinsics)
}

// Non Linear from Amy Tabb
func distort(point mat.Vector, intrinsics mat.Matrix, distCoeffs mat.Matrix) mat.Vector {
	// Non linear algorithm of lens distortion (explained by Amy Tabb)
	allDistCoeffs := make([]float64, OPENCV_DISTORT_VALUES)
	distCoeffsDense := mat.DenseCopyOf(distCoeffs)
	for index, coeff := range distCoeffsDense.RawMatrix().Data {
		allDistCoeffs[index] = coeff
	}
	k1, k2, p1, p2, k3, k4, k5, k6 := allDistCoeffs[0], allDistCoeffs[1], allDistCoeffs[2], allDistCoeffs[3], allDistCoeffs[4], allDistCoeffs[5], allDistCoeffs[6], allDistCoeffs[7]

	x_u, y_u := normalizePixel(point, intrinsics)

	r2 := math.Pow(x_u, 2) + math.Pow(y_u, 2)
	x := (x_u * (1 + k1*r2 + k2*(math.Pow(r2, 2)) + k3*(math.Pow(r2, 3))) / (1 + k4*r2 + k5*(math.Pow(r2, 2)) + k6*(math.Pow(r2, 3)))) + 2*p1*x_u*y_u + p2*(r2+2*(math.Pow(x_u, 2)))
	y := (y_u * (1 + k1*r2 + k2*(math.Pow(r2, 2)) + k3*(math.Pow(r2, 3))) / (1 + k4*r2 + k5*(math.Pow(r2, 2)) + k6*(math.Pow(r2, 3)))) + 2*p2*x_u*y_u + p1*(r2+2*(math.Pow(y_u, 2)))

	vec := mat.NewVecDense(2, []float64{x, y})
	return denormalizePixel(vec, intrinsics)
}

func ProjectPoints(position mat.Vector, intrinsics mat.Matrix, extrinsics mat.Matrix, distCoeffs mat.Matrix) Pos {
	var projMat mat.Dense
	var point mat.Dense
	projMat.Mul(intrinsics, extrinsics)

	point.Mul(&projMat, position)

	pointVec := mat.VecDenseCopyOf(scaleHomogeonousPoint(point.ColView(0)))

	pos := pointVec.SliceVec(0, pointVec.Len()-1)

	pos = distort(pos, intrinsics, distCoeffs)
	return Pos{X: pos.AtVec(0), Y: pos.AtVec(1)}
}

func TriangulatePoint(projPoints []ProjPoint) []float64 {
	var A mat.Dense

	if len(projPoints) < 2 {
		return nil
	}

	for _, projPoint := range projPoints {
		projMat := mat.DenseCopyOf(projPoint.Mat)
		point := projPoint.Point
		var view mat.Dense
		var row1 mat.Dense
		var row2 mat.Dense
		projMat.RowView(2)

		row1.Scale(point.AtVec(1), projMat.Slice(2, 3, 0, projMat.RawMatrix().Cols))
		row1.Sub(&row1, projMat.Slice(1, 2, 0, projMat.RawMatrix().Cols))

		row2.Scale(point.AtVec(0), projMat.Slice(2, 3, 0, projMat.RawMatrix().Cols))
		row2.Sub(projMat.Slice(0, 1, 0, projMat.RawMatrix().Cols), &row2)

		view.Stack(&row1, &row2)

		if A.IsEmpty() {
			A = view
		} else {
			var copyA mat.Dense
			copyA.CloneFrom(&A)
			A.Reset()
			A.Stack(&copyA, &view)
		}

	}

	var svd mat.SVD
	ok := svd.Factorize(&A, mat.SVDThin)
	if !ok {
		log.Fatal("failed to factorize A")
	}
	var matrixV mat.Dense
	Vh := &matrixV

	svd.VTo(Vh)

	transposedV := Vh.T()
	VhTransposed := mat.DenseCopyOf(transposedV)
	X := VhTransposed.RowView(VhTransposed.RawMatrix().Rows - 1)

	scaledX := scaleHomogeonousPoint(X)
	return []float64{scaledX.AtVec(0), scaledX.AtVec(1), scaledX.AtVec(2), scaledX.AtVec(3)}
}

// Method by Charles Jekel, I just used SVD to solve the least squared problem
//
// Args:
// spX ([]float64): list of positions of all the cameras on the X axis
// spY ([]float64): list of positions of all the cameras on the Y axis
// spZ ([]float64): list of positions of all the cameras on the Z axis
//
//	Returns:
//	float64: radius of the sphere
//	[]float64: center of the sphere
func SphereFit(spX []float64, spY []float64, spZ []float64) (float64, mat.Vector) {
	// Assemble the A matrix
	var ok bool

	vecX := mat.NewDense(1, len(spX), slices.Clone(spX))
	vecX.Scale(2, vecX)
	vecY := mat.NewDense(1, len(spY), slices.Clone(spY))
	vecY.Scale(2, vecY)
	vecZ := mat.NewDense(1, len(spZ), slices.Clone(spZ))
	vecZ.Scale(2, vecZ)

	A := mat.NewDense(len(spX), 4, nil)
	A.SetCol(0, vecX.RawRowView(0))
	A.SetCol(1, vecY.RawRowView(0))
	A.SetCol(2, vecZ.RawRowView(0))
	A.SetCol(3, slices.Repeat([]float64{1}, len(spX)))

	vecXSquared := mat.NewDense(1, len(spX), spX)
	vecYSquared := mat.NewDense(1, len(spY), spY)
	vecZSquared := mat.NewDense(1, len(spZ), spZ)
	sumVec := mat.NewDense(1, vecX.RawMatrix().Cols, nil)

	vecXSquared.MulElem(vecXSquared, vecXSquared)
	vecYSquared.MulElem(vecYSquared, vecYSquared)
	vecZSquared.MulElem(vecZSquared, vecZSquared)

	sumVec.Add(vecXSquared, vecYSquared)
	sumVec.Add(sumVec, vecZSquared)

	// Assemble the b matrix
	b := mat.NewDense(len(spX), 1, nil)
	b.SetCol(0, sumVec.RawRowView(0))

	var svd mat.SVD
	ok = svd.Factorize(A, mat.SVDThin)
	if !ok {
		log.Fatal("failed to factorize A")
	}

	var matrixU mat.Dense
	var matrixV mat.Dense
	var bPrime mat.Dense
	var y mat.Dense
	U := &matrixU
	Vh := &matrixV

	svd.UTo(U)
	svd.VTo(Vh)
	bPrime.Mul(U.T(), b)

	s := svd.Values(nil)

	y.DivElem(bPrime.T(), mat.NewDense(1, len(s), s))
	transY := y.T()

	var center mat.Dense
	center.Mul(Vh, transY)

	t := (center.At(0, 0) * center.At(0, 0)) + (center.At(1, 0) * center.At(1, 0)) + (center.At(2, 0) * center.At(2, 0)) + center.At(3, 0)

	center.Scale(1/center.At(3, 0), &center)
	return math.Sqrt(t), center.ColView(0)
}
