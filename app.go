package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"

	"gonum.org/v1/gonum/mat"
	"sphaeroptica.be/photogrammetry/photogrammetry"
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

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

type cameraViewer struct {
	images []virtualCameraImage
}

// Get images
func (a *App) images(projectFile string) cameraViewer {
	// Open our jsonFile
	jsonFile, err := os.Open(projectFile)
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var calibFile project
	json.Unmarshal([]byte(byteValue), &calibFile)

	fmt.Printf("Intrinsics : %+v\n", calibFile.Intrinsics)

	keys := make([]string, 0, len(calibFile.Extrinsics))

	for k := range calibFile.Extrinsics {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Printf("Number Images = %d\n", len(keys))

	centers := make(map[string]mat.Vector)

	var centersX []float64
	var centersY []float64
	var centersZ []float64

	encodedImages := make([]virtualCameraImage, 0)

	for _, image := range keys {

		extrinsics := calibFile.Extrinsics[image].Matrix

		extrinsicsMat := mat.NewDense(extrinsics.Shape.Row, extrinsics.Shape.Col, extrinsics.Matrix)
		rotationMat := mat.DenseCopyOf(extrinsicsMat.Slice(0, 3, 0, 3))
		transMat := mat.DenseCopyOf(extrinsicsMat.Slice(0, 3, 3, 4))
		worldCoord := sph.GetCameraWorldsCoordinates(rotationMat, transMat)
		centersX = append(centersX, worldCoord.At(0, 0))
		centersY = append(centersY, worldCoord.At(1, 0))
		centersZ = append(centersZ, worldCoord.At(2, 0))
	}

	_, center := sph.SphereFit(centersX, centersY, centersZ)

	var centerVecDense mat.VecDense
	centerVecDense.CloneFromVec(center)
	centerVec := centerVecDense.SliceVec(0, 3)

	for index, imageData := range encodedImages {
		imageName := imageData.name
		C := centers[imageName]
		var vector mat.VecDense
		fmt.Printf("Center = %v\n", sph.FormatMatrixPrint(centerVec))
		fmt.Printf("C = %v\n", sph.FormatMatrixPrint(C))

		vector.SubVec(C, centerVec)
		long, lat := photogrammetry.GetLongLat(vector)
		imageData.longitude = photogrammetry.Rad2Degrees(long)
		imageData.latitude = photogrammetry.Rad2Degrees(lat)
		encodedImages[index] = imageData
	}

	return cameraViewer{images: encodedImages}
}

/*
  to_jsonify = {}
  encoded_images = []
  centers = {}
  centers_x = []
  centers_y = []
  centers_z = []
  for image_name in calib_file["extrinsics"]:
    try:
      image_data = get_response_image(f"{directory}/{calib_file['thumbnails']}/{image_name}")
      image_data["name"] = image_name

      mat = np.matrix(calib_file["extrinsics"][image_name]["matrix"])
      rotation = mat[0:3, 0:3]
      trans = mat[0:3, 3]
      C = converters.get_camera_world_coordinates(rotation, trans)

      centers[image_name] = C
      centers_x.append(C.item(0)) # x
      centers_y.append(C.item(1)) # y
      centers_z.append(C.item(2)) # z

      encoded_images.append(image_data)
    except Exception as error:
       print(error)
       continue
  _, center = reconstruction.sphereFit(centers_x, centers_y, centers_z)

  for image_data in encoded_images:
    image_name = image_data["name"]
    C = centers[image_name]
    vec = C - center
    long, lat = converters.get_long_lat(vec)
    image_data["longitude"], image_data["latitude"] = converters.rad2degrees(long), converters.rad2degrees(lat)

  print(f"Sending {len(encoded_images)} images")
  to_jsonify["images"] = encoded_images
  return jsonify(to_jsonify)
*/
