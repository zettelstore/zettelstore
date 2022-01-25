// Copyright 2012 - 2018 The ASCIIToSVG Contributors
// All rights reserved.

package draw

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"regexp"
	"sort"
	"strconv"
	"unicode/utf8"
)

// NewCanvas returns a new Canvas, initialized from the provided data. If tabWidth is set to a non-negative
// value, that value will be used to convert tabs to spaces within the grid. Creation of the Canvas
// can fail if the diagram contains invalid UTF-8 sequences.
func NewCanvas(data []byte, tabWidth int) (*Canvas, error) {
	c := &Canvas{
		options: optionMaps{
			"__a2s__closed__options__": {
				"fill": "#fff",
			},
		},
	}

	lines := bytes.Split(data, []byte("\n"))
	c.size.Y = len(lines)

	// Diagrams will often not be padded to a uniform width. To overcome this, we scan over
	// each line and figure out which is the longest. This becomes the width of the canvas.
	for i, line := range lines {
		if ok := utf8.Valid(line); !ok {
			return nil, fmt.Errorf("invalid UTF-8 encoding on line %d", i)
		}

		l, err := expandTabs(line, tabWidth)
		if err != nil {
			return nil, err
		}

		lines[i] = l

		if i1 := utf8.RuneCount(lines[i]); i1 > c.size.X {
			c.size.X = i1
		}
	}

	c.grid = make([]char, c.size.X*c.size.Y)
	c.visited = make([]bool, c.size.X*c.size.Y)
	for y, line := range lines {
		x := 0
		for len(line) > 0 {
			r, l := utf8.DecodeRune(line)
			c.grid[y*c.size.X+x] = char(r)
			x++
			line = line[l:]
		}

		for ; x < c.size.X; x++ {
			c.grid[y*c.size.X+x] = ' '
		}
	}

	c.findObjects()
	return c, nil
}

// The expandTabs function pads tab characters to the specified width of spaces for the provided
// line of input. We cannot simply pad based on byte-offset since our input is UTF-8 encoded.
// Fortunately, we can assume that this function is called that the line contains only valid
// UTF-8 sequences. We first decode the line rune-wise, and use individual runes to figure out
// where we are within the line. When we encounter a tab character, we expand based on our rune
// index.
func expandTabs(line []byte, tabWidth int) ([]byte, error) {
	// Initial sizing of our output slice assumes no UTF-8 bytes or tabs, since this is often
	// the common case.
	out := make([]byte, 0, len(line))

	// pos tracks our position in the input byte slice, while index tracks our position in the
	// resulting output slice.
	pos := 0
	index := 0
	for _, c := range line {
		if c == '\t' {
			// Loop over the remaining space count for this particular tabstop until
			// the next, replacing each position with a space.
			for s := tabWidth - (pos % tabWidth); s > 0; s-- {
				out = append(out, ' ')
				index++
			}
			pos++
		} else {
			// We need to know the byte length of the rune at this position so that we
			// can account for our tab expansion properly. So we first decode the rune
			// at this position to get its length in bytes, plop that rune back into our
			// output slice, and account accordingly.
			r, l := utf8.DecodeRune(line[pos:])
			if r == utf8.RuneError {
				return nil, fmt.Errorf("invalid rune at byte offset %d; rune offset %d", pos, index)
			}

			enc := make([]byte, l)
			utf8.EncodeRune(enc, r)
			out = append(out, enc...)

			pos += l
			index++
		}
	}

	return out, nil
}

type optionMaps map[string]map[string]interface{}

// Canvas is the parsed source data.
type Canvas struct {
	// (0,0) is top left.
	grid    []char
	visited []bool
	objects objects
	size    image.Point
	options optionMaps
}

// String provides a view into the underlying grid.
func (c *Canvas) String() string {
	return fmt.Sprintf("%+v", c.grid)
}

// Objects returns all the objects found in the underlying grid.
func (c *Canvas) Objects() []*object {
	return c.objects
}

// Size returns the visual dimensions of the Canvas.
func (c *Canvas) Size() image.Point {
	return c.size
}

// Options returns a map of options to apply to Objects based on the object's tag. This
// maps tag name to a map of option names to options.
func (c *Canvas) Options() optionMaps {
	return c.options
}

// EnclosingObjects returns the set of objects that contain this point in order from most
// to least specific.
func (c *Canvas) EnclosingObjects(p point) []*object {
	maxTL := point{x: -1, y: -1}

	var q []*object
	for _, o := range c.objects {
		// An object can't really contain another unless it is a polygon.
		if !o.IsClosed() {
			continue
		}

		if o.HasPoint(p) && o.Corners()[0].x > maxTL.x && o.Corners()[0].y > maxTL.y {
			q = append(q, o)
			maxTL.x = o.Corners()[0].x
			maxTL.y = o.Corners()[0].y
		}
	}

	return q
}

// findObjects finds all objects (lines, polygons, and text) within the underlying grid.
func (c *Canvas) findObjects() {
	p := point{}

	// Find any new paths by starting with a point that wasn't yet visited, beginning at the top
	// left of the grid.
	for y := 0; y < c.size.Y; y++ {
		p.y = y
		for x := 0; x < c.size.X; x++ {
			p.x = x
			if c.isVisited(p) {
				continue
			}
			if ch := c.at(p); ch.isPathStart() {
				// Found the start of a one or multiple connected paths. Traverse all
				// connecting points. This will generate multiple objects if multiple
				// paths (either open or closed) are found.
				c.visit(p)
				objs := c.scanPath([]point{p})
				for _, obj := range objs {
					// For all points in all objects found, mark the points as visited.
					for _, p := range obj.Points() {
						c.visit(p)
					}
				}
				c.objects = append(c.objects, objs...)
			}
		}
	}

	// A second pass through the grid attempts to identify any text within the grid.
	for y := 0; y < c.size.Y; y++ {
		p.y = y
		for x := 0; x < c.size.X; x++ {
			p.x = x
			if c.isVisited(p) {
				continue
			}
			if ch := c.at(p); ch.isTextStart() {
				obj := c.scanText(p)

				// scanText will return nil if the text at this area is simply
				// setting options on a container object.
				if obj == nil {
					continue
				}
				for _, p := range obj.Points() {
					c.visit(p)
				}
				c.objects = append(c.objects, obj)
			}
		}
	}

	sort.Sort(c.objects)
}

// scanPath tries to complete a total path (for lines or polygons) starting with some partial path.
// It recurses when it finds multiple unvisited outgoing paths.
func (c *Canvas) scanPath(points []point) objects {
	cur := points[len(points)-1]
	next := c.next(cur)

	// If there are no points that can progress traversal of the path, finalize the one we're
	// working on, and return it. This is the terminal condition in the passive flow.
	if len(next) == 0 {
		if len(points) == 1 {
			// Discard 'path' of 1 point. Do not mark point as visited.
			c.unvisit(cur)
			return nil
		}

		// TODO(dhobsd): Determine if path is sharing the line with another path. If so,
		// we may want to join the objects such that we don't get weird rendering artifacts.
		o := &object{points: points}
		o.seal(c)
		return objects{o}
	}

	// If we have hit a point that can create a closed path, create an object and close
	// the path. Additionally, recurse to other progress directions in case e.g. an open
	// path spawns from this point. Paths are always closed vertically.
	if cur.x == points[0].x && cur.y == points[0].y+1 {
		o := &object{points: points}
		o.seal(c)
		r := objects{o}
		return append(r, c.scanPath([]point{cur})...)
	}

	// We scan depth-first instead of breadth-first, making it possible to find a
	// closed path.
	var objs objects
	for _, n := range next {
		if c.isVisited(n) {
			continue
		}
		c.visit(n)
		p2 := make([]point, len(points)+1)
		copy(p2, points)
		p2[len(p2)-1] = n
		objs = append(objs, c.scanPath(p2)...)
	}
	return objs
}

// The next returns the points that can be used to make progress, scanning (in order) horizontal
// progress to the left or right, vertical progress above or below, or diagonal progress to NW,
// NE, SW, and SE. It skips any points already visited, and returns all of the possible progress
// points.
func (c *Canvas) next(pos point) []point {
	// Our caller must have called c.visit prior to calling this function.
	if !c.isVisited(pos) {
		panic(fmt.Errorf("internal error; revisiting %s", pos))
	}

	var out []point

	ch := c.at(pos)
	if ch.canHorizontal() {
		nextHorizontal := func(p point) {
			if !c.isVisited(p) && c.at(p).canHorizontal() {
				out = append(out, p)
			}
		}
		if c.canLeft(pos) {
			n := pos
			n.x--
			nextHorizontal(n)
		}
		if c.canRight(pos) {
			n := pos
			n.x++
			nextHorizontal(n)
		}
	}
	if ch.canVertical() {
		nextVertical := func(p point) {
			if !c.isVisited(p) && c.at(p).canVertical() {
				out = append(out, p)
			}
		}
		if c.canUp(pos) {
			n := pos
			n.y--
			nextVertical(n)
		}
		if c.canDown(pos) {
			n := pos
			n.y++
			nextVertical(n)
		}
	}
	if c.canDiagonal(pos) {
		nextDiagonal := func(from, to point) {
			if !c.isVisited(to) && c.at(to).canDiagonalFrom(c.at(from)) {
				out = append(out, to)
			}
		}
		if c.canUp(pos) {
			if c.canLeft(pos) {
				n := pos
				n.x--
				n.y--
				nextDiagonal(pos, n)
			}
			if c.canRight(pos) {
				n := pos
				n.x++
				n.y--
				nextDiagonal(pos, n)
			}
		}
		if c.canDown(pos) {
			if c.canLeft(pos) {
				n := pos
				n.x--
				n.y++
				nextDiagonal(pos, n)
			}
			if c.canRight(pos) {
				n := pos
				n.x++
				n.y++
				nextDiagonal(pos, n)
			}
		}
	}

	return out
}

// Used for matching [X, Y]: {...} tag definitions. These definitions target specific objects.
var objTagRE = regexp.MustCompile(`(\d+)\s*,\s*(\d+)$`)

// scanText extracts a line of text.
func (c *Canvas) scanText(start point) *object {
	obj := &object{points: []point{start}, isText: true}
	whiteSpaceStreak := 0
	cur := start

	tagged := 0
	tag := []rune{}
	tagDef := []rune{}

	for c.canRight(cur) {
		if cur.x == start.x && c.at(cur).isObjectStartTag() {
			tagged++
		} else if cur.x > start.x && c.at(cur).isObjectEndTag() {
			tagged++
		}

		cur.x++
		if c.isVisited(cur) {
			// If the point is already visited, we hit a polygon or a line.
			break
		}
		ch := c.at(cur)
		if !ch.isTextCont() {
			break
		}
		if tagged == 0 && ch.isSpace() {
			whiteSpaceStreak++
			// Stop when we see 3 consecutive whitespace points.
			if whiteSpaceStreak > 2 {
				break
			}
		} else {
			whiteSpaceStreak = 0
		}

		switch tagged {
		case 1:
			if !c.at(cur).isObjectEndTag() {
				tag = append(tag, rune(ch))
			}
		case 2:
			if c.at(cur).isTagDefinitionSeparator() {
				tagged++
			} else {
				tagged = -1
			}
		case 3:
			tagDef = append(tagDef, rune(ch))
		}

		obj.points = append(obj.points, cur)
	}

	// If we found a start and end tag marker, we either need to assign the tag to the object,
	// or we need to assign the specified options to the global canvas option space.
	if tagged == 2 {
		t := string(tag)
		if container := c.EnclosingObjects(start); container != nil {
			container[0].SetTag(t)
		}

		// The tag applies to the text object as well so that properties like
		// a2s:label can be set.
		obj.SetTag(t)
	} else if tagged == 3 {
		t := string(tag)

		// A tag definition targeting an object will not be found within any object; we need
		// to do that calculation here.
		if matches := objTagRE.FindStringSubmatch(t); matches != nil {
			if targetX, err := strconv.ParseInt(matches[1], 10, 0); err == nil {
				if targetY, err1 := strconv.ParseInt(matches[2], 10, 0); err1 == nil {
					for i, o := range c.objects {
						corner := o.Corners()[0]
						if corner.x == int(targetX) && corner.y == int(targetY) {
							c.objects[i].SetTag(t)
							break
						}
					}
				}
			}
		}
		// This is a tag definition. Parse the JSON and assign the options to the canvas.
		var m interface{}
		def := []byte(string(tagDef))
		if err := json.Unmarshal(def, &m); err != nil {
			// TODO(dhobsd): Gross.
			panic(err)
		}

		// The tag applies to the reference object as well, so that properties like
		// a2s:delref can be set.
		obj.SetTag(t)
		c.options[t] = m.(map[string]interface{})
	}

	// Trim the right side of the text object.
	for len(obj.points) != 0 && c.at(obj.points[len(obj.points)-1]).isSpace() {
		obj.points = obj.points[:len(obj.points)-1]
	}

	obj.seal(c)
	return obj
}

func (c *Canvas) at(p point) char {
	return c.grid[p.y*c.size.X+p.x]
}

func (c *Canvas) isVisited(p point) bool {
	return c.visited[p.y*c.size.X+p.x]
}

func (c *Canvas) visit(p point) {
	// TODO(dhobsd): Change code to ensure that visit() is called once and only
	// once per point.
	c.visited[p.y*c.size.X+p.x] = true
}

func (c *Canvas) unvisit(p point) {
	o := p.y*c.size.X + p.x
	if !c.visited[o] {
		panic(fmt.Errorf("internal error: point %+v never visited", p))
	}
	c.visited[o] = false
}

func (c *Canvas) canLeft(p point) bool {
	return p.x > 0
}

func (c *Canvas) canRight(p point) bool {
	return p.x < c.size.X-1
}

func (c *Canvas) canUp(p point) bool {
	return p.y > 0
}

func (c *Canvas) canDown(p point) bool {
	return p.y < c.size.Y-1
}

func (c *Canvas) canDiagonal(p point) bool {
	return (c.canLeft(p) || c.canRight(p)) && (c.canUp(p) || c.canDown(p))
}
