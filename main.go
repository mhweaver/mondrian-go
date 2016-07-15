package main

import (
	"errors"
	"github.com/disintegration/gift"
	farbfeld "github.com/hullerob/go.farbfeld"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math/rand"
	"os"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	var src image.Image
	src, err := farbfeld.Decode(os.Stdin)
	if err != nil {
		log.Panic(err)
	}
	size := src.Bounds().Size()
	width := size.X
	height := size.Y
	g := gift.New(
		gift.Resize(width/4, height/4, gift.NearestNeighborResampling),
		gift.GaussianBlur(2),
		gift.Resize(width, height, gift.LinearResampling),
	)
	blurred := image.NewRGBA(g.Bounds(src.Bounds()))
	g.Draw(blurred, src)
	dst := mondrian(blurred)
	farbfeld.Encode(os.Stdout, dst)
}

const (
	RED = iota
	BLUE
	YELLOW
	COPY
	WHITE
)

func mondrian(img image.Image) image.Image {
	// Get a list of non-overlapping rectangles
	rectangles := getRectangles(img.Bounds())

	dst := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))

	imageMap := make(map[int]image.Image)
	imageMap[RED] = &image.Uniform{color.RGBA{255, 0, 0, 255}}
	imageMap[BLUE] = &image.Uniform{color.RGBA{0, 0, 255, 255}}
	imageMap[YELLOW] = &image.Uniform{color.RGBA{255, 255, 0, 255}}
	imageMap[WHITE] = &image.Uniform{color.RGBA{255, 255, 255, 255}}
	imageMap[COPY] = img

	borderColor := &image.Uniform{color.RGBA{0, 0, 0, 255}} // Black
	draw.Draw(dst, dst.Bounds(), borderColor, image.ZP, draw.Src)

	for _, r := range rectangles {
		var imageToCopy image.Image

		isCopyOrWhite := rand.Intn(4) != 0
		if isCopyOrWhite {
			if rand.Intn(4) == 0 {
				imageToCopy = imageMap[COPY]
			} else {
				imageToCopy = imageMap[WHITE]
			}
		} else {
			imageToCopy = imageMap[rand.Intn(WHITE+1)]
		}
		inset := r.Inset(2) // Inset the rectangle by 2 pixels in order to let the border color show
		draw.Draw(dst, inset, imageToCopy, inset.Min, draw.Src)
	}

	return dst
}

func getRectangles(initialRectangle image.Rectangle) []image.Rectangle {
	leftRect := image.Rect(initialRectangle.Min.X, initialRectangle.Min.Y, initialRectangle.Max.X/2, initialRectangle.Max.Y)
	rightRect := image.Rect(initialRectangle.Max.X/2, initialRectangle.Min.Y, initialRectangle.Max.X, initialRectangle.Max.Y)
	rectangles := []image.Rectangle{initialRectangle}
	numSplits := 15 + rand.Intn(30)
	padding := 100 // The minimum amount of internal padding to allow when splitting rectangles
	// Note: This padding can still be violated when intersections are found

	for i := 0; i < numSplits; i++ {
		// Grab a random rectangle
		r := rectangles[rand.Intn(len(rectangles))]

		// Split it, producing 2 new rectangles
		s1, s2, err := splitRectangle(r, padding)
		if err != nil {
			//i--
			continue
		}
		// Add the new rectangles to the list
		rectangles = append(rectangles, s1, s2)
	}
	rectangles = append(rectangles, leftRect, rightRect)

	// Since we may have added duplicates, we should probably remove those...
	uniqueRectangles := make([]image.Rectangle, numSplits)
	seen := make(map[image.Rectangle]bool)
	for _, r := range rectangles {
		if !seen[r] {
			seen[r] = true
			uniqueRectangles = append(uniqueRectangles, r)
		}
	}

	intersections := findIntersections(uniqueRectangles)

	return intersections
}

func splitRectangle(r image.Rectangle, padding int) (s1, s2 image.Rectangle, err error) {
	vertical := rand.Intn(2) == 0
	var min, max int
	if vertical {
		min = r.Min.X + padding
		max = r.Max.X - padding
		if max-min <= 0 {
			return r, r, errors.New("Not enough room to split")
		}
		offset := rand.Intn(max-min) + min
		s1 = image.Rect(r.Min.X, r.Min.Y, offset, r.Max.Y) // Left
		s2 = image.Rect(offset, r.Min.Y, r.Max.X, r.Max.Y) // Right
	} else {
		min = r.Min.Y + padding
		max = r.Max.Y - padding
		if max-min <= 0 {
			return r, r, errors.New("Not enough room to split")
		}
		offset := rand.Intn(max-min) + min
		s1 = image.Rect(r.Min.X, r.Min.Y, r.Max.X, offset) // Top
		s2 = image.Rect(r.Min.X, offset, r.Max.X, r.Max.Y) // Bottom
	}

	return
}

func findIntersections(rs []image.Rectangle) []image.Rectangle {
	seenIntersections := make(map[image.Rectangle]bool)

	// Find all intersections of all rectangles, then add those intersections to the list of rectangles.
	// Keep looking until all intersections have been found.
	for i, r := range rs {
		for _, s := range rs[i+1:] {
			intersection := r.Intersect(s)
			if !intersection.Empty() && !seenIntersections[intersection] {
				rs = append(rs, intersection)
				seenIntersections[intersection] = true
			}
		}
	}

	// Reduce the set of intersections down to a set of non-overlapping rectangles
	noOverlap := make([]image.Rectangle, 0)
	for _, r := range rs {
		rOverlaps := false
		for _, s := range rs {
			if r.Overlaps(s) && r != s && !r.In(s) {
				rOverlaps = true
				break
			}
		}
		if !rOverlaps {
			noOverlap = append(noOverlap, r)
		}
	}

	return noOverlap
}
