package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSimpleProblem(t *testing.T) {
	sgf := "(;AB[cc][cd][dd]AW[dd][de]C[Example problem])"
	parser := GoParser{}
	problem, err := parser.ParseSGFLine(sgf)
	if err != nil {
		t.Fatalf("unexpected error occurred: %v", err)
	}
	if len(problem.Black) != 3 {
		t.Errorf("expected 3 black stones but got %d", len(problem.Black))
	}
	if len(problem.White) != 2 {
		t.Errorf("expected 2 white stones but got %d", len(problem.White))
	}
	if problem.Name != "Example problem" {
		t.Errorf("expected 'Example problem' but got %q", problem.Name)
	}
}

func TestParseLineNoComment(t *testing.T) {
	sgf := "(;AB[aa][bb]AW[cc][dd])"
	parser := GoParser{}
	problem, err := parser.ParseSGFLine(sgf)
	if err != nil {
		t.Fatalf("unexpected error occurred: %v", err)
	}
	if len(problem.Black) != 2 {
		t.Errorf("expected 2 black stones but got %d", len(problem.Black))
	}
	if len(problem.White) != 2 {
		t.Errorf("expected 2 white stones but got %d", len(problem.White))
	}
	if problem.Name != "" {
		t.Errorf("expected empty name but got %q", problem.Name)
	}
}

func TestParseReorderedProperties(t *testing.T) {
	sgf := "(;C[Reorder]AW[dd]AB[cc])"
	parser := GoParser{}
	problem, err := parser.ParseSGFLine(sgf)
	if err != nil {
		t.Fatalf("unexpected error occurred: %v", err)
	}
	if len(problem.Black) != 1 || problem.Black[0] != "cc" {
		t.Errorf("expected black[0] == 'cc', got %v", problem.Black)
	}
	if len(problem.White) != 1 || problem.White[0] != "dd" {
		t.Errorf("expected white[0] == 'dd', got %v", problem.White)
	}
	if problem.Name != "Reorder" {
		t.Errorf("expected name 'Reorder' but got %q", problem.Name)
	}
}

func TestParseInvalidLine(t *testing.T) {
	parser := GoParser{}
	_, err := parser.ParseSGFLine("invalid line")
	if err == nil {
		t.Error("expected error for invalid SGF line but got nil")
	}
}

func TestLoadEasyGoProblems(t *testing.T) {
	fileName := filepath.Join("..", "files", "cho-easy.sgf")
	parser := GoParser{}
	err := parser.LoadProblems(fileName)
	if err != nil {
		t.Errorf("unexpected problem occurred with loading problems: %v", err)
	}
	if len(parser.Problems) == 0 {
		t.Error("expected problems to be loaded but got none")
	}
}

func TestRenderProblem(t *testing.T) {
	// Create a sample problem
	prob := &GoProblem{
		Name:  "TestX",
		Black: []string{"dd", "ee"},
		White: []string{"cc"},
	}
	// Use a temporary directory for output
	outDir := t.TempDir()
	// Render with small board size
	imgPath, err := RenderProblem(prob, outDir, 200, 20)
	if err != nil {
		t.Fatalf("RenderProblem returned error: %v", err)
	}
	// Check file exists
	if _, err := os.Stat(imgPath); os.IsNotExist(err) {
		t.Errorf("expected image file at %s but not found", imgPath)
	}
	// Check filename matches problem name
	if filepath.Base(imgPath) != "TestX.png" {
		t.Errorf("expected filename 'TestX.png', got %s", filepath.Base(imgPath))
	}
}

func TestLoadMediumGoProblems(t *testing.T) {
	fileName := filepath.Join("..", "files", "cho-medium.sgf")
	parser := GoParser{}
	err := parser.LoadProblems(fileName)
	if err != nil {
		t.Errorf("unexpected problem occurred with loading problems: %v", err)
	}
	if len(parser.Problems) == 0 {
		t.Error("expected problems to be loaded but got none")
	}
}

func TestLoadHardGoProblems(t *testing.T) {
	fileName := filepath.Join("..", "files", "cho-hard.sgf")
	parser := GoParser{}
	err := parser.LoadProblems(fileName)
	if err != nil {
		t.Errorf("unexpected problem occurred with loading problems: %v", err)
	}
	if len(parser.Problems) == 0 {
		t.Error("expected problems to be loaded but got none")
	}
}

// TestDumpExampleImage writes a sample GoProblem image to 'tests/example.png' for manual inspection
func TestDumpExampleImage(t *testing.T) {
	prob := &GoProblem{
		Name:  "ExampleProblem",
		Black: []string{"dd", "ee"},
		White: []string{"cc", "ff"},
	}
	outDir := filepath.Join("tests")
	// ensure directory exists
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create tests dir: %v", err)
	}
	imgPath, err := RenderProblem(prob, outDir, 300, 30)
	if err != nil {
		t.Fatalf("RenderProblem failed: %v", err)
	}
	t.Logf("wrote example image to %s", imgPath)
}
