package imports

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/h2non/bimg"
)

var ACCEPTABLE_IMAGES_EXT = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
}

const THUMBNAILS_SIZE = 1500

type SaveThumbnail struct {
	Path  string
	Image string
}

func ReadChildImages(dir string, thumbnails string) (map[string]string, int, int, []SaveThumbnail, error) {
	toRet := make(map[string]string)

	thumbnailsToCreate := []SaveThumbnail{}

	thumbWidth, thumbHeight := THUMBNAILS_SIZE, THUMBNAILS_SIZE

	entries, err := os.ReadDir(dir)

	if err != nil {
		return nil, 0, 0, nil, err
	}

	for _, v := range entries {
		if v.IsDir() {
			continue
		}
		_, ok := ACCEPTABLE_IMAGES_EXT[filepath.Ext(v.Name())]
		// If the key exists
		if ok {
			toRet[strings.TrimSuffix(filepath.Base(v.Name()), filepath.Ext(v.Name()))] = v.Name()
			thumbPath := fmt.Sprintf("%s/%s/%s", dir, thumbnails, v.Name())
			thumbExists, _ := exists(thumbPath)

			if thumbExists {
				buffer, err := bimg.Read(thumbPath)
				if err != nil {
					thumbnailsToCreate = append(thumbnailsToCreate, SaveThumbnail{Path: thumbPath, Image: fmt.Sprintf("%s/%s", dir, v.Name())})
				}
				newImage := bimg.NewImage(buffer)
				size, err := newImage.Size()
				if err != nil {
					thumbnailsToCreate = append(thumbnailsToCreate, SaveThumbnail{Path: thumbPath, Image: fmt.Sprintf("%s/%s", dir, v.Name())})
				}
				thumbWidth = size.Width
				thumbHeight = size.Height

			} else {
				thumbnailsToCreate = append(thumbnailsToCreate, SaveThumbnail{Path: thumbPath, Image: fmt.Sprintf("%s/%s", dir, v.Name())})
			}
		}
	}

	return toRet, thumbWidth, thumbHeight, thumbnailsToCreate, nil
}

func CreateThumbnails(thumbnailsDirPath string, thumbnails []SaveThumbnail, thumbWidth int, thumbHeight int) (int, int, error) {
	thumbExists, _ := exists(thumbnailsDirPath)
	if !thumbExists {
		fmt.Printf("ThumbnailsDir doesn't exist : %s\n", thumbnailsDirPath)
		os.Mkdir(thumbnailsDirPath, 0644)
	}
	for _, thumbnail := range thumbnails {
		buffer, err := bimg.Read(thumbnail.Image)
		if err != nil {
			return 0, 0, err
		}

		fullImage := bimg.NewImage(buffer)
		fullSize, err := fullImage.Size()
		if err != nil {
			return 0, 0, err
		}
		var resizedImage []byte

		if fullSize.Width > fullSize.Height {
			resizedImage, err = fullImage.Process(bimg.Options{Width: thumbWidth, Compression: 90})
		} else {
			resizedImage, err = fullImage.Process(bimg.Options{Height: thumbHeight, Compression: 90})
		}

		if err != nil || resizedImage == nil {
			return 0, 0, err
		}

		thumbnailSize, err := bimg.NewImage(resizedImage).Size()
		if err != nil {
			return 0, 0, err
		}
		thumbWidth = thumbnailSize.Width
		thumbHeight = thumbnailSize.Height

		bimg.Write(thumbnail.Path, resizedImage)

	}
	return thumbWidth, thumbHeight, nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}
