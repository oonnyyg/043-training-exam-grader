package exam

import "fmt"

// Summary is the aggregate statistics written alongside per-sheet results.
type Summary struct {
	TotalSheets   int            `json:"total_sheets"`
	ValidSheets   int            `json:"valid_sheets"`
	InvalidSheets int            `json:"invalid_sheets"`
	MeanScore     *float64       `json:"mean_score"`
	PassCount     int            `json:"pass_count"`
	QuestionStats []QuestionStat `json:"question_stats"`
}

// QuestionStat reports the full-mark rate for one question.
type QuestionStat struct {
	QuestionID string   `json:"question_id"`
	Points     int      `json:"points"`
	FullMarks  int      `json:"full_marks"`
	FullRate   *float64 `json:"full_rate"`
}

// MarkDuplicates invalidates every sheet sharing a non-empty student id.
// First-come-first-served is intentionally not used.
func MarkDuplicates(results []SheetResult) {
	count := make(map[string]int)
	for i := range results {
		if results[i].StudentID != "" {
			count[results[i].StudentID]++
		}
	}
	for i := range results {
		id := results[i].StudentID
		if id != "" && count[id] > 1 {
			if results[i].Valid {
				results[i].Valid = false
				results[i].Questions = nil
				results[i].Total = 0
				results[i].Passed = false
			}
			results[i].Errors = append(results[i].Errors, fmt.Sprintf("duplicate student_id %q", id))
		}
	}
}

// BuildSummary computes aggregate statistics over graded sheets. Only valid
// sheets participate in denominators; unanswered questions still count.
func BuildSummary(paper *Paper, results []SheetResult) Summary {
	summary := Summary{
		TotalSheets:   len(results),
		QuestionStats: make([]QuestionStat, len(paper.Questions)),
	}
	fullMarks := make([]int, len(paper.Questions))
	scoreSum := 0.0
	for i := range results {
		r := &results[i]
		if r.Valid {
			summary.ValidSheets++
			scoreSum += r.Total
			if r.Passed {
				summary.PassCount++
			}
			for qi, qr := range r.Questions {
				if qr.Score == float64(qr.Points) {
					fullMarks[qi]++
				}
			}
		} else {
			summary.InvalidSheets++
		}
	}
	for i := range paper.Questions {
		stat := QuestionStat{QuestionID: paper.Questions[i].ID, Points: paper.Questions[i].Points}
		if summary.ValidSheets > 0 {
			stat.FullMarks = fullMarks[i]
			rate := float64(fullMarks[i]) / float64(summary.ValidSheets)
			stat.FullRate = &rate
		}
		summary.QuestionStats[i] = stat
	}
	if summary.ValidSheets > 0 {
		mean := scoreSum / float64(summary.ValidSheets)
		summary.MeanScore = &mean
	}
	return summary
}
