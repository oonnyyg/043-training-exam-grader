package grading

import (
	"strings"
	"testing"
)

const validPaperJSON = `{
  "pass_score": 6,
  "questions": [
    {"id": "q1", "type": "single", "score": 2, "options": ["A","B","C"], "correct_options": ["A"]},
    {"id": "q2", "type": "multiple", "score": 4, "options": ["A","B","C","D"], "correct_options": ["A","C"]},
    {"id": "q3", "type": "number", "score": 3, "correct_value": 10, "tolerance": 0.5}
  ]
}`

func TestLoadPaperValid(t *testing.T) {
	p, err := LoadPaper([]byte(validPaperJSON))
	if err != nil {
		t.Fatalf("合法试卷被拒: %v", err)
	}
	if got := p.TotalScore(); got != 9 {
		t.Fatalf("满分应为 9，实际 %v", got)
	}
	if p.PassScore != 6 {
		t.Fatalf("及格线应为 6")
	}
	if len(p.Questions) != 3 {
		t.Fatalf("应有 3 道题")
	}
}

func TestLoadPaperInvalid(t *testing.T) {
	cases := map[string]string{
		"非法 JSON": `{not json`,
		"缺少题目":    `{"pass_score":0,"questions":[]}`,
		"题号为空":    `{"pass_score":0,"questions":[{"id":"","type":"single","score":1,"options":["A"],"correct_options":["A"]}]}`,
		"题号重复": `{"pass_score":0,"questions":[
			{"id":"q","type":"single","score":1,"options":["A"],"correct_options":["A"]},
			{"id":"q","type":"single","score":1,"options":["A"],"correct_options":["A"]}]}`,
		"分值为零":      `{"pass_score":0,"questions":[{"id":"q","type":"single","score":0,"options":["A"],"correct_options":["A"]}]}`,
		"分值为负":      `{"pass_score":0,"questions":[{"id":"q","type":"single","score":-1,"options":["A"],"correct_options":["A"]}]}`,
		"分值非整数":     `{"pass_score":0,"questions":[{"id":"q","type":"single","score":1.5,"options":["A"],"correct_options":["A"]}]}`,
		"单选两个正确项":   `{"pass_score":0,"questions":[{"id":"q","type":"single","score":1,"options":["A","B"],"correct_options":["A","B"]}]}`,
		"多选零个正确项":   `{"pass_score":0,"questions":[{"id":"q","type":"multiple","score":1,"options":["A","B"],"correct_options":[]}]}`,
		"正确项不在选项中":  `{"pass_score":0,"questions":[{"id":"q","type":"single","score":1,"options":["A"],"correct_options":["B"]}]}`,
		"正确项重复":     `{"pass_score":0,"questions":[{"id":"q","type":"multiple","score":1,"options":["A","B"],"correct_options":["A","A"]}]}`,
		"选项重复":      `{"pass_score":0,"questions":[{"id":"q","type":"single","score":1,"options":["A","A"],"correct_options":["A"]}]}`,
		"选项编号为空":    `{"pass_score":0,"questions":[{"id":"q","type":"single","score":1,"options":[""],"correct_options":[""]}]}`,
		"数值题标准值缺失":  `{"pass_score":0,"questions":[{"id":"q","type":"number","score":1,"tolerance":0.1}]}`,
		"数值题容限为负":   `{"pass_score":0,"questions":[{"id":"q","type":"number","score":1,"correct_value":1,"tolerance":-0.1}]}`,
		"未知题型":      `{"pass_score":0,"questions":[{"id":"q","type":"essay","score":1}]}`,
		"及格线为负":     `{"pass_score":-1,"questions":[{"id":"q","type":"single","score":1,"options":["A"],"correct_options":["A"]}]}`,
		"及格线超过满分":   `{"pass_score":2,"questions":[{"id":"q","type":"single","score":1,"options":["A"],"correct_options":["A"]}]}`,
		"缺少及格线":     `{"questions":[{"id":"q","type":"single","score":1,"options":["A"],"correct_options":["A"]}]}`,
		"数值题容限为字符串": `{"pass_score":0,"questions":[{"id":"q","type":"number","score":1,"correct_value":1,"tolerance":"x"}]}`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadPaper([]byte(src)); err == nil {
				t.Fatalf("应当报错: %s", name)
			}
		})
	}
}

func TestLoadPaperPassScoreBoundary(t *testing.T) {
	src := strings.Replace(validPaperJSON, `"pass_score": 6`, `"pass_score": 9`, 1)
	if _, err := LoadPaper([]byte(src)); err != nil {
		t.Fatalf("及格线等于满分应合法: %v", err)
	}
}
