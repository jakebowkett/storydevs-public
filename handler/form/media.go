package form

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/jakebowkett/go-jpegutil/jpegutil"
	"github.com/jakebowkett/go-pngutil/pngutil"
	sd "github.com/jakebowkett/storydevs"
)

func emptyPara(p sd.Paragraph) bool {
	if len(p.Span) != 1 {
		return false
	}
	rr := []rune(p.Span[0].Text)
	if len(rr) != 1 {
		return false
	}
	return rr[0] == 0x200b
}

func (v *validator) validateRichText(rv reflect.Value, tbl *sd.DbTable, ignore ignore) error {

	rt, ok := rv.Interface().(sd.RichText)
	if !ok {
		return fmt.Errorf("Failed sd.RichText type assertion at %q", v.src)
	}
	if !v.sf.Optional && len(rt) == 0 {
		return fmt.Errorf("No paragraphs for non-optional field %q", v.src)
	}

	tbl.Columns = append(tbl.Columns, "words")
	tbl.Values = append(tbl.Values, rt.Words())

	tc := v.config.Thread
	overall := 0
	sOverall := 0
	for pIdx, p := range rt {
		if !in(tc.ParagraphTypes, p.Kind) {
			return fmt.Errorf("Paragraph %d has invalid kind %q at %q", pIdx, p.Kind, v.src)
		}
		if len(p.Span) == 0 {
			return fmt.Errorf("Paragraph %d has no spans at %q", pIdx, v.src)
		}
		paraLen := 0
		sIdx := 0
		isEmpty := emptyPara(p)
		for _, span := range p.Span {
			var newTbl sd.DbTable
			newTbl.Name = v.name + "_span"
			newTbl.Columns = []string{"p", "span", "kind", "text"}
			newTbl.Values = []interface{}{pIdx, sOverall, p.Kind, span.Text}
			if span.Link.String != "" {
				newTbl.Columns = append(newTbl.Columns, "link")
				newTbl.Values = append(newTbl.Values, span.Link.String)
			}
			for _, f := range span.Format {
				if !in(tc.InlineStyles, f) {
					return fmt.Errorf("Format %q in p %d, span %d is invalid at %q",
						f, pIdx, sIdx, v.src)
				}
				newTbl.Columns = append(newTbl.Columns, f)
				newTbl.Values = append(newTbl.Values, true)
			}
			if span.Text == "" {
				return fmt.Errorf("Text in p %d, span %d is empty at %q", pIdx, sIdx, v.src)
			}
			if !isEmpty && ctrlChar.MatchString(span.Text) {
				return fmt.Errorf("Text in p %d, span %d contains control character at %q",
					pIdx, sIdx, v.src)
			}
			if ctrlChar.MatchString(span.Link.String) {
				return fmt.Errorf("Link in p %d, span %d contains control character at %q",
					pIdx, sIdx, v.src)
			}
			n := len([]rune(span.Text))
			paraLen += n
			overall += n
			sIdx++
			sOverall++
			tbl.Tables = append(tbl.Tables, &newTbl)
		}
		if paraLen > tc.MaxParagraph {
			return fmt.Errorf("Paragraph %d exceeds rune limit at %q", pIdx, v.src)
		}
	}
	if overall < v.sf.Min {
		return fmt.Errorf("Overall rune minimum not met at %q", v.src)
	}
	if v.sf.Max > 0 && overall > v.sf.Max {
		return fmt.Errorf("Overall rune maximum exceeded at %q", v.src)
	}

	return nil
}

func (v *validator) validateMedia(rv reflect.Value, tbl *sd.DbTable, ignore ignore) error {
	m, ok := rv.Interface().(sd.Media)
	if !ok {
		return fmt.Errorf("failed type assertion at %q", v.src)
	}
	switch {
	case len(m.RichText) > 0: // Handled by the later iteration over Media
		return nil
	case m.File.Name.String() != "":
		return v.validateMediaFileName(rv, tbl)
	case m.File.Data != nil:
		return v.validateMediaFile(rv, tbl)
	case !v.sf.Optional:
		return fmt.Errorf("Non optional Media field %q contains no media.", v.src)
	}
	return nil
}

/*
validateMediaFileName handles the case where a user has already
submitted a file and is updating their profile. This results
in a file name being submitted whose existence we verify.
*/

var fileFmts = strings.Join(sd.FileFormats, "|")
var validFileName = regexp.MustCompile(`^[a-zA-Z0-9]+\.(` + fileFmts + `)$`)

func (v *validator) validateMediaFileName(rv reflect.Value, tbl *sd.DbTable) error {

	media, ok := rv.Addr().Interface().(*sd.Media)
	if !ok {
		return fmt.Errorf("failed type assertion at %q", v.src)
	}
	if !validFileName.MatchString(media.File.Name.String()) {
		return fmt.Errorf("malformed media file name at %q", v.src)
	}

	switch strings.TrimPrefix(filepath.Ext(media.File.Name.String()), ".") {
	case sd.FormatJPEG:
		media.Kind = sd.MediaImage
		media.Format = sd.FormatJPEG
	case sd.FormatPNG:
		media.Kind = sd.MediaImage
		media.Format = sd.FormatPNG
	default:
		return fmt.Errorf("field %q is an unknown file format", v.src)
	}

	// Assign aspect ratio.
	path, err := filepath.Abs(filepath.Join(v.config.DirUser, media.File.Name.String()))
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}
	media.Aspect = aspect(img)

	// Retain this file.
	v.TableTree.Retain = append(v.TableTree.Retain, media.File.Name.String())

	v.pushPath("filename")
	v.valToTable(tbl, media.File.Name.String())
	v.popPath()

	return nil
}

func (v *validator) validateMediaFile(rv reflect.Value, tbl *sd.DbTable) error {

	media, ok := rv.Addr().Interface().(*sd.Media)
	if !ok {
		return fmt.Errorf("failed type assertion at %q", v.src)
	}

	/*
		Read a little bit of the file (magic bytes, headers) to
		see if it's an allowed format and structurally sound.
	*/
	kind, format, err := v.fileKind(media.File.Data)
	if err != nil {
		return err
	}

	f, err := v.sf.Find(kind)
	if err != nil {
		return err
	}
	if err := v.checkFileSize(media.File.Data, f.Max, kind); err != nil {
		return err
	}

	media.Kind = kind
	media.Format = format
	media.Artist = v.handle

	/*
		Commit associated files to disk and assign
		aspect ratio and FileName to media. Return
		paths of files written to disk.
	*/
	var paths []string
	switch format {
	case sd.FormatJPEG:
		paths, err = v.writeJPEG(media, nil)
	case sd.FormatPNG:
		paths, err = v.writePNG(media)
	}

	v.TableTree.Written = append(v.TableTree.Written, paths...)

	v.pushPath("filename")
	v.valToTable(tbl, media.File.Name.String())
	v.popPath()

	return nil
}

func (v *validator) validateFileName(fn string, tbl *sd.DbTable) error {

	if fn == "" {
		v.valToTable(tbl, nil)
		return nil
	}

	if !validFileName.MatchString(fn) {
		return fmt.Errorf("malformed file name at %q", v.src)
	}

	// Check file exists
	path, err := filepath.Abs(filepath.Join(v.config.DirUser, fn))
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}

	// Retain this file.
	v.TableTree.Retain = append(v.TableTree.Retain, fn)
	v.valToTable(tbl, fn)

	return nil
}

func (v *validator) validateFile(rv reflect.Value, tbl *sd.DbTable, ignore ignore) error {

	file, ok := rv.Addr().Interface().(*sd.File)
	if !ok {
		return fmt.Errorf("failed type assertion at %q", v.src)
	}

	if file.Data == nil {
		return v.validateFileName(file.Name.String(), tbl)
	}

	/*
		Read a little bit of the file (magic bytes, headers) to
		see if it's an allowed format and structurally sound.
	*/
	kind, format, err := v.fileKind(file.Data)
	if err != nil {
		return err
	}
	if err := v.checkFileSize(file.Data, v.sf.Max, kind); err != nil {
		return err
	}

	/*
		It is very important to decode images from file.Data before
		replacing their metadata. The latter operation alters the
		original reader (as documented in the packages). The decode
		functions expect the supplied readers to be seeked to the
		start but after replacing metadata file.Data will be seeked
		to beyond the magic bytes identifying the file format, causing
		the decode functions to return an error. The replace metadata
		functions on the other hand will seek to the correct position
		regardless.
	*/
	var r io.Reader
	var img image.Image
	var imgErr error
	switch format {
	case sd.FormatJPEG:
		img, imgErr = jpeg.Decode(file.Data)
		r, err = jpegutil.ReplaceMeta(file.Data, nil)
	case sd.FormatPNG:
		img, imgErr = png.Decode(file.Data)
		r, err = pngutil.ReplaceMeta(file.Data, nil)
	default:
		err = fmt.Errorf("Invalid file format: %q", format)
	}
	if err != nil {
		return err
	}
	if imgErr != nil {
		println(imgErr.Error())
		return err
	}

	// Create the thumbnail.
	rThumb, err := thumbFromImg(img, format)
	if err != nil {
		return err
	}

	var fn string
	var disk string
	var diskThumb string

	errs, err := v.retry.Try(func() (err error) {
		fn, disk, diskThumb, err = makeFilePath(v.config, format)
		if err != nil {
			return err
		}
		if err := writeIfNotExists(disk, r); err != nil {
			return err
		}
		if err := writeIfNotExists(diskThumb, rThumb); err != nil {
			return err
		}
		v.TableTree.Written = append(v.TableTree.Written, disk, diskThumb)
		v.valToTable(tbl, fn)
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
		return fmt.Errorf("Unable to save %s to disk.", strings.ToUpper(format))
	}

	return nil
}

func (v *validator) fileKind(file sd.ReadSeekCloser) (string, string, error) {

	var kind string
	var format string
	switch {
	case jpegutil.Assert(file) == nil:
		kind = sd.MediaImage
		format = sd.FormatJPEG
	case pngutil.Assert(file) == nil:
		kind = sd.MediaImage
		format = sd.FormatPNG
	default:
		err := fmt.Errorf("field %q contains unknown or malformed file format", v.src)
		return kind, format, err
	}

	// Check the kind of file is allowed.
	if !in(v.sf.ValueText(), kind) {
		err := fmt.Errorf("field %q contains disallowed file kind %q", v.src, kind)
		return kind, format, err
	}

	return kind, format, nil
}

func (v *validator) checkFileSize(file sd.ReadSeekCloser, max int, kind string) error {
	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	// Note: max is in kilobytes, not bytes.
	if maxBytes := int64(max) * 1024; size > maxBytes {
		msg := "field %q disallows files larger than %dKiB for kind %q, got %dKiB"
		return fmt.Errorf(msg, v.src, max, kind, size/1024)
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	return nil
}

func in(ss []string, s string) bool {
	for i := range ss {
		if ss[i] == s {
			return true
		}
	}
	return false
}
