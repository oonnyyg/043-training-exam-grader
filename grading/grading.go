package grading

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
)

type rawSheet struct {
	CandidateID *string                    `json:"candidate_id"`
	Answers     map[string]json.RawMessage `json:"answers"`
}

type parsedSheet struct {
	id      string
	idFound bool
	raw     rawSheet
	errs    []string
}

// GradeFile 对 JSONL 答卷逐行阅卷，返回与输入行一一对应的结果及汇总。
func GradeFile(paper *Paper, lines [][]byte) ([]Result, Summary) {
	parsedSheets := make([]parsedSheet, len(lines))
	idLines := make(map[string][]int)

	for i, line := range lines {
		cur := parsedSheet{}
		if len(bytes.TrimSpace(line)) == 0 {
			cur.errs = append(cur.errs, "非法 JSON 行: 空行")
			parsedSheets[i] = cur
			continue
		}
		if err := json.Unmarshal(line, &cur.raw); err != nil {
			cur.errs = append(cur.errs, fmt.Sprintf("非法 JSON 行: %v", err))
			// 尝试单独提取考生编号，便于在结果中保留。
			if id, ok := extractCandidateID(line); ok {
				cur.id = id
				cur.idFound = true
			}
			parsedSheets[i] = cur
			continue
		}
		if cur.raw.CandidateID == nil {
			cur.errs = append(cur.errs, "缺少 candidate_id 字段")
		} else {
			cur.id = *cur.raw.CandidateID
			cur.idFound = true
			if cur.id == "" {
				cur.errs = append(cur.errs, "考生编号为空")
			} else {
				idLines[cur.id] = append(idLines[cur.id], i)
			}
		}
		if cur.raw.Answers == nil {
			cur.errs = append(cur.errs, "缺少 answers 字段或其类型错误")
		}
		parsedSheets[i] = cur
	}

	results := make([]Result, len(lines))
	for i := range lines {
		results[i] = gradeOne(paper, i+1, parsedSheets[i])
	}

	// 重复考生编号：涉及的所有答卷一律无效，不采用先到先得。
	for id, idxs := range idLines {
		if len(idxs) <= 1 {
			continue
		}
		for _, idx := range idxs {
			results[idx].Valid = false
			results[idx].Questions = nil
			results[idx].TotalScore = nil
			results[idx].Passed = nil
			results[idx].Errors = append(results[idx].Errors,
				fmt.Sprintf("考生编号 %q 重复（共出现 %d 次），相关答卷全部无效", id, len(idxs)))
		}
	}

	return results, buildSummary(paper, results)
}

// extractCandidateID 在可能损坏的 JSON 中尽力提取顶层 candidate_id 字符串。
// 只要扫描到该键值就返回，其后的语法错误被忽略。
func extractCandidateID(line []byte) (string, bool) {
	dec := json.NewDecoder(bytes.NewReader(line))
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", false
		}
		switch v := tok.(type) {
		case json.Delim:
			switch v {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
		case string:
			if depth == 1 && v == "candidate_id" {
				valTok, err := dec.Token()
				if err != nil {
					return "", false
				}
				if id, ok := valTok.(string); ok {
					return id, true
				}
				return "", false
			}
		}
	}
}

func gradeOne(paper *Paper, lineNo int, ps parsedSheet) Result {
	res := Result{Line: lineNo}
	if ps.idFound {
		res.CandidateID = ps.id
	}
	res.Errors = append([]string(nil), ps.errs...)

	if ps.raw.Answers != nil {
		// 未知题号：答卷中存在试卷里没有的题。
		known := make(map[string]bool, len(paper.Questions))
		for _, q := range paper.Questions {
			known[q.ID] = true
			if rawAns, present := ps.raw.Answers[q.ID]; present {
				if msg, ok := validateAnswer(q, rawAns); !ok {
					res.Errors = append(res.Errors, msg)
				}
			}
		}
		for id := range ps.raw.Answers {
			if !known[id] {
				res.Errors = append(res.Errors, fmt.Sprintf("包含未知题号 %q", id))
			}
		}
	}

	if len(res.Errors) > 0 {
		return res
	}

	res.Valid = true
	res.Questions = make([]QuestionResult, 0, len(paper.Questions))
	total := 0.0
	for _, q := range paper.Questions {
		rawAns, present := ps.raw.Answers[q.ID]
		qr := QuestionResult{QuestionID: q.ID}
		if !present {
			qr.Status = StatusUnanswered
		} else {
			qr.Score, qr.Status = scoreAnswer(q, rawAns)
		}
		total += qr.Score
		res.Questions = append(res.Questions, qr)
	}
	res.TotalScore = &total
	passed := total >= paper.PassScore
	res.Passed = &passed
	return res
}

// validateAnswer 校验单题答案的类型与取值，返回错误信息与是否合法。
func validateAnswer(q *Question, raw json.RawMessage) (string, bool) {
	switch q.Type {
	case Single, Multiple:
		var opts []string
		if err := json.Unmarshal(raw, &opts); err != nil {
			return fmt.Sprintf("题 %q答案类型错误（应为选项编号列表）", q.ID), false
		}
		seen := make(map[string]bool, len(opts))
		for _, opt := range opts {
			if !contains(q.Options, opt) {
				return fmt.Sprintf("题 %q引用了未定义选项 %q", q.ID, opt), false
			}
			if seen[opt] {
				return fmt.Sprintf("题 %q选项 %q 重复提交", q.ID, opt), false
			}
			seen[opt] = true
		}
		if q.Type == Single && len(opts) > 1 {
			return fmt.Sprintf("题 %q为单选题却提交了多个选项", q.ID), false
		}
	case Number:
		var num float64
		if err := json.Unmarshal(raw, &num); err != nil {
			return fmt.Sprintf("题 %q答案类型错误（应为单个数值）", q.ID), false
		}
		if math.IsNaN(num) || math.IsInf(num, 0) {
			return fmt.Sprintf("题 %q答案必须是有限数值", q.ID), false
		}
	}
	return "", true
}

// scoreAnswer 对合法答案评分。
func scoreAnswer(q *Question, raw json.RawMessage) (float64, string) {
	switch q.Type {
	case Single:
		var opts []string
		_ = json.Unmarshal(raw, &opts)
		// 空选项列表视为答错，得零分。
		if len(opts) == 1 && opts[0] == q.CorrectOptions[0] {
			return q.Score, StatusCorrect
		}
		return 0, StatusWrong
	case Multiple:
		var opts []string
		_ = json.Unmarshal(raw, &opts)
		correctSet := make(map[string]bool, len(q.CorrectOptions))
		for _, opt := range q.CorrectOptions {
			correctSet[opt] = true
		}
		for _, opt := range opts {
			if !correctSet[opt] {
				return 0, StatusWrong // 包含错误选项，零分
			}
		}
		correctChosen := 0
		for _, opt := range opts {
			if correctSet[opt] {
				correctChosen++
			}
		}
		got := q.Score * float64(correctChosen) / float64(len(q.CorrectOptions))
		switch {
		case correctChosen == len(q.CorrectOptions):
			return got, StatusCorrect
		case correctChosen == 0:
			return got, StatusWrong
		default:
			return got, StatusPartial
		}
	case Number:
		var num float64
		_ = json.Unmarshal(raw, &num)
		if math.Abs(num-q.CorrectValue) <= q.Tolerance {
			return q.Score, StatusCorrect
		}
		return 0, StatusWrong
	}
	return 0, StatusWrong
}

func buildSummary(paper *Paper, results []Result) Summary {
	s := Summary{QuestionFullRate: make(map[string]*float64, len(paper.Questions))}
	fullCounts := make(map[string]int, len(paper.Questions))
	totalScore := 0.0

	for _, r := range results {
		if !r.Valid {
			s.InvalidCount++
			continue
		}
		s.ValidCount++
		totalScore += *r.TotalScore
		if *r.Passed {
			s.PassCount++
		}
		for _, qr := range r.Questions {
			// 满分率：得分等于该题分值（漏答自然不计入）。
			if qr.Score == paper.Question(qr.QuestionID).Score {
				fullCounts[qr.QuestionID]++
			}
		}
	}

	if s.ValidCount > 0 {
		avg := totalScore / float64(s.ValidCount)
		s.AverageScore = &avg
		for _, q := range paper.Questions {
			rate := float64(fullCounts[q.ID]) / float64(s.ValidCount)
			s.QuestionFullRate[q.ID] = &rate
		}
	} else {
		for _, q := range paper.Questions {
			s.QuestionFullRate[q.ID] = nil
		}
	}
	return s
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
