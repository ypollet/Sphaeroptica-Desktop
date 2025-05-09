package imports

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"gonum.org/v1/gonum/mat"
	sph "sphaeroptica.be/photogrammetry/photogrammetry"
)

func ReadIntrinsicMetashape(file string) (*sph.Intrinsics, error) {
	xmlFile, err := os.Open(file)

	// if we os.Open returns an error then handle it
	if err != nil {
		return nil, err
	}
	defer xmlFile.Close()

	byteValue, _ := io.ReadAll(xmlFile)
	var intrinsicFile sph.IntrinsicsXML
	err = xml.Unmarshal([]byte(byteValue), &intrinsicFile)

	if err != nil {
		return nil, err
	}

	cameraDataString := strings.Fields(intrinsicFile.Camera_Matrix.Data)
	cameraData := make([]float64, len(cameraDataString))

	for index := range cameraDataString {
		val, _ := strconv.ParseFloat(cameraDataString[index], 64)
		cameraData[index] = val
	}
	distortionDataString := strings.Fields(intrinsicFile.Distortion_Coefficients.Data)
	distortionData := make([]float64, len(distortionDataString))

	for index := range distortionDataString {
		val, _ := strconv.ParseFloat(distortionDataString[index], 64)
		distortionData[index] = val
	}

	return &sph.Intrinsics{
		Height: intrinsicFile.Image_Height,
		Width:  intrinsicFile.Image_Width,
		CameraMatrix: sph.MatrixInfo{
			Shape: sph.Shape{
				Row: intrinsicFile.Camera_Matrix.Rows,
				Col: intrinsicFile.Camera_Matrix.Cols,
			},
			Data: cameraData,
		},
		DistortionMatrix: sph.MatrixInfo{
			Shape: sph.Shape{
				Row: intrinsicFile.Distortion_Coefficients.Rows,
				Col: intrinsicFile.Distortion_Coefficients.Cols,
			},
			Data: distortionData,
		},
	}, nil
}

func ReadExtrinsicMetashape(file string, images map[string]string) (map[string]sph.Extrinsics, float64, float64, error) {

	f, err := os.Open(file)
	if err != nil {
		return nil, 0, 0, err
	}
	defer f.Close()

	var centersX []float64
	var centersY []float64
	var centersZ []float64

	centers := make(map[string]mat.Vector)

	csvReader := csv.NewReader(f)
	csvReader.Comma = '\t'
	csvReader.Comment = '#'

	extMap := make(map[string]sph.Extrinsics)

	// header = {"Label", "X", "Y", "Z", "Omega", "Phi", "Kappa", "r11", "r12", "r13", "r21", "r22", "r23", "r31", "r32", "r33"}
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		X, _ := strconv.ParseFloat(record[1], 64)
		Y, _ := strconv.ParseFloat(record[2], 64)
		Z, _ := strconv.ParseFloat(record[3], 64)

		r11, _ := strconv.ParseFloat(record[7], 64)
		r12, _ := strconv.ParseFloat(record[8], 64)
		r13, _ := strconv.ParseFloat(record[9], 64)
		r21, _ := strconv.ParseFloat(record[10], 64)
		r22, _ := strconv.ParseFloat(record[11], 64)
		r23, _ := strconv.ParseFloat(record[12], 64)
		r31, _ := strconv.ParseFloat(record[13], 64)
		r32, _ := strconv.ParseFloat(record[14], 64)
		r33, _ := strconv.ParseFloat(record[15], 64)

		rotMat := mat.NewDense(3, 3, []float64{r11, r12, r13, r21, r22, r23, r31, r32, r33})
		rotMat.Mul(sph.RotateXAxis(math.Pi), rotMat)

		// t = np.array(-mat.dot(t_w)).T
		var transMat mat.Dense
		transMat.Mul(rotMat, mat.NewDense(3, 1, []float64{X, Y, Z}))
		transMat.Sub(mat.NewDense(3, 1, []float64{0, 0, 0}), &transMat)

		worldCoord := sph.GetCameraWorldsCoordinates(rotMat, &transMat)
		centersX = append(centersX, worldCoord.AtVec(0))
		centersY = append(centersY, worldCoord.AtVec(1))
		centersZ = append(centersZ, worldCoord.AtVec(2))
		centers[images[record[0]]] = worldCoord

		extMap[images[record[0]]] = sph.Extrinsics{
			Matrix: sph.MatrixInfo{Shape: sph.Shape{Row: 3, Col: 4},
				Data: []float64{
					rotMat.At(0, 0),
					rotMat.At(0, 1),
					rotMat.At(0, 2),
					transMat.At(0, 0),
					rotMat.At(1, 0),
					rotMat.At(1, 1),
					rotMat.At(1, 2),
					transMat.At(1, 0),
					rotMat.At(2, 0),
					rotMat.At(2, 1),
					rotMat.At(2, 2),
					transMat.At(2, 0),
				},
			},
		}
	}

	_, center := sph.SphereFit(centersX, centersY, centersZ)
	var centerVecDense mat.VecDense
	centerVecDense.CloneFromVec(center)
	centerVec := centerVecDense.SliceVec(0, 3)

	latMin := 90.0
	latMax := -90.0
	for _, C := range centers {
		var vector mat.VecDense

		vector.SubVec(C, centerVec)
		_, lat := sph.GetLongLat(vector)
		lat = sph.Rad2Degrees(lat)
		if lat < latMin {
			latMin = lat
		}
		if lat > latMax {
			latMax = lat
		}
	}

	return extMap, latMin, latMax, nil
}
