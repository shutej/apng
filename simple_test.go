package apng_test

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io/ioutil"

	"github.com/shutej/apng"
)

const frames = 10

func Example() {
	b := image.Rect(0, 0, 100, 100)

	buf := bytes.NewBuffer(nil)

	buf.WriteString(apng.PngHeader)

	ihdr := &apng.Chunk_IHDR{
		Width:     uint32(b.Max.X),
		Height:    uint32(b.Max.Y),
		BitDepth:  apng.BitDepth_8,
		ColorType: apng.ColorType_TrueColorAlpha,
	}
	ihdr.WriteTo(buf)

	actl := &apng.Chunk_acTL{
		NumFrames: frames,
		NumPlays:  1,
	}
	actl.WriteTo(buf)

	// Writes the default image for the PNG.  Not part of the animation in this case...
	m := image.NewNRGBA(b)

	seq := apng.NewSequenceNumbers()

	cl := apng.DefaultCompression
	e := ihdr.NewEncoder_IDAT(m, cl)
	n := 0
	for e.Next() {
		n++
		fmt.Printf("IDAT (chunk %d)\n", n)
		e.Chunk().WriteTo(buf)
	}
	if err := e.Err(); err != nil {
		panic(err)
	}

	// Writes the animation frames...
	for i := 0; i < frames; i++ {
		m.Set(i*b.Max.X/frames, i*b.Max.Y/frames, color.NRGBA{R: 255, A: 255})
		fctl := &apng.Chunk_fcTL{
			SequenceNumber: seq.Next(),
			Width:          uint32(b.Max.X),
			Height:         uint32(b.Max.Y),
			DelayNum:       100, // 10 fps
			DelayDen:       1000,
		}
		fctl.WriteTo(buf)

		n = 0
		e = ihdr.NewEncoder_fdAT(seq, m, cl)
		for e.Next() {
			n++
			fmt.Printf("fdAT (frame %d, chunk %d)\n", i, n)
			e.Chunk().WriteTo(buf)
		}
		if err := e.Err(); err != nil {
			panic(err)
		}
	}

	iend := &apng.Chunk_IEND{}
	iend.WriteTo(buf)

	if err := ioutil.WriteFile("example.png", buf.Bytes(), 0600); err != nil {
		panic(err)
	}

	// Output:
	// IDAT (chunk 1)
	// fdAT (frame 0, chunk 1)
	// fdAT (frame 1, chunk 1)
	// fdAT (frame 2, chunk 1)
	// fdAT (frame 3, chunk 1)
	// fdAT (frame 4, chunk 1)
	// fdAT (frame 5, chunk 1)
	// fdAT (frame 6, chunk 1)
	// fdAT (frame 7, chunk 1)
	// fdAT (frame 8, chunk 1)
	// fdAT (frame 9, chunk 1)
}
