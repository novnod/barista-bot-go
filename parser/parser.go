package parser

import (
	"bufio"
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/fogleman/gg"
)

type GoProblem struct {
	Name  string
	Black []string
	White []string
}

type GoParser struct {
	Problems []*GoProblem
}

// LoadProblems reads an SGF file line-by-line, parsing each as a GoProblem
func (p *GoParser) LoadProblems(fileLocation string) error {
	file, err := os.Open(fileLocation)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		prob, err := p.ParseSGFLine(line)
		if err != nil {
			log.Printf("skipping line: %s: %v", line, err)
			continue
		}
		p.Problems = append(p.Problems, prob)
	}
	return scanner.Err()
}

// ParseSGFLine extracts stones and comment, returning a GoProblem or error
func (p *GoParser) ParseSGFLine(line string) (*GoProblem, error) {
	black := extractCoords(blackRunRe, line)
	white := extractCoords(whiteRunRe, line)
	comment := extractComment(commentRe, line)

	if len(black) == 0 && len(white) == 0 && comment == "" {
		return nil, fmt.Errorf("no SGF properties found in line")
	}

	return &GoProblem{Name: comment, Black: black, White: white}, nil
}

// RenderProblem draws the GoProblem onto a 19×19 board PNG.
// boardsizePx is the total image width/height in pixels (e.g. 800).
// marginPx leaves blank space around the outer lines (e.g. 40).
func RenderProblem(p *GoProblem, outputDir string, boardsizePx, marginPx int) (string, error) {
	dc := gg.NewContext(boardsizePx, boardsizePx)
	dc.SetColor(color.RGBA{R: 240, G: 200, B: 150, A: 255}) // light wood background
	dc.Clear()

	// Draw grid
	n := 19
	step := float64(boardsizePx-2*marginPx) / float64(n-1)
	dc.SetLineWidth(2)
	dc.SetColor(color.Black)
	for i := range n {
		x := float64(marginPx) + float64(i)*step
		dc.DrawLine(x, float64(marginPx), x, float64(boardsizePx-marginPx))
		dc.DrawLine(float64(marginPx), x, float64(boardsizePx-marginPx), x)
	}
	dc.Stroke()

	// Draw star points (hoshi)
	radius := step * 0.1
	for _, ix := range []int{3, 9, 15} {
		for _, iy := range []int{3, 9, 15} {
			cx := float64(marginPx) + float64(ix)*step
			cy := float64(marginPx) + float64(iy)*step
			dc.DrawCircle(cx, cy, radius)
			dc.Fill()
		}
	}

	// Helper to draw stones
	drawStone := func(coord string, fill color.Color) error {
		x, y, err := sgfToIndex(coord)
		if err != nil {
			return err
		}
		cx := float64(marginPx) + float64(x)*step
		cy := float64(marginPx) + float64(y)*step
		stoneR := step * 0.4
		dc.DrawCircle(cx, cy, stoneR)
		dc.SetColor(fill)
		dc.Fill()
		dc.SetLineWidth(1)
		dc.SetColor(color.Black)
		dc.DrawCircle(cx, cy, stoneR)
		dc.Stroke()
		return nil
	}

	// Place stones
	for _, c := range p.Black {
		if err := drawStone(c, color.Black); err != nil {
			return "", err
		}
	}
	for _, c := range p.White {
		if err := drawStone(c, color.White); err != nil {
			return "", err
		}
	}

	// Label problem name
	dc.SetColor(color.Black)
	dc.LoadFontFace("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 14)
	dc.DrawStringAnchored(p.Name, float64(boardsizePx)/2, float64(boardsizePx)-10, 0.5, 0.5)

	// Save image
	filename := fmt.Sprintf("%s.png", sanitizeFilename(p.Name))
	outPath := filepath.Join(outputDir, filename)
	if err := dc.SavePNG(outPath); err != nil {
		return "", err
	}
	return outPath, nil
}

// --------- Utilities ---------

var (
	blackRunRe = regexp.MustCompile(`AB((?:\[[a-s]{2}\])+)`)
	whiteRunRe = regexp.MustCompile(`AW((?:\[[a-s]{2}\])+)`)
	coordRe    = regexp.MustCompile(`\[([a-s]{2})\]`)
	commentRe  = regexp.MustCompile(`C\[(.*?)\]`)
)

// extractCoords finds a run of coords (e.g. "[cc][dd]...") and returns each coord
func extractCoords(runRe *regexp.Regexp, line string) []string {
	var coords []string
	if m := runRe.FindStringSubmatch(line); m != nil {
		for _, c := range coordRe.FindAllStringSubmatch(m[1], -1) {
			coords = append(coords, c[1])
		}
	}
	return coords
}

// extractComment returns the first matched comment, or empty string
func extractComment(re *regexp.Regexp, line string) string {
	if m := re.FindStringSubmatch(line); m != nil {
		return m[1]
	}
	return ""
}

// sgfToIndex converts SGF coordinate ("ab") to 0-based x,y indices
func sgfToIndex(s string) (int, int, error) {
	if len(s) != 2 {
		return 0, 0, fmt.Errorf("invalid coord %q", s)
	}
	x := int(s[0] - 'a')
	y := int(s[1] - 'a')
	if x < 0 || x >= 19 || y < 0 || y >= 19 {
		return 0, 0, fmt.Errorf("coord out of range %q", s)
	}
	return x, y, nil
}

// sanitizeFilename converts "Example problem" → "Example_problem"
func sanitizeFilename(name string) string {
	var out string
	for _, r := range name {
		switch {
		case r == ' ':
			out += "_"
		case (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-':
			out += string(r)
		}
	}
	if out == "" {
		return "problem"
	}
	return out
}
