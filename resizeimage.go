package resizeimage

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/nfnt/resize"
)

// ResizeImage reads image size and if exceeds max size (either height or width),
// resize image to fit into max size.
func ResizeImage(filePath string, maxSize float64, postfix string) error {
	if postfix == "" {
		return errors.New("please specify postfix")
	}
	base := filepath.Base(filePath)
	ext := filepath.Ext(filePath)
	ext = strings.ToLower(ext)

	imageFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer imageFile.Close()

	var decImage image.Image
	var gifImage *gif.GIF
	var imageConfig image.Config
	if ext == ".jpg" || ext == ".jpeg" {
		decImage, err = jpeg.Decode(imageFile)
		if err != nil {
			log.Println(err)
		}
		_, err = imageFile.Seek(io.SeekStart, 0)
		if err != nil {
			return err
		}
		imageConfig, err = jpeg.DecodeConfig(imageFile)
		if err != nil {
			return err
		}
	} else if ext == ".png" {
		decImage, err = png.Decode(imageFile)
		if err != nil {
			return err
		}
		_, err = imageFile.Seek(io.SeekStart, 0)
		if err != nil {
			return err
		}
		imageConfig, err = png.DecodeConfig(imageFile)
		if err != nil {
			return err
		}
	} else if ext == ".gif" {
		gifImage, err = gif.DecodeAll(imageFile)
		if err != nil {
			return err
		}
		imageConfig = gifImage.Config
		if err != nil {
			return err
		}
	} else {
		return nil
	}

	width := float64(imageConfig.Width)
	height := float64(imageConfig.Height)

	var ratio float64
	if width > height && width > maxSize {
		ratio = maxSize / width
	} else if height > maxSize {
		ratio = maxSize / height
	} else {
		ratio = 1
	}

	tmpFileName := base[0:len(base)-len(ext)] + postfix + ext
	tmpFile, err := os.Create(tmpFileName)
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	if ratio == 1 {
		_, err = imageFile.Seek(io.SeekStart, 0)
		if err != nil {
			return err
		}
		_, err := io.Copy(tmpFile, imageFile)
		if err != nil {
			log.Fatal(err)
			return err
		}
	} else {
		if ext == ".jpg" || ext == ".jpeg" {
			resized := resize.Resize(uint(math.Floor(width*ratio)), uint(math.Floor(height*ratio)),
				decImage, resize.Lanczos3)
			jpeg.Encode(tmpFile, resized, nil)
		} else if ext == ".png" {
			resized := resize.Resize(uint(math.Floor(width*ratio)), uint(math.Floor(height*ratio)),
				decImage, resize.Lanczos3)
			png.Encode(tmpFile, resized)
		} else if ext == ".gif" {
			for index, frame := range gifImage.Image {
				rect := frame.Bounds()
				tmpImage := frame.SubImage(rect)
				resizedImage := resize.Resize(uint(math.Floor(float64(rect.Dx())*ratio)),
					uint(math.Floor(float64(rect.Dy())*ratio)),
					tmpImage, resize.Lanczos3)
				// Add colors from original gif image
				var tmpPalette color.Palette
				for x := 1; x <= rect.Dx(); x++ {
					for y := 1; y <= rect.Dy(); y++ {
						if !Contains(tmpPalette, gifImage.Image[index].At(x, y)) {
							tmpPalette = append(tmpPalette, gifImage.Image[index].At(x, y))
						}
					}
				}

				// After first image, image may contains only difference
				// bounds may not start from at (0,0)
				resizedBounds := resizedImage.Bounds()
				if index >= 1 {
					marginX := int(math.Floor(float64(rect.Min.X) * ratio))
					marginY := int(math.Floor(float64(rect.Min.Y) * ratio))
					resizedBounds = image.Rect(marginX, marginY, resizedBounds.Dx()+marginX,
						resizedBounds.Dy()+marginY)
				}
				resizedPalette := image.NewPaletted(resizedBounds, tmpPalette)
				draw.Draw(resizedPalette, resizedBounds, resizedImage, image.ZP, draw.Src)
				gifImage.Image[index] = resizedPalette
			}
			// Set size to resized size
			gifImage.Config.Width = int(math.Floor(width * ratio))
			gifImage.Config.Height = int(math.Floor(height * ratio))
			gif.EncodeAll(tmpFile, gifImage)
		}
	}
	return nil
}

// Contains checks if color is already in the Palette or not.
func Contains(colorPalette color.Palette, c color.Color) bool {
	for _, tmpColor := range colorPalette {
		if tmpColor == c {
			return true
		}
	}
	return false
}
