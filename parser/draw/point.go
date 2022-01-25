// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package draw

import "fmt"

// A renderHint suggests ways the SVG renderer may appropriately represent this point.
type renderHint int

const (
	// noHint indicates no hints are provided for this point.
	_ renderHint = iota
	// roundedCorner indicates the renderer should smooth corners on this path.
	roundedCorner
	// startMarker indicates this point should have an SVG marker-start attribute.
	startMarker
	// endMarker indicates this point should have an SVG marker-end attribute.
	endMarker
	// tick indicates the renderer should mark a tick in the path at this point.
	tick
	// dot indicates the renderer should insert a filled dot in the path at this point.
	dot
)

// A point is an X,Y coordinate in the diagram's grid. The grid represents (0, 0) as the top-left
// of the diagram. The point also provides hints to the renderer as to how it should be interpreted.
type point struct {
	// The x coordinate of this point.
	x int
	// The y coordinate of this point.
	y int
	// Hints for the renderer.
	hint renderHint
}

// String implements fmt.Stringer on Point.
func (p point) String() string {
	return fmt.Sprintf("(%d,%d)", p.x, p.y)
}

// isHorizontal returns true if p1 and p2 are horizontally aligned.
func isHorizontal(p1, p2 point) bool {
	d := p1.x - p2.x
	return d <= 1 && d >= -1 && p1.y == p2.y
}

// isVertical returns true if p1 and p2 are vertically aligned.
func isVertical(p1, p2 point) bool {
	d := p1.y - p2.y
	return d <= 1 && d >= -1 && p1.x == p2.x
}

// The following functions return true when the diagonals are connected in various compass directions.
func isDiagonalSE(p1, p2 point) bool {
	return p1.x-p2.x == -1 && p1.y-p2.y == -1
}
func isDiagonalSW(p1, p2 point) bool {
	return p1.x-p2.x == 1 && p1.y-p2.y == -1
}
func isDiagonalNW(p1, p2 point) bool {
	return p1.x-p2.x == 1 && p1.y-p2.y == 1
}
func isDiagonalNE(p1, p2 point) bool {
	return p1.x-p2.x == -1 && p1.y-p2.y == 1
}
