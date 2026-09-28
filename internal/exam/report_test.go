package exam

import "testing"

func TestDuplicateStudentIDsInvalidateAll(t *testing.T) {
	paper := validPaper()
	results := []SheetResult{
		ProcessLine(paper, 1, []byte(`{"student_id":"S1","answers":{"q1":["A"]}}`)),
		ProcessLine(paper, 2, []byte(`{"student_id":"S1","answers":{"q1":["B"]}}`)),
		ProcessLine(paper, 3, []byte(`{"student_id":"S2","answers":{"q1":["A"]}}`)),
	}
	MarkDuplicates(results)
	if results[0].Valid || results[1].Valid {
		t.Fatalf("both duplicate sheets must be invalid: %+v %+v", results[0], results[1])
	}
	if results[0].Questions != nil || results[1].Total != 0 {
		t.Fatalf("duplicate sheets must drop grading results")
	}
	if !results[2].Valid {
		t.Fatalf("unique sheet must stay valid: %+v", results[2])
	}
}

func TestSummaryStats(t *testing.T) {
	paper := validPaper()
	results := []SheetResult{
		ProcessLine(paper, 1, []byte(`{"student_id":"S1","answers":{"q1":["A"],"q2":["A","B"],"q3":10}}`)),
		ProcessLine(paper, 2, []byte(`{"student_id":"S2","answers":{"q1":["B"],"q2":["A"],"q3":20}}`)),
		ProcessLine(paper, 3, []byte(`{"student_id":"S3","answers":{"q9":["A"]}}`)),
	}
	MarkDuplicates(results)
	summary := BuildSummary(paper, results)

	if summary.TotalSheets != 3 || summary.ValidSheets != 2 || summary.InvalidSheets != 1 {
		t.Fatalf("counts: %+v", summary)
	}
	if summary.MeanScore == nil {
		t.Fatal("mean score must be present with valid sheets")
	}
	if want := (10.0 + 2.0) / 2.0; *summary.MeanScore != want {
		t.Fatalf("mean = %v, want %v", *summary.MeanScore, want)
	}
	if summary.PassCount != 1 {
		t.Fatalf("pass count = %d, want 1", summary.PassCount)
	}
	if *summary.QuestionStats[0].FullRate != 0.5 || summary.QuestionStats[0].FullMarks != 1 {
		t.Fatalf("q1 full rate: %+v", summary.QuestionStats[0])
	}
	if *summary.QuestionStats[2].FullRate != 0.5 {
		t.Fatalf("q3 full rate: %+v", summary.QuestionStats[2])
	}
}

func TestSummaryWithNoValidSheets(t *testing.T) {
	paper := validPaper()
	results := []SheetResult{
		ProcessLine(paper, 1, []byte(`{"student_id":"S1","answers":{"q1":["Z"]}}`)),
		ProcessLine(paper, 2, []byte(`{broken`)),
	}
	MarkDuplicates(results)
	summary := BuildSummary(paper, results)
	if summary.MeanScore != nil {
		t.Fatalf("mean must be null, got %v", *summary.MeanScore)
	}
	if summary.PassCount != 0 {
		t.Fatalf("pass count must be 0, got %d", summary.PassCount)
	}
	for _, stat := range summary.QuestionStats {
		if stat.FullRate != nil {
			t.Fatalf("full rate must be null: %+v", stat)
		}
	}
}

func TestSummaryEmptyFile(t *testing.T) {
	paper := validPaper()
	summary := BuildSummary(paper, nil)
	if summary.TotalSheets != 0 || summary.ValidSheets != 0 || summary.InvalidSheets != 0 {
		t.Fatalf("empty summary: %+v", summary)
	}
	if summary.MeanScore != nil || summary.PassCount != 0 {
		t.Fatalf("empty file stats: %+v", summary)
	}
	if len(summary.QuestionStats) != len(paper.Questions) {
		t.Fatalf("question stats should still be emitted")
	}
}

func TestUnansweredParticipatesInStats(t *testing.T) {
	paper := validPaper()
	results := []SheetResult{
		ProcessLine(paper, 1, []byte(`{"student_id":"S1","answers":{}}`)),
		ProcessLine(paper, 2, []byte(`{"student_id":"S2","answers":{"q1":["A"],"q2":["A","B"],"q3":10}}`)),
	}
	summary := BuildSummary(paper, results)
	if *summary.QuestionStats[0].FullRate != 0.5 {
		t.Fatalf("unanswered should count in denominator: %+v", summary.QuestionStats[0])
	}
}
