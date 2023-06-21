package form

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jakebowkett/go-gen/gen"
	"github.com/jakebowkett/go-jpegutil/jpegutil"
	"github.com/jakebowkett/go-pngutil/pngutil"
	sd "github.com/jakebowkett/storydevs"
	"golang.org/x/image/draw"
)

type decoded struct {
	img image.Image
	rs  io.ReadSeeker
}

func (v *validator) writeJPEG(media *sd.Media, dec *decoded) (paths []string, err error) {

	/*
		If dec is not supplied we've got a JPEG. Otherwise
		media.File is a PNG which must be replaced.
	*/
	var img image.Image
	var rs io.ReadSeeker
	if dec == nil {
		if img, err = jpeg.Decode(media.File.Data); err != nil {
			return nil, err
		}
		rs = media.File.Data
	} else {
		img = dec.img
		rs = dec.rs
	}

	// Create the thumbnail.
	rsThumb, err := thumbFromImg(img, sd.FormatJPEG)
	if err != nil {
		return nil, err
	}

	// Replace/add metadata.
	md := make(jpegutil.Metadata)
	md[jpegutil.MetaTitle] = fmt.Sprintf("%s by %s on StoryDevs", media.Title, media.Artist)
	md[jpegutil.MetaArtist] = media.Info
	md[jpegutil.MetaCopyright] = fmt.Sprintf("%s %d", media.Artist, time.Now().Year())
	r, err := jpegutil.ReplaceMeta(rs, md)
	if err != nil {
		return nil, err
	}
	rThumb, err := jpegutil.ReplaceMeta(rsThumb, md)
	if err != nil {
		return nil, err
	}

	/*
		Write image to disk. If the write doesn't
		occur because a file with that name already
		exists then we retry.
	*/
	var fn string
	var disk string
	var diskThumb string

	errs, err := v.retry.Try(func() (err error) {
		fn, disk, diskThumb, err = makeFilePath(v.config, sd.FormatJPEG)
		if err != nil {
			return err
		}
		if err := writeIfNotExists(disk, r); err != nil {
			return err
		}
		if err := writeIfNotExists(diskThumb, rThumb); err != nil {
			return err
		}
		media.File.Name.Set(fn)
		media.Aspect = aspect(img)
		return nil
	})

	log := v.log

	if err != nil {
		log.ErrorMulti(v.reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsDisk, len(errs)).
			Data(sd.LK_FileName, fn).
			Data(sd.LK_Mode, v.mode).
			Data(sd.LK_ResourceSlug, v.rSlug).
			Data(sd.LK_PersSlug, v.pSlug).
			Data(sd.LK_PersHandle, v.handle)
		return nil, errors.New("Unable to save JPEG to disk.")
	}

	return []string{disk, diskThumb}, nil
}

func (v *validator) writePNG(media *sd.Media) (paths []string, err error) {

	// Get size of file as PNG.
	size, err := media.File.Data.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	if _, err := media.File.Data.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	/*
		Re-encode it as a decent quality JPEG. If
		it's smaller, save it as a JPEG. (And change
		its extension).
	*/
	img, err := png.Decode(media.File.Data)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, img, &jpeg.Options{Quality: 100})
	if err != nil {
		return nil, err
	}
	if int64(buf.Len()) < size {
		return v.writeJPEG(media, &decoded{img, bytes.NewReader(buf.Bytes())})
	}

	// Create the thumbnail.
	rsThumb, err := thumbFromImg(img, sd.FormatPNG)
	if err != nil {
		return nil, err
	}

	// Replace/add metadata.
	md := make(pngutil.Metadata)
	md[pngutil.MetaTitle] = fmt.Sprintf("%s by %s on StoryDevs", media.Title, media.Artist)
	md[pngutil.MetaAuthor] = media.Info
	md[pngutil.MetaCopyright] = fmt.Sprintf("%s %d", media.Artist, time.Now().Year())
	r, err := pngutil.ReplaceMeta(media.File.Data, md)
	if err != nil {
		return nil, err
	}
	rThumb, err := pngutil.ReplaceMeta(rsThumb, md)
	if err != nil {
		return nil, err
	}

	/*
		Write image to disk. If the write doesn't
		occur because a file with that name already
		exists then we retry.
	*/
	var fn string
	var disk string
	var diskThumb string

	errs, err := v.retry.Try(func() (err error) {
		fn, disk, diskThumb, err = makeFilePath(v.config, sd.FormatPNG)
		if err != nil {
			return err
		}
		if err := writeIfNotExists(disk, r); err != nil {
			return err
		}
		if err := writeIfNotExists(diskThumb, rThumb); err != nil {
			return err
		}
		media.File.Name.Set(fn)
		media.Aspect = aspect(img)
		return nil
	})

	log := v.log

	if err != nil {
		log.ErrorMulti(v.reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsDisk, len(errs)).
			Data(sd.LK_FileName, fn).
			Data(sd.LK_Mode, v.mode).
			Data(sd.LK_ResourceSlug, v.rSlug).
			Data(sd.LK_PersSlug, v.pSlug).
			Data(sd.LK_PersHandle, v.handle)
		return nil, errors.New("Unable to save PNG to disk.")
	}

	return []string{disk, diskThumb}, nil
}

/*
The suffix is inserted between the generated file name and its extension.
*/
func makeFilePath(c *sd.Config, format string) (fn, disk, diskThumb string, err error) {
	slug, _ := gen.AlphaNum(c.SlugLen)
	fn = slug + "." + format
	fnThumb := slug + "_thumb." + format
	disk, err = filepath.Abs(filepath.Join(c.DirUser, fn))
	if err != nil {
		return "", "", "", err
	}
	diskThumb, err = filepath.Abs(filepath.Join(c.DirUser, fnThumb))
	if err != nil {
		return "", "", "", err
	}
	return fn, disk, diskThumb, nil
}

/*
writeIfNotExists attempts to write r to path. If a file
already exists at path it will not be overwritten and ok
will be false.

writeIfNotExists will create all the directories in path
if they do not exist.
*/
func writeIfNotExists(path string, r io.Reader) (err error) {
	dir := strings.TrimSuffix(path, filepath.Base(path))
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	t := io.TeeReader(r, f)
	p := make([]byte, 64, 64)
	for {
		_, err = t.Read(p)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
	}
	return nil
}

func thumbFromImg(img image.Image, format string) (rs io.ReadSeeker, err error) {
	imgThumb := scaleImage(img, 320)
	var bb bytes.Buffer
	switch format {
	case sd.FormatJPEG:
		err = jpeg.Encode(&bb, imgThumb, nil)
	case sd.FormatPNG:
		err = png.Encode(&bb, imgThumb)
	}
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(bb.Bytes()), nil
}

func scaleImage(img image.Image, max int) image.Image {
	if max > longest(img) {
		return img
	}
	x, y := xy(img)
	aspect := float64(x) / float64(y)
	if x > y {
		x = max
		y = int(math.Round(float64(max) / aspect))
	} else {
		y = max
		x = int(math.Round(float64(max) * aspect))
	}
	scaledRect := image.Rect(0, 0, x, y)
	scaled := image.NewRGBA(scaledRect)
	draw.CatmullRom.Scale(scaled, scaledRect, img, img.Bounds(), draw.Over, nil)
	return scaled
}

func aspect(img image.Image) float64 {
	x, y := xy(img)
	return float64(x) / float64(y)
}

func xy(img image.Image) (x int, y int) {
	rect := img.Bounds()
	return rect.Max.X, rect.Max.Y
}

func longest(img image.Image) int {
	x, y := xy(img)
	if x > y {
		return x
	}
	return y
}
