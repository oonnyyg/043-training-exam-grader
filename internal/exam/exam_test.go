package exam

import (
	"math"
	"testing"
)

func validPaper() *Paper {
	value, tolerance := 10.0, 0.5
	return &Paper{
		PassScore: 5,
		Questions: []Question{
			{
				ID: "q1", Type: SingleChoice, Points: 2,
				Options: []Option{{ID: "A"}, {ID: "B"}}, Correct: []string{"A"},
			},
			{
				ID: "q2", Type: MultipleChoice, Points: 4,
				Options: []Option{{ID: "A"}, {ID: "B"}, {ID: "C"}}, Correct: []string{"A", "B"},
			},
			{
				ID: "q3", Type: Numeric, Points: 4,
				Value: &value, Tolerance: &tolerance,
			},
		},
	}
}

func TestValidPaper(t *testing.T) {
	if err := validPaper().Validate(); err != nil {
		t.Fatalf("expected valid paper, got %v", err)
	}
	if got := validPaper().TotalPoints(); got != 10 {
		t.Fatalf("total points = %d, want 10", got)
	}
}

func TestPaperValidation(t *testing.T) {
	bad := func(modify func(*Paper)) *Paper {
		p := validPaper()
		modify(p)
		return p
	}
	tests := []struct {
		name  string
		paper *Paper
	}{
		{"no questions", bad(func(p *Paper) { p.Questions = nil })},
		{"empty question id", bad(func(p *Paper) { p.Questions[0].ID = "" })},
		{"duplicate question id", bad(func(p *Paper) { p.Questions[1].ID = "q1" })},
		{"zero points", bad(func(p *Paper) { p.Questions[0].Points = 0 })},
		{"negative points", bad(func(p *Paper) { p.Questions[0].Points = -3 })},
		{"unknown type", bad(func(p *Paper) { p.Questions[0].Type = "essay" })},
		{"empty option id", bad(func(p *Paper) { p.Questions[0].Options[0].ID = "" })},
		{"duplicate option id", bad(func(p *Paper) { p.Questions[0].Options[1].ID = "A" })},
		{"correct not an option", bad(func(p *Paper) { p.Questions[0].Correct = []string{"Z"} })},
		{"repeated correct", bad(func(p *Paper) { p.Questions[1].Correct = []string{"A", "A"} })},
		{"single with two correct", bad(func(p *Paper) { p.Questions[0].Correct = []string{"A", "B"} })},
		{"multiple with none correct", bad(func(p *Paper) { p.Questions[1].Correct = nil })},
		{"missing numeric value", bad(func(p *Paper) { p.Questions[2].Value = nil })},
		{"non-finite value", bad(func(p *Paper) { v := math.NaN(); p.Questions[2].Value = &v })},
		{"negative tolerance", bad(func(p *Paper) { v := -0.1; p.Questions[2].Tolerance = &v })},
		{"pass below zero", bad(func(p *Paper) { p.PassScore = -1 })},
		{"pass above total", bad(func(p *Paper) { p.PassScore = 10.1 })},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.paper.Validate(); err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}

func TestPassScoreBoundaryValid(t *testing.T) {
	p := validPaper()
	p.PassScore = 0
	if err := p.Validate(); err != nil {
		t.Fatalf("pass score 0: %v", err)
	}
	p.PassScore = 10
	if err := p.Validate(); err != nil {
		t.Fatalf("pass score 10: %v", err)
	}
}
