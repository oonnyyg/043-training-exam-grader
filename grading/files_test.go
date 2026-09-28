package grading

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunEndToEnd(t *testing.T) {
	dir := t.TempDir()
	paperFile := filepath.Join(dir, "paper.json")
	answersFile := filepath.Join(dir, "answers.jsonl")
	outDir := filepath.Join(dir, "out")

	if err := os.WriteFile(paperFile, []byte(validPaperJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	rows := "" +
		`{"candidate_id":"C1","answers":{"q1":["A"],"q2":["A","C"],"q3":10}}` + "\n" +
		`{"candidate_id":"C2","answers":{"q3":11}}` + "\n" +
		`{broken` + "\n"
	if err := os.WriteFile(answersFile, []byte(rows), 0o644); err != nil {
		t.Fatal(err)
	}

	results, summary, err := Run(paperFile, answersFile, outDir)
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("应有 3 条结果，实际 %d", len(results))
	}
	if summary.ValidCount != 2 || summary.InvalidCount != 1 {
		t.Fatalf("汇总错误: %+v", summary)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "results.jsonl"))
	if err != nil {
		t.Fatalf("results.jsonl 未生成: %v", err)
	}
	var got []Result
	for _, line := range splitLines(data) {
		var r Result
		if err := json.Unmarshal(line, &r); err != nil {
			t.Fatalf("结果行不是合法 JSON: %v", err)
		}
		got = append(got, r)
	}
	if len(got) != 3 || got[2].Valid || got[2].CandidateID != "" {
		t.Fatalf("结果文件内容异常: %+v", got)
	}

	sData, err := os.ReadFile(filepath.Join(outDir, "summary.json"))
	if err != nil {
		t.Fatalf("summary.json 未生成: %v", err)
	}
	var s Summary
	if err := json.Unmarshal(sData, &s); err != nil {
		t.Fatalf("汇总 JSON 非法: %v", err)
	}
	if s.PassCount != 1 {
		t.Fatalf("及格人数应为 1")
	}
}

func TestRunInvalidPaperDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(outDir, "results.jsonl")
	if err := os.WriteFile(marker, []byte("KEEP"), 0o644); err != nil {
		t.Fatal(err)
	}
	badPaper := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(badPaper, []byte(`{bad`), 0o644); err != nil {
		t.Fatal(err)
	}
	answers := filepath.Join(dir, "a.jsonl")
	if err := os.WriteFile(answers, []byte("\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := Run(badPaper, answers, outDir); err == nil {
		t.Fatalf("非法试卷应当报错")
	}
	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("原有报告不应被删除: %v", err)
	}
	if string(got) != "KEEP" {
		t.Fatalf("原有报告被覆盖: %q", got)
	}
}

func TestRunEmptyAnswersFile(t *testing.T) {
	dir := t.TempDir()
	paperFile := filepath.Join(dir, "paper.json")
	answersFile := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(paperFile, []byte(validPaperJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(answersFile, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	results, summary, err := Run(paperFile, answersFile, filepath.Join(dir, "out"))
	if err != nil {
		t.Fatalf("空答卷文件应正常处理: %v", err)
	}
	if len(results) != 0 || summary.ValidCount != 0 || summary.AverageScore != nil {
		t.Fatalf("空答卷统计异常: %+v", summary)
	}
}

func splitLines(data []byte) [][]byte {
	var out [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				out = append(out, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		out = append(out, data[start:])
	}
	return out
}
