package photogrammetry

import (
	"log"
	"math"
	"slices"

	"gonum.org/v1/gonum/mat"
)

// Method by Charles Jekel, I just used SVD to solve the least squared problem
//
// Args:
// spX ([]float32): list of positions of all the cameras on the X axis
// spY ([]float32): list of positions of all the cameras on the Y axis
// spZ ([]float32): list of positions of all the cameras on the Z axis
//
//	Returns:
//	float32: radius of the sphere
//	[]float32: center of the sphere
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

/*
def sphereFit(spX,spY,spZ):
    #   Assemble the A matrix
    spX = np.array(spX)
    spY = np.array(spY)
    spZ = np.array(spZ)
    A = np.zeros((len(spX),4))
    A[:,0] = spX*2
    A[:,1] = spY*2
    A[:,2] = spZ*2
    A[:,3] = 1

    #   Assemble the b matrix
    b = np.zeros((len(spX),1))
    b[:,0] = (spX*spX) + (spY*spY) + (spZ*spZ)

    #solve SVD
    U, s, Vh = np.linalg.svd(A, full_matrices = False)
    b_prime = np.transpose(U)@b

    y = (b_prime.T/s).T

    C = np.transpose(Vh)@y
    t = (C[0]*C[0])+(C[1]*C[1])+(C[2]*C[2])+C[3]
    radius = math.sqrt(t)

    return radius, C[0:3]
*/
