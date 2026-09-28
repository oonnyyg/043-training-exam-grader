package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"example.com/training-exam-grader/internal/exam"
)

const (
	resultsFileName = "results.jsonl"
	summaryFileName = "summary.json"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "examgrade:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("examgrade", flag.ContinueOnError)
	paperPath := fs.String("paper", "", "path to the JSON exam paper")
	answersPath := fs.String("answers", "", "path to the JSONL answer-sheet file")
	outputDir := fs.String("out", "", "directory for results.jsonl and summary.json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *paperPath == "" || *answersPath == "" || *outputDir == "" {
		return errors.New("usage: examgrade -paper paper.json -answers answers.jsonl -out report_dir")
	}

	paper, err := loadPaper(*paperPath)
	if err != nil {
		return fmt.Errorf("invalid paper: %w", err)
	}

	results, err := gradeFile(paper, *answersPath)
	if err != nil {
		return err
	}
	exam.MarkDuplicates(results)
	summary := exam.BuildSummary(paper, results)

	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := writeResults(*outputDir, results); err != nil {
		return err
	}
	if err := writeSummary(*outputDir, summary); err != nil {
		return err
	}
	fmt.Printf("graded %d sheet(s): %d valid, %d invalid\n",
		summary.TotalSheets, summary.ValidSheets, summary.InvalidSheets)
	return nil
}

func loadPaper(path string) (*exam.Paper, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var paper exam.Paper
	if err := json.Unmarshal(data, &paper); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := paper.Validate(); err != nil {
		return nil, err
	}
	return &paper, nil
}

func gradeFile(paper *exam.Paper, path string) ([]exam.SheetResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []exam.SheetResult
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		payload := scanner.Bytes()
		if len(bytes.TrimSpace(payload)) == 0 {
			continue
		}
		results = append(results, exam.ProcessLine(paper, line, append([]byte(nil), payload...)))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read answers: %w", err)
	}
	return results, nil
}

func writeResults(dir string, results []exam.SheetResult) error {
	path := dir + string(os.PathSeparator) + resultsFileName
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", resultsFileName, err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	for i := range results {
		if err := encoder.Encode(&results[i]); err != nil {
			return fmt.Errorf("write %s: %w", resultsFileName, err)
		}
	}
	return nil
}

func writeSummary(dir string, summary exam.Summary) error {
	path := dir + string(os.PathSeparator) + summaryFileName
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", summaryFileName, err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		return fmt.Errorf("write %s: %w", summaryFileName, err)
	}
	return nil
}
