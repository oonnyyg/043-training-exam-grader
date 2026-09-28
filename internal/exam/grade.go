package exam

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
)

// Per-question grading outcomes.
const (
	StatusCorrect    = "correct"
	StatusPartial    = "partial"
	StatusWrong      = "wrong"
	StatusUnanswered = "unanswered"
)

// QuestionResult is the graded outcome of one question for one student.
type QuestionResult struct {
	QuestionID string  `json:"question_id"`
	Status     string  `json:"status"`
	Points     int     `json:"points"`
	Score      float64 `json:"score"`
}

// SheetResult is the full outcome for one answer-sheet line.
type SheetResult struct {
	Line      int              `json:"line"`
	StudentID string           `json:"student_id"`
	Valid     bool             `json:"valid"`
	Errors    []string         `json:"errors,omitempty"`
	Questions []QuestionResult `json:"questions,omitempty"`
	Total     float64          `json:"total,omitempty"`
	Passed    bool             `json:"passed,omitempty"`
}

type rawSheet struct {
	StudentID *string                    `json:"student_id"`
	Answers   map[string]json.RawMessage `json:"answers"`
}

// ProcessLine decodes, validates and grades one JSONL answer-sheet line.
// Malformed JSON is reported as an invalid result carrying only the line
// number; other lines are still processed.
func ProcessLine(paper *Paper, line int, payload []byte) SheetResult {
	result := SheetResult{Line: line}
	dec := json.NewDecoder(bytes.NewReader(payload))
	var raw rawSheet
	if err := dec.Decode(&raw); err != nil {
		result.Errors = []string{fmt.Sprintf("invalid JSON: %v", err)}
		return result
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); err != io.EOF {
		result.Errors = []string{"invalid JSON: unexpected data after the answer-sheet object"}
		return result
	}
	if raw.StudentID == nil {
		result.Errors = append(result.Errors, "missing student_id")
	} else if *raw.StudentID == "" {
		result.Errors = append(result.Errors, "empty student_id")
	} else {
		result.StudentID = *raw.StudentID
	}
	Grade(paper, &result, raw)
	return result
}

// Grade validates and grades one parsed answer sheet against the paper.
func Grade(paper *Paper, result *SheetResult, raw rawSheet) {
	var problems []string
	questionByID := make(map[string]*Question, len(paper.Questions))
	for i := range paper.Questions {
		questionByID[paper.Questions[i].ID] = &paper.Questions[i]
	}

	answers := raw.Answers
	if answers != nil {
		for qid := range answers {
			if _, ok := questionByID[qid]; !ok {
				problems = append(problems, fmt.Sprintf("unknown question id %q", qid))
			}
		}
	}

	graded := make([]QuestionResult, 0, len(paper.Questions))
	for i := range paper.Questions {
		q := &paper.Questions[i]
		qr := QuestionResult{QuestionID: q.ID, Points: q.Points}
		rawAnswer, answered := answers[q.ID]
		if answers == nil || !answered || isJSONNull(rawAnswer) {
			qr.Status = StatusUnanswered
			graded = append(graded, qr)
			continue
		}
		switch q.Type {
		case SingleChoice:
			score, status, err := gradeChoice(q, rawAnswer, true)
			if err != nil {
				problems = append(problems, fmt.Sprintf("question %q: %v", q.ID, err))
			}
			qr.Score, qr.Status = score, status
		case MultipleChoice:
			score, status, err := gradeChoice(q, rawAnswer, false)
			if err != nil {
				problems = append(problems, fmt.Sprintf("question %q: %v", q.ID, err))
			}
			qr.Score, qr.Status = score, status
		case Numeric:
			score, status, err := gradeNumeric(q, rawAnswer)
			if err != nil {
				problems = append(problems, fmt.Sprintf("question %q: %v", q.ID, err))
			}
			qr.Score, qr.Status = score, status
		}
		graded = append(graded, qr)
	}

	if len(problems) > 0 || len(result.Errors) > 0 {
		result.Errors = append(result.Errors, problems...)
		return
	}
	result.Valid = true
	result.Questions = graded
	total := 0.0
	for _, qr := range graded {
		total += qr.Score
	}
	result.Total = total
	result.Passed = total >= paper.PassScore
}

func gradeChoice(q *Question, raw json.RawMessage, single bool) (float64, string, error) {
	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil {
		return 0, "", fmt.Errorf("answer must be an array of option ids: %v", err)
	}
	if len(ids) == 0 {
		return 0, StatusUnanswered, nil
	}
	if single && len(ids) > 1 {
		return 0, "", fmt.Errorf("single-choice question submitted with %d options", len(ids))
	}
	defined := make(map[string]bool, len(q.Options))
	for _, opt := range q.Options {
		defined[opt.ID] = true
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !defined[id] {
			return 0, "", fmt.Errorf("undefined option %q", id)
		}
		if seen[id] {
			return 0, "", fmt.Errorf("duplicate option %q", id)
		}
		seen[id] = true
	}

	correctSet := make(map[string]bool, len(q.Correct))
	for _, id := range q.Correct {
		correctSet[id] = true
	}
	selectedCorrect := 0
	for _, id := range ids {
		if !correctSet[id] {
			return 0, StatusWrong, nil
		}
		selectedCorrect++
	}
	if single {
		return float64(q.Points), StatusCorrect, nil
	}
	if selectedCorrect == len(q.Correct) {
		return float64(q.Points), StatusCorrect, nil
	}
	score := float64(q.Points) * float64(selectedCorrect) / float64(len(q.Correct))
	return score, StatusPartial, nil
}

func gradeNumeric(q *Question, raw json.RawMessage) (float64, string, error) {
	var value float64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, "", fmt.Errorf("answer must be a single finite number: %v", err)
	}
	if !isFinite(value) {
		return 0, "", fmt.Errorf("answer must be a finite number, got %v", value)
	}
	if math.Abs(value-*q.Value) <= *q.Tolerance {
		return float64(q.Points), StatusCorrect, nil
	}
	return 0, StatusWrong, nil
}

func isJSONNull(raw json.RawMessage) bool {
	return string(bytes.TrimSpace(raw)) == "null"
}
