package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const paperJSON = `{
  "pass_score": 5,
  "questions": [
    {"id":"q1","type":"single","points":2,"options":[{"id":"A"},{"id":"B"}],"correct":["A"]},
    {"id":"q2","type":"multiple","points":4,"options":[{"id":"A"},{"id":"B"},{"id":"C"}],"correct":["A","B"]},
    {"id":"q3","type":"numeric","points":4,"value":10,"tolerance":0.5}
  ]
}`

const answersJSONL = `{"student_id":"S1","answers":{"q1":["A"],"q2":["A","B"],"q3":10.2}}
{"student_id":"S2","answers":{"q1":["B"],"q2":["A"],"q3":99}}
{broken
{"student_id":"S1","answers":{"q1":["A"]}}
`

func writeInputs(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	paperPath := filepath.Join(dir, "paper.json")
	answersPath := filepath.Join(dir, "answers.jsonl")
	outDir := filepath.Join(dir, "report")
	if err := os.WriteFile(paperPath, []byte(paperJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(answersPath, []byte(answersJSONL), 0o644); err != nil {
		t.Fatal(err)
	}
	return paperPath, answersPath, outDir
}

func TestEndToEnd(t *testing.T) {
	paperPath, answersPath, outDir := writeInputs(t)
	if err := run([]string{"-paper", paperPath, "-answers", answersPath, "-out", outDir}); err != nil {
		t.Fatalf("run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "results.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var results []map[string]any
	for _, line := range splitLines(data) {
		var row map[string]any
		if err := json.Unmarshal(line, &row); err != nil {
			t.Fatalf("invalid result line %q: %v", line, err)
		}
		results = append(results, row)
	}
	if len(results) != 4 {
		t.Fatalf("expected 4 result rows (blank line skipped), got %d", len(results))
	}
	if results[2]["line"].(float64) != 3 {
		t.Fatalf("line numbers must follow physical order: %+v", results[2])
	}
	if results[2]["valid"] != false {
		t.Fatalf("broken JSON must be invalid: %+v", results[2])
	}
	if results[0]["valid"] != false || results[3]["valid"] != false {
		t.Fatalf("duplicate S1 sheets must both be invalid")
	}

	summaryData, err := os.ReadFile(filepath.Join(outDir, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var summary map[string]any
	if err := json.Unmarshal(summaryData, &summary); err != nil {
		t.Fatal(err)
	}
	if summary["valid_sheets"].(float64) != 1 {
		t.Fatalf("only S2 is valid: %v", summary)
	}
}

func TestEmptyAnswersFile(t *testing.T) {
	paperPath, answersPath, outDir := writeInputs(t)
	if err := os.WriteFile(answersPath, []byte("\n  \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-paper", paperPath, "-answers", answersPath, "-out", outDir}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "results.jsonl")); err != nil {
		t.Fatalf("results file should exist even when empty: %v", err)
	}
	summaryData, _ := os.ReadFile(filepath.Join(outDir, "summary.json"))
	var summary map[string]any
	if err := json.Unmarshal(summaryData, &summary); err != nil {
		t.Fatal(err)
	}
	if summary["mean_score"] != nil || summary["pass_count"].(float64) != 0 {
		t.Fatalf("empty file summary: %v", summary)
	}
}

func TestInvalidPaperNoReport(t *testing.T) {
	dir := t.TempDir()
	paperPath := filepath.Join(dir, "paper.json")
	answersPath := filepath.Join(dir, "answers.jsonl")
	outDir := filepath.Join(dir, "report")
	os.WriteFile(answersPath, []byte(``), 0o644)

	os.WriteFile(paperPath, []byte(`{"pass_score":5,"questions":[]}`), 0o644)
	err := run([]string{"-paper", paperPath, "-answers", answersPath, "-out", outDir})
	if err == nil {
		t.Fatal("expected invalid paper error")
	}
	if _, statErr := os.Stat(outDir); !os.IsNotExist(statErr) {
		t.Fatalf("report directory must not be created for invalid paper")
	}
}

func TestExistingReportNotOverwrittenByInvalidPaper(t *testing.T) {
	paperPath, answersPath, outDir := writeInputs(t)
	if err := run([]string{"-paper", paperPath, "-answers", answersPath, "-out", outDir}); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(outDir, "summary.json")
	before, _ := os.ReadFile(marker)

	badPaper := filepath.Join(filepath.Dir(paperPath), "bad.json")
	os.WriteFile(badPaper, []byte(`{"pass_score":999,"questions":[]}`), 0o644)
	if err := run([]string{"-paper", badPaper, "-answers", answersPath, "-out", outDir}); err == nil {
		t.Fatal("expected invalid paper error")
	}
	after, _ := os.ReadFile(marker)
	if string(before) != string(after) {
		t.Fatalf("existing summary.json must not be overwritten")
	}
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if len(data[start:i]) > 0 {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
