package exam

import (
	"errors"
	"fmt"
	"math"
)

// QuestionType identifies how a question is answered and graded.
type QuestionType string

const (
	SingleChoice   QuestionType = "single"
	MultipleChoice QuestionType = "multiple"
	Numeric        QuestionType = "numeric"
)

// Option is one selectable choice of a choice question.
type Option struct {
	ID   string `json:"id"`
	Text string `json:"text,omitempty"`
}

// Question describes one exam question and its standard answer.
type Question struct {
	ID        string       `json:"id"`
	Type      QuestionType `json:"type"`
	Points    int          `json:"points"`
	Options   []Option     `json:"options,omitempty"`
	Correct   []string     `json:"correct,omitempty"`
	Value     *float64     `json:"value,omitempty"`
	Tolerance *float64     `json:"tolerance,omitempty"`
}

// Paper is a complete exam definition.
type Paper struct {
	Title     string     `json:"title,omitempty"`
	Questions []Question `json:"questions"`
	PassScore float64    `json:"pass_score"`
}

// TotalPoints returns the sum of every question's points.
func (p *Paper) TotalPoints() int {
	total := 0
	for i := range p.Questions {
		total += p.Questions[i].Points
	}
	return total
}

// Validate checks every structural and grading rule of the paper.
func (p *Paper) Validate() error {
	if len(p.Questions) == 0 {
		return errors.New("paper must contain at least one question")
	}
	seen := make(map[string]bool, len(p.Questions))
	for i := range p.Questions {
		q := &p.Questions[i]
		if q.ID == "" {
			return fmt.Errorf("question #%d has an empty id", i+1)
		}
		if seen[q.ID] {
			return fmt.Errorf("duplicate question id %q", q.ID)
		}
		seen[q.ID] = true
		if q.Points <= 0 {
			return fmt.Errorf("question %q: points must be a positive integer, got %d", q.ID, q.Points)
		}
		switch q.Type {
		case SingleChoice, MultipleChoice:
			if err := validateChoice(q); err != nil {
				return err
			}
		case Numeric:
			if err := validateNumeric(q); err != nil {
				return err
			}
		default:
			return fmt.Errorf("question %q: unknown type %q", q.ID, q.Type)
		}
	}
	total := p.TotalPoints()
	if math.IsNaN(p.PassScore) || math.IsInf(p.PassScore, 0) || p.PassScore < 0 || p.PassScore > float64(total) {
		return fmt.Errorf("pass_score must be between 0 and %d, got %v", total, p.PassScore)
	}
	return nil
}

func validateChoice(q *Question) error {
	if len(q.Options) == 0 {
		return fmt.Errorf("question %q: choice questions require at least one option", q.ID)
	}
	optionIDs := make(map[string]bool, len(q.Options))
	for _, opt := range q.Options {
		if opt.ID == "" {
			return fmt.Errorf("question %q: option has an empty id", q.ID)
		}
		if optionIDs[opt.ID] {
			return fmt.Errorf("question %q: duplicate option id %q", q.ID, opt.ID)
		}
		optionIDs[opt.ID] = true
	}
	if len(q.Correct) == 0 {
		return fmt.Errorf("question %q: standard answer must list at least one correct option", q.ID)
	}
	correct := make(map[string]bool, len(q.Correct))
	for _, id := range q.Correct {
		if !optionIDs[id] {
			return fmt.Errorf("question %q: standard answer references undefined option %q", q.ID, id)
		}
		if correct[id] {
			return fmt.Errorf("question %q: standard answer repeats option %q", q.ID, id)
		}
		correct[id] = true
	}
	if q.Type == SingleChoice && len(q.Correct) != 1 {
		return fmt.Errorf("question %q: single-choice question must have exactly one correct option, got %d", q.ID, len(q.Correct))
	}
	return nil
}

func validateNumeric(q *Question) error {
	if q.Value == nil {
		return fmt.Errorf("question %q: numeric question requires a finite value", q.ID)
	}
	if !isFinite(*q.Value) {
		return fmt.Errorf("question %q: value must be finite, got %v", q.ID, *q.Value)
	}
	if q.Tolerance == nil {
		return fmt.Errorf("question %q: numeric question requires a tolerance", q.ID)
	}
	if !isFinite(*q.Tolerance) || *q.Tolerance < 0 {
		return fmt.Errorf("question %q: tolerance must be a finite non-negative number, got %v", q.ID, *q.Tolerance)
	}
	return nil
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
