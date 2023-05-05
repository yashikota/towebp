package main

import (
	"image"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/GoWebProd/uuid7"

	"github.com/adrium/goheif"
	"github.com/disintegration/imaging"
	"github.com/gabriel-vasile/mimetype"
	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
)

type writerSkipper struct {
	w           io.Writer
	bytesToSkip int
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: towebp <input image>")
	}

	u := uuid7.New()

	input := os.Args[1]
	output := u.Next().String() + ".webp"

	var img image.Image
	var err error

	switch contentType(input) {
	case "image/heic", "image/heif":
		img, err = decodeHeic(input)
	case "image/jpeg", "image/png":
		img, err = decode(input)
	default:
		log.Fatalf("Unsupported input format: %s, content-type: %s", filepath.Ext(input), contentType(input))
	}

	if err != nil {
		log.Fatal(err)
	}

	options, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, 75)
	if err != nil {
		log.Fatalln(err)
	}

	outputFile, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer outputFile.Close()

	if err := webp.Encode(outputFile, img, options); err != nil {
		log.Fatalln(err)
	}
}

func contentType(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	mime, err := mimetype.DetectReader(file)
	if err != nil {
		log.Fatal(err)
	}

	return mime.String()
}

func decodeHeic(input string) (image.Image, error) {
	fileInput, err := os.Open(input)
	if err != nil {
		return nil, err
	}
	defer fileInput.Close()

	exif, err := goheif.ExtractExif(fileInput)
	if err != nil {
		return nil, err
	}

	img, err := goheif.Decode(fileInput)
	if err != nil {
		return nil, err
	}

	fileJpg, err := os.CreateTemp("", "temp_*.jpg")
	if err != nil {
		return nil, err
	}
	defer os.Remove(fileJpg.Name())
	defer fileJpg.Close()

	w, _ := newWriterExif(fileJpg, exif)
	if err := imaging.Encode(w, img, imaging.JPEG); err != nil {
		return nil, err
	}

	return decode(fileJpg.Name())
}

func decode(input string) (image.Image, error) {
	fileInput, err := os.Open(input)
	if err != nil {
		return nil, err
	}
	defer fileInput.Close()

	return imaging.Decode(fileInput, imaging.AutoOrientation(true))
}

func (w *writerSkipper) Write(data []byte) (int, error) {
	if w.bytesToSkip <= 0 {
		return w.w.Write(data)
	}

	if dataLen := len(data); dataLen < w.bytesToSkip {
		w.bytesToSkip -= dataLen
		return dataLen, nil
	}

	if n, err := w.w.Write(data[w.bytesToSkip:]); err == nil {
		n += w.bytesToSkip
		w.bytesToSkip = 0
		return n, nil
	} else {
		return n, err
	}
}

func newWriterExif(w io.Writer, exif []byte) (io.Writer, error) {
	writer := &writerSkipper{w, 2}
	soi := []byte{0xff, 0xd8}
	if _, err := w.Write(soi); err != nil {
		return nil, err
	}

	if exif != nil {
		app1Marker := 0xe1
		markerlen := 2 + len(exif)
		marker := []byte{0xff, uint8(app1Marker), uint8(markerlen >> 8), uint8(markerlen & 0xff)}
		if _, err := w.Write(marker); err != nil {
			return nil, err
		}

		if _, err := w.Write(exif); err != nil {
			return nil, err
		}
	}

	return writer, nil
}
