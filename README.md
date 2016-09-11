# apng

This is a fork of code in Go's image/png package.

Package apng is used to do low-level APNG encoding.  The APNG format is not
widely supported in browsers, however, it can be an efficient way to represent
raster graphics in a lossless encoding, for instance to overlay over a video
with ffmpeg.

For encoding details, see:

* https://en.wikipedia.org/wiki/APNG#Technical_details
* https://wiki.mozilla.org/APNG_Specification
* https://www.w3.org/TR/PNG/
