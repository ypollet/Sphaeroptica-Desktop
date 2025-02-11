package photogrammetry

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
)

func FormatMatrixPrint(matrix mat.Matrix) fmt.Formatter {
	return mat.Formatted(matrix, mat.Prefix("    "), mat.Squeeze())
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func Degrees2Rad(deg float64) float64 {
	res := deg * math.Pi / 180
	return roundFloat(res, 10)
}

func Rad2Degrees(rad float64) float64 {
	res := rad * 180 / math.Pi
	return roundFloat(res, 10)
}

func GetCameraWorldsCoordinates(rotation *mat.Dense, trans *mat.Dense) mat.Vector {
	var coordinates mat.Dense
	coordinates.Mul(rotation.T(), trans)
	coordinates.Scale(-1, &coordinates)
	return coordinates.ColView(0)
}

func GetLongLat(vector mat.VecDense) (float64, float64) {
	norm := vector.Norm(2)
	vector.ScaleVec(1/norm, &vector)
	x := vector.At(0, 0)
	y := vector.At(1, 0)
	z := vector.At(2, 0)

	latitude := math.Atan2(z, math.Sqrt(math.Pow(x, 2)+math.Pow(y, 2)))
	longitude := math.Atan2(y, x)
	return longitude, latitude
}

/*


def get_long_lat(vector : np.ndarray) -> tuple[float, float]:
    """get geographic coordinates from a vector (centered at the origin (0,0,0))

    Args:
        vector (np.ndarray): given vector

    Returns:
        float: longitude
        float: latitude
    """

    C_normed = vector / np.linalg.norm(vector)
    x,y,z = C_normed.reshape((3,1)).tolist()
    x = x[0]
    y = y[0]
    z = z[0]
    latitude = math.atan2(z, math.sqrt(x**2 + y**2))
    longitude = math.atan2(y,x)
    return longitude, latitude
*/
