// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package png

import (
	"bufio"
	"compress/zlib"
	"hash/crc32"
	"image"
	"image/color"
	"io"
)

// CompressionLevel tells the encoding algorithm how to trade compression speed
// for image size.
type CompressionLevel int

func (l CompressionLevel) zlib() int {
	switch l {
	case DefaultCompression:
		return zlib.DefaultCompression
	case NoCompression:
		return zlib.NoCompression
	case BestSpeed:
		return zlib.BestSpeed
	case BestCompression:
		return zlib.BestCompression
	default:
		return zlib.DefaultCompression
	}
}

const (
	DefaultCompression CompressionLevel = 0
	NoCompression      CompressionLevel = -1
	BestSpeed          CompressionLevel = -2
	BestCompression    CompressionLevel = -3

	// Positive CompressionLevel values are reserved to mean a numeric zlib
	// compression level, although that is not implemented yet.
)

// ColorType is the type of color of the image, per the PNG spec.
type ColorType uint8

const sizeOfColorType = 1

const (
	ColorType_Grayscale      = ColorType(0)
	ColorType_TrueColor      = ColorType(2)
	ColorType_Paletted       = ColorType(3)
	ColorType_GrayscaleAlpha = ColorType(4)
	ColorType_TrueColorAlpha = ColorType(6)
)

// BitDepth is the bit depth of the image, as per the PNG spec.
type BitDepth uint8

const sizeOfBitDepth = 1

const (
	BitDepth_1  = BitDepth(1)
	BitDepth_2  = BitDepth(2)
	BitDepth_4  = BitDepth(4)
	BitDepth_8  = BitDepth(8)
	BitDepth_16 = BitDepth(16)
)

// CompressionMethod is the compression method, as per the PNG spec.
type CompressionMethod uint8

const sizeOfCompressionMethod = 1

const (
	CompressionMethod_Default = CompressionMethod(0)
)

// CompressionMethod is the filter method, as per the PNG spec.
type FilterMethod uint8

const sizeOfFilterMethod = 1

const (
	FilterMethod_Default = FilterMethod(0)
)

// CompressionMethod is the interlace method, as per the PNG spec.
type InterlaceMethod uint8

const sizeOfInterlaceMethod = 1

const (
	InterlaceMethd_NonInterlaced = InterlaceMethod(0)
	InterlaceMethd_Interlaced    = InterlaceMethod(1)
)

// Chunk_IHDR is the image header chunk, as per the PNG spec.
type Chunk_IHDR struct {
	Width             uint32
	Height            uint32
	BitDepth          BitDepth
	ColorType         ColorType
	CompressionMethod CompressionMethod
	FilterMethod      FilterMethod
	InterlaceMethod   InterlaceMethod
}

// A cb is a combination of color type and bit depth.
const (
	cbInvalid = iota
	cbG1
	cbG2
	cbG4
	cbG8
	cbGA8
	cbTC8
	cbP1
	cbP2
	cbP4
	cbP8
	cbTCA8
	cbG16
	cbGA16
	cbTC16
	cbTCA16
)

func (c *Chunk_IHDR) cb() int {
	switch true {
	case c.ColorType == ColorType_Grayscale && c.BitDepth == BitDepth_8:
		return cbG8
	case c.ColorType == ColorType_TrueColor && c.BitDepth == BitDepth_8:
		return cbTC8
	case c.ColorType == ColorType_Paletted && c.BitDepth == BitDepth_8:
		return cbP8
	case c.ColorType == ColorType_TrueColorAlpha && c.BitDepth == BitDepth_8:
		return cbTCA8
	case c.ColorType == ColorType_TrueColor && c.BitDepth == BitDepth_16:
		return cbTC16
	case c.ColorType == ColorType_TrueColorAlpha && c.BitDepth == BitDepth_16:
		return cbTCA16
	case c.ColorType == ColorType_Grayscale && c.BitDepth == BitDepth_16:
		return cbG16
	}
	return cbInvalid
}

// WriteTo encodes the IHDR chunk to the io.Writer.  This supports the
// io.WriterTo interface.
func (c *Chunk_IHDR) WriteTo(w io.Writer) (int64, error) {
	buf := [sizeOfUint32*2 + sizeOfBitDepth + sizeOfColorType + sizeOfCompressionMethod + sizeOfFilterMethod + sizeOfInterlaceMethod]byte{}
	writeUint32(buf[0:4], c.Width)
	writeUint32(buf[4:8], c.Height)
	buf[8] = byte(c.BitDepth)
	buf[9] = byte(c.ColorType)
	buf[10] = byte(c.CompressionMethod)
	buf[11] = byte(c.FilterMethod)
	buf[12] = byte(c.InterlaceMethod)
	return writeChunkTo("IHDR", buf[0:len(buf)], w)
}

// Chunk_PLTE is the palette chunk, as per the PNG spec.  Write this after IHDR
// but before TRNS or any image data.
type Chunk_PLTE struct {
	data []byte
}

// NewChunk_PLTE makes a new palette chunk from a color.Palette.
func NewChunk_PLTE(p color.Palette) {
	chunk := &Chunk_PLTE{
		data: make([]byte, 3*len(p)),
	}
	for i, c := range p {
		c1 := color.NRGBAModel.Convert(c).(color.NRGBA)
		chunk.data[3*i+0] = c1.R
		chunk.data[3*i+1] = c1.G
		chunk.data[3*i+2] = c1.B
	}
}

// WriteTo encodes the palette chunk to the io.Writer.  This supports the
// io.WriterTo interface.
func (c *Chunk_PLTE) WriteTo(w io.Writer) (int64, error) {
	return writeChunkTo("PLTE", c.data, w)
}

// Chunk_tRNS is the transparency chunk, as per the PNG spec.  Write this after
// IHDR and PLTE but before any image data.
type Chunk_tRNS struct {
	data []byte
}

// NewChunk_tRNS makes a new transparency chunk from a color.Palette.
func NewChunk_tRNS(p color.Palette) {
	chunk := &Chunk_tRNS{
		data: make([]byte, len(p)),
	}
	for i, c := range p {
		c1 := color.NRGBAModel.Convert(c).(color.NRGBA)
		chunk.data[i] = c1.A
	}
}

// WriteTo encodes the transparency chunk to the io.Writer.  This supports the
// io.WriterTo interface.
func (c *Chunk_tRNS) WriteTo(w io.Writer) (int64, error) {
	return writeChunkTo("tRNS", c.data, w)
}

// Chunk_IEND is the ending chunk, as per the PNG spec.  Write this after all other chunks.
type Chunk_IEND struct{}

// WriteTo encodes the ending chunk to the io.Writer.  This supports the
// io.WriterTo interface.
func (c *Chunk_IEND) WriteTo(w io.Writer) (int64, error) {
	return writeChunkTo("IEND", nil, w)
}

// Chunk_acTL is the animation control chunk, as per the APNG spec.  Write this
// before any image data.
type Chunk_acTL struct {
	NumFrames uint32 // Number of frames
	NumPlays  uint32 // Number of times to loop this APNG. 0 indicates infinite looping.
}

// WriteTo encodes the animation control chunk to the io.Writer.  This supports
// the io.WriterTo interface.
func (c *Chunk_acTL) WriteTo(w io.Writer) (int64, error) {
	buf := [sizeOfUint32 * 2]byte{}
	writeUint32(buf[0:4], c.NumFrames)
	writeUint32(buf[4:8], c.NumPlays)
	return writeChunkTo("acTL", buf[0:len(buf)], w)
}

// DisposeOp is the dispose operator, as per the APNG spec.
type DisposeOp uint8

const sizeOfDisposeOp = 1

const (
	DisposeOp_None       = DisposeOp(0)
	DisposeOp_Background = DisposeOp(1)
	DisposeOp_Previous   = DisposeOp(2)
)

// BlendOp is the blend operator, as per the APNG spec.
type BlendOp uint8

const sizeOfBlendOp = 1

const (
	BlendOp_Source = BlendOp(0)
	BlendOp_Over   = BlendOp(1)
)

// Chunk_fcTL is the frame control chunk, as per the APNG spec.
type Chunk_fcTL struct {
	SequenceNumber uint32    // Sequence number of the animation chunk, starting from 0
	Width          uint32    // Width of the following frame
	Height         uint32    // Height of the following frame
	XOffset        uint32    // X position at which to render the following frame
	YOffset        uint32    // Y position at which to render the following frame
	DelayNum       uint16    // Frame delay fraction numerator
	DelayDen       uint16    // Frame delay fraction denominator
	DisposeOp      DisposeOp // Type of frame area disposal to be done after rendering this frame
	BlendOp        BlendOp   // Type of frame area rendering for this frame
}

// WriteTo encodes the frame control chunk to the io.Writer.  This supports the
// io.WriterTo interface.
func (c *Chunk_fcTL) WriteTo(w io.Writer) (int64, error) {
	buf := [sizeOfUint32*5 + sizeOfUint16*2 + sizeOfDisposeOp + sizeOfBlendOp]byte{}
	writeUint32(buf[0:4], c.SequenceNumber)
	writeUint32(buf[4:8], c.Width)
	writeUint32(buf[8:12], c.Height)
	writeUint32(buf[12:16], c.XOffset)
	writeUint32(buf[16:20], c.YOffset)
	writeUint16(buf[20:22], c.DelayNum)
	writeUint16(buf[22:24], c.DelayDen)
	buf[24] = byte(c.DisposeOp)
	buf[25] = byte(c.BlendOp)
	return writeChunkTo("fcTL", buf[0:len(buf)], w)
}

// SequenceNumbers is used to track sequence numbers across all frames and
// chunks; use this with Chunk_fcTL and Encoder_fdAT.
type SequenceNumbers uint32

func NewSequenceNumbers() *SequenceNumbers {
	return new(SequenceNumbers)
}

func (s *SequenceNumbers) Next() uint32 {
	tmp := uint32(*s)
	*s++
	return tmp
}

// Chunk_IDAT is one image data chunk, as per the PNG spec.
type Chunk_IDAT []byte

// WriteTo encodes the image data chunk to the io.Writer.  This supports the
// io.WriterTo interface.
func (c Chunk_IDAT) WriteTo(w io.Writer) (int64, error) {
	return writeChunkTo("IDAT", c, w)
}

type atom struct {
	buf []byte
	err error
}

type atomWriter chan *atom

func (aw atomWriter) Write(b []byte) (int, error) {
	aw <- &atom{buf: b}
	return len(b), nil
}

// Encoder_IDAT is used to encode an image into one or more image data chunks.
type Encoder_IDAT struct {
	aw atomWriter
	a  *atom
}

// NewEncoder_IDAT makes a new image data encoder for the given image and compression level.
func (c *Chunk_IHDR) NewEncoder_IDAT(m image.Image, cl CompressionLevel) *Encoder_IDAT {
	aw := make(atomWriter)
	go func() {
		z, err := zlib.NewWriterLevel(bufio.NewWriterSize(aw, 1<<15), cl.zlib())
		if err != nil {
			aw <- &atom{err: err}
			return
		}
		defer close(aw)
		if err := writeImage(z, m, c.cb(), cl != NoCompression); err != nil {
			aw <- &atom{err: err}
			return
		}
	}()
	return &Encoder_IDAT{aw: aw}
}

// Next is used to advance the encoder to the next chunk.  Call this before
// using either Chunk or Err.
func (e *Encoder_IDAT) Next() bool {
	var ok bool
	if e.Err() != nil {
		return false
	}
	e.a, ok = <-e.aw
	return ok
}

// Err returns any errors encountered while encoding image data chunks.
func (e *Encoder_IDAT) Err() error {
	if e.a != nil && e.a.err != nil {
		return e.a.err
	}
	return nil
}

// Chunk returns the current image data chunk.
func (e *Encoder_IDAT) Chunk() Chunk_IDAT {
	return Chunk_IDAT(e.a.buf)
}

// Chunk_fdAT is the frame data chunk, as per the APNG spec.
type Chunk_fdAT struct {
	SequenceNumber uint32
	Chunk_IDAT     Chunk_IDAT
}

// WriteTo encodes the frame data chunk to the io.Writer.  This supports the
// io.WriterTo interface.
func (c *Chunk_fdAT) WriteTo(w io.Writer) (int64, error) {
	buf := make([]byte, 4+len(c.Chunk_IDAT))
	writeUint32(buf[0:4], c.SequenceNumber)
	// TODO: avoid this copy?
	copy(buf[4:len(buf)], c.Chunk_IDAT)
	return writeChunkTo("fdAT", buf, w)
}

type Encoder_fdAT struct {
	seq          *SequenceNumbers
	encoder_IDAT *Encoder_IDAT
}

// NewEncoder_fdAT makes a new frame data encoder for the given sequence
// numbers, image, and compression level.
func (c *Chunk_IHDR) NewEncoder_fdAT(seq *SequenceNumbers, m image.Image, cl CompressionLevel) *Encoder_fdAT {
	return &Encoder_fdAT{
		seq:          seq,
		encoder_IDAT: c.NewEncoder_IDAT(m, cl),
	}
}

// Next is used to advance the encoder to the next chunk.  Call this before
// using either Chunk or Err.
func (e *Encoder_fdAT) Next() bool {
	return e.encoder_IDAT.Next()
}

// Err returns any errors encountered while encoding image data chunks.
func (e *Encoder_fdAT) Err() error {
	return e.encoder_IDAT.Err()
}

// Chunk returns the current image data chunk.
func (e *Encoder_fdAT) Chunk() *Chunk_fdAT {
	return &Chunk_fdAT{
		SequenceNumber: e.seq.Next(),
		Chunk_IDAT:     e.encoder_IDAT.Chunk(),
	}
}

// Big-endian.
func writeUint16(b []uint8, u uint16) {
	b[0] = uint8(u >> 8)
	b[1] = uint8(u >> 0)
}

const sizeOfUint16 = 2

// Big-endian.
func writeUint32(b []uint8, u uint32) {
	b[0] = uint8(u >> 24)
	b[1] = uint8(u >> 16)
	b[2] = uint8(u >> 8)
	b[3] = uint8(u >> 0)
}

const sizeOfUint32 = 4

func writeChunkTo(name string, b []byte, w io.Writer) (int64, error) {
	header := [8]byte{}
	footer := [4]byte{}

	writeUint32(header[:4], uint32(len(b)))
	header[4] = name[0]
	header[5] = name[1]
	header[6] = name[2]
	header[7] = name[3]

	crc := crc32.NewIEEE()
	crc.Write(header[4:8])
	crc.Write(b)
	writeUint32(footer[:4], crc.Sum32())

	hl, err := w.Write(header[:8])
	if err != nil {
		return int64(hl), err
	}
	bl, err := w.Write(b)
	if err != nil {
		return int64(hl + bl), err
	}
	fl, err := w.Write(footer[:4])
	return int64(hl + bl + fl), err
}

func writeImage(w io.Writer, m image.Image, cb int, applyFilter bool) error {
	bpp := 0 // Bytes per pixel.

	switch cb {
	case cbG8:
		bpp = 1
	case cbTC8:
		bpp = 3
	case cbP8:
		bpp = 1
	case cbTCA8:
		bpp = 4
	case cbTC16:
		bpp = 6
	case cbTCA16:
		bpp = 8
	case cbG16:
		bpp = 2
	}
	// cr[*] and pr are the bytes for the current and previous row.
	// cr[0] is unfiltered (or equivalently, filtered with the ftNone filter).
	// cr[ft], for non-zero filter types ft, are buffers for transforming cr[0] under the
	// other PNG filter types. These buffers are allocated once and re-used for each row.
	// The +1 is for the per-row filter type, which is at cr[*][0].
	b := m.Bounds()
	var cr [nFilter][]uint8
	for i := range cr {
		cr[i] = make([]uint8, 1+bpp*b.Dx())
		cr[i][0] = uint8(i)
	}
	pr := make([]uint8, 1+bpp*b.Dx())

	gray, _ := m.(*image.Gray)
	rgba, _ := m.(*image.RGBA)
	paletted, _ := m.(*image.Paletted)
	nrgba, _ := m.(*image.NRGBA)

	for y := b.Min.Y; y < b.Max.Y; y++ {
		// Convert from colors to bytes.
		i := 1
		switch cb {
		case cbG8:
			if gray != nil {
				offset := (y - b.Min.Y) * gray.Stride
				copy(cr[0][1:], gray.Pix[offset:offset+b.Dx()])
			} else {
				for x := b.Min.X; x < b.Max.X; x++ {
					c := color.GrayModel.Convert(m.At(x, y)).(color.Gray)
					cr[0][i] = c.Y
					i++
				}
			}
		case cbTC8:
			// We have previously verified that the alpha value is fully opaque.
			cr0 := cr[0]
			stride, pix := 0, []byte(nil)
			if rgba != nil {
				stride, pix = rgba.Stride, rgba.Pix
			} else if nrgba != nil {
				stride, pix = nrgba.Stride, nrgba.Pix
			}
			if stride != 0 {
				j0 := (y - b.Min.Y) * stride
				j1 := j0 + b.Dx()*4
				for j := j0; j < j1; j += 4 {
					cr0[i+0] = pix[j+0]
					cr0[i+1] = pix[j+1]
					cr0[i+2] = pix[j+2]
					i += 3
				}
			} else {
				for x := b.Min.X; x < b.Max.X; x++ {
					r, g, b, _ := m.At(x, y).RGBA()
					cr0[i+0] = uint8(r >> 8)
					cr0[i+1] = uint8(g >> 8)
					cr0[i+2] = uint8(b >> 8)
					i += 3
				}
			}
		case cbP8:
			if paletted != nil {
				offset := (y - b.Min.Y) * paletted.Stride
				copy(cr[0][1:], paletted.Pix[offset:offset+b.Dx()])
			} else {
				pi := m.(image.PalettedImage)
				for x := b.Min.X; x < b.Max.X; x++ {
					cr[0][i] = pi.ColorIndexAt(x, y)
					i += 1
				}
			}
		case cbTCA8:
			if nrgba != nil {
				offset := (y - b.Min.Y) * nrgba.Stride
				copy(cr[0][1:], nrgba.Pix[offset:offset+b.Dx()*4])
			} else {
				// Convert from image.Image (which is alpha-premultiplied) to PNG's non-alpha-premultiplied.
				for x := b.Min.X; x < b.Max.X; x++ {
					c := color.NRGBAModel.Convert(m.At(x, y)).(color.NRGBA)
					cr[0][i+0] = c.R
					cr[0][i+1] = c.G
					cr[0][i+2] = c.B
					cr[0][i+3] = c.A
					i += 4
				}
			}
		case cbG16:
			for x := b.Min.X; x < b.Max.X; x++ {
				c := color.Gray16Model.Convert(m.At(x, y)).(color.Gray16)
				cr[0][i+0] = uint8(c.Y >> 8)
				cr[0][i+1] = uint8(c.Y)
				i += 2
			}
		case cbTC16:
			// We have previously verified that the alpha value is fully opaque.
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, b, _ := m.At(x, y).RGBA()
				cr[0][i+0] = uint8(r >> 8)
				cr[0][i+1] = uint8(r)
				cr[0][i+2] = uint8(g >> 8)
				cr[0][i+3] = uint8(g)
				cr[0][i+4] = uint8(b >> 8)
				cr[0][i+5] = uint8(b)
				i += 6
			}
		case cbTCA16:
			// Convert from image.Image (which is alpha-premultiplied) to PNG's non-alpha-premultiplied.
			for x := b.Min.X; x < b.Max.X; x++ {
				c := color.NRGBA64Model.Convert(m.At(x, y)).(color.NRGBA64)
				cr[0][i+0] = uint8(c.R >> 8)
				cr[0][i+1] = uint8(c.R)
				cr[0][i+2] = uint8(c.G >> 8)
				cr[0][i+3] = uint8(c.G)
				cr[0][i+4] = uint8(c.B >> 8)
				cr[0][i+5] = uint8(c.B)
				cr[0][i+6] = uint8(c.A >> 8)
				cr[0][i+7] = uint8(c.A)
				i += 8
			}
		}

		// Apply the filter.
		f := ftNone
		if applyFilter {
			f = filter(&cr, pr, bpp)
		}

		// Write the compressed bytes.
		if _, err := w.Write(cr[f]); err != nil {
			return err
		}

		// The current row for y is the previous row for y+1.
		pr, cr[0] = cr[0], pr
	}
	return nil
}
