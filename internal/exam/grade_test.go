package exam

import "testing"

func gradeSheet(t *testing.T, paper *Paper, line string) SheetResult {
	t.Helper()
	return ProcessLine(paper, 1, []byte(line))
}

func TestSingleChoiceGrading(t *testing.T) {
	paper := validPaper()
	correct := gradeSheet(t, paper, `{"student_id":"S1","answers":{"q1":["A"]}}`)
	if !correct.Valid || correct.Questions[0].Status != StatusCorrect || correct.Questions[0].Score != 2 {
		t.Fatalf("correct answer: %+v", correct)
	}
	wrong := gradeSheet(t, paper, `{"student_id":"S2","answers":{"q1":["B"]}}`)
	if !wrong.Valid || wrong.Questions[0].Status != StatusWrong || wrong.Questions[0].Score != 0 {
		t.Fatalf("wrong answer: %+v", wrong)
	}
}

func TestMultipleChoicePartialScore(t *testing.T) {
	paper := validPaper()
	half := gradeSheet(t, paper, `{"student_id":"S1","answers":{"q2":["A"]}}`)
	if half.Questions[1].Status != StatusPartial || half.Questions[1].Score != 2 {
		t.Fatalf("half correct: %+v", half.Questions[1])
	}

	full := gradeSheet(t, paper, `{"student_id":"S2","answers":{"q2":["B","A"]}}`)
	if full.Questions[1].Status != StatusCorrect || full.Questions[1].Score != 4 {
		t.Fatalf("all correct: %+v", full.Questions[1])
	}

	withWrong := gradeSheet(t, paper, `{"student_id":"S3","answers":{"q2":["A","C"]}}`)
	if withWrong.Questions[1].Status != StatusWrong || withWrong.Questions[1].Score != 0 {
		t.Fatalf("contains wrong option: %+v", withWrong.Questions[1])
	}

	empty := gradeSheet(t, paper, `{"student_id":"S4","answers":{"q2":[]}}`)
	if empty.Questions[1].Status != StatusUnanswered || empty.Questions[1].Score != 0 {
		t.Fatalf("empty list: %+v", empty.Questions[1])
	}

	paper3 := &Paper{PassScore: 0, Questions: []Question{{
		ID: "m", Type: MultipleChoice, Points: 3,
		Options: []Option{{ID: "A"}, {ID: "B"}, {ID: "C"}},
		Correct: []string{"A", "B", "C"},
	}}}
	third := gradeSheet(t, paper3, `{"student_id":"S5","answers":{"m":["A"]}}`)
	if third.Questions[0].Score != 1 {
		t.Fatalf("one of three: score = %v, want 1", third.Questions[0].Score)
	}
}

func TestNumericToleranceBoundary(t *testing.T) {
	paper := validPaper()
	for _, tc := range []struct {
		answer string
		want   float64
		status string
	}{
		{"9.5", 4, StatusCorrect},
		{"10.5", 4, StatusCorrect},
		{"9.500001", 4, StatusCorrect},
		{"10.500001", 0, StatusWrong},
		{"100", 0, StatusWrong},
	} {
		r := gradeSheet(t, paper, `{"student_id":"S1","answers":{"q3":`+tc.answer+`}}`)
		got := r.Questions[2]
		if got.Score != tc.want || got.Status != tc.status {
			t.Fatalf("answer %s: got score %v status %q, want %v %q",
				tc.answer, got.Score, got.Status, tc.want, tc.status)
		}
	}
}

func TestUnansweredCountsZero(t *testing.T) {
	paper := validPaper()
	r := gradeSheet(t, paper, `{"student_id":"S1","answers":{}}`)
	if !r.Valid || r.Total != 0 || r.Passed {
		t.Fatalf("empty answers: %+v", r)
	}
	for _, qr := range r.Questions {
		if qr.Status != StatusUnanswered || qr.Score != 0 {
			t.Fatalf("question %s should be unanswered zero, got %+v", qr.QuestionID, qr)
		}
	}

	missing := gradeSheet(t, paper, `{"student_id":"S2","answers":{"q1":["A"]}}`)
	if missing.Questions[1].Status != StatusUnanswered {
		t.Fatalf("omitted question: %+v", missing.Questions[1])
	}
}

func TestPassBoundaryInclusive(t *testing.T) {
	paper := validPaper()
	r := gradeSheet(t, paper, `{"student_id":"S1","answers":{"q1":["A"],"q2":["A"]}}`)
	if r.Total != 4 || r.Passed {
		t.Fatalf("score 4 should fail: %+v", r)
	}
	r2 := gradeSheet(t, paper, `{"student_id":"S2","answers":{"q1":["A"],"q3":10}}`)
	if r2.Total != 6 || !r2.Passed {
		t.Fatalf("score equal to pass score should pass: %+v", r2)
	}
}

func TestInvalidSheets(t *testing.T) {
	paper := validPaper()
	tests := []struct {
		name string
		line string
	}{
		{"malformed json", `{"student_id":`},
		{"missing student id", `{"answers":{"q1":["A"]}}`},
		{"empty student id", `{"student_id":"","answers":{"q1":["A"]}}`},
		{"unknown question", `{"student_id":"S1","answers":{"q9":["A"]}}`},
		{"undefined option", `{"student_id":"S1","answers":{"q1":["Z"]}}`},
		{"duplicate option", `{"student_id":"S1","answers":{"q2":["A","A"]}}`},
		{"single with multiple", `{"student_id":"S1","answers":{"q1":["A","B"]}}`},
		{"wrong answer type", `{"student_id":"S1","answers":{"q3":"ten"}}`},
		{"choice answer not array", `{"student_id":"S1","answers":{"q1":"A"}}`},
		{"non-finite numeric", `{"student_id":"S1","answers":{"q3":1e999}}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := gradeSheet(t, paper, tc.line)
			if r.Valid {
				t.Fatalf("expected invalid sheet, got %+v", r)
			}
			if len(r.Errors) == 0 {
				t.Fatalf("expected specific error message")
			}
			if r.Questions != nil || r.Total != 0 || r.Passed {
				t.Fatalf("invalid sheet must not carry grading results: %+v", r)
			}
		})
	}
}

func TestMalformedJSONPreservesLineOnly(t *testing.T) {
	r := ProcessLine(validPaper(), 42, []byte(`{broken`))
	if r.Valid || r.Line != 42 || r.StudentID != "" || len(r.Errors) != 1 {
		t.Fatalf("malformed line result: %+v", r)
	}
}
