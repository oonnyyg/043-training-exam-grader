package grading

import (
	"encoding/json"
	"fmt"
	"math"
)

type rawPaper struct {
	PassScore *float64      `json:"pass_score"`
	Questions []rawQuestion `json:"questions"`
}

type rawQuestion struct {
	ID             string   `json:"id"`
	Type           string   `json:"type"`
	Score          *float64 `json:"score"`
	Options        []string `json:"options"`
	CorrectOptions []string `json:"correct_options"`
	CorrectValue   *float64 `json:"correct_value"`
	Tolerance      *float64 `json:"tolerance"`
}

// LoadPaper 解析并校验 JSON 试卷数据。
func LoadPaper(data []byte) (*Paper, error) {
	var raw rawPaper
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("试卷 JSON 解析失败: %w", err)
	}

	if raw.PassScore == nil {
		return nil, fmt.Errorf("试卷缺少 pass_score 字段")
	}
	passScore := *raw.PassScore
	if math.IsNaN(passScore) || math.IsInf(passScore, 0) {
		return nil, fmt.Errorf("及格线必须是有限数值")
	}

	if len(raw.Questions) == 0 {
		return nil, fmt.Errorf("试卷至少需要一道题")
	}

	paper := &Paper{
		PassScore: passScore,
		Questions: make([]*Question, 0, len(raw.Questions)),
		index:     make(map[string]*Question),
	}

	for i, rq := range raw.Questions {
		at := fmt.Sprintf("第 %d 题", i+1)
		if rq.ID == "" {
			return nil, fmt.Errorf("%s题号为空", at)
		}
		at = fmt.Sprintf("题 %q", rq.ID)
		if _, exists := paper.index[rq.ID]; exists {
			return nil, fmt.Errorf("题号 %q 重复", rq.ID)
		}
		if rq.Score == nil {
			return nil, fmt.Errorf("%s缺少 score 字段", at)
		}
		score := *rq.Score
		if math.IsNaN(score) || math.IsInf(score, 0) || score <= 0 || score != math.Trunc(score) {
			return nil, fmt.Errorf("%s分值必须是正整数", at)
		}

		q := &Question{ID: rq.ID, Score: score}

		switch QuestionType(rq.Type) {
		case Single, Multiple:
			q.Type = QuestionType(rq.Type)
			if err := validateChoice(at, rq, q); err != nil {
				return nil, err
			}
		case Number:
			q.Type = Number
			if rq.CorrectValue == nil {
				return nil, fmt.Errorf("%s缺少 correct_value 字段", at)
			}
			if math.IsNaN(*rq.CorrectValue) || math.IsInf(*rq.CorrectValue, 0) {
				return nil, fmt.Errorf("%s标准值必须是有限数值", at)
			}
			if rq.Tolerance == nil {
				return nil, fmt.Errorf("%s缺少 tolerance 字段", at)
			}
			if math.IsNaN(*rq.Tolerance) || math.IsInf(*rq.Tolerance, 0) || *rq.Tolerance < 0 {
				return nil, fmt.Errorf("%s容限必须是有限非负数", at)
			}
			q.CorrectValue = *rq.CorrectValue
			q.Tolerance = *rq.Tolerance
		default:
			return nil, fmt.Errorf("%s类型 %q 非法（仅支持 single/multiple/number）", at, rq.Type)
		}

		paper.index[q.ID] = q
		paper.Questions = append(paper.Questions, q)
	}

	total := paper.TotalScore()
	if passScore < 0 || passScore > total {
		return nil, fmt.Errorf("及格线 %g 超出合法范围 [0, %g]", passScore, total)
	}

	return paper, nil
}

func validateChoice(at string, rq rawQuestion, q *Question) error {
	if len(rq.Options) == 0 {
		return fmt.Errorf("%s缺少选项", at)
	}
	seen := make(map[string]bool, len(rq.Options))
	for _, opt := range rq.Options {
		if opt == "" {
			return fmt.Errorf("%s选项编号为空", at)
		}
		if seen[opt] {
			return fmt.Errorf("%s选项编号 %q 重复", at, opt)
		}
		seen[opt] = true
	}
	q.Options = append([]string(nil), rq.Options...)

	if rq.CorrectOptions == nil {
		return fmt.Errorf("%s缺少 correct_options 字段", at)
	}
	correctSeen := make(map[string]bool, len(rq.CorrectOptions))
	for _, opt := range rq.CorrectOptions {
		if !seen[opt] {
			return fmt.Errorf("%s标准答案 %q 不在选项中", at, opt)
		}
		if correctSeen[opt] {
			return fmt.Errorf("%s标准答案 %q 重复", at, opt)
		}
		correctSeen[opt] = true
	}
	switch q.Type {
	case Single:
		if len(rq.CorrectOptions) != 1 {
			return fmt.Errorf("%s单选必须恰有一个正确选项", at)
		}
	case Multiple:
		if len(rq.CorrectOptions) == 0 {
			return fmt.Errorf("%s多选至少有一个正确选项", at)
		}
	}
	q.CorrectOptions = append([]string(nil), rq.CorrectOptions...)
	return nil
}
