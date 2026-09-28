package grading

import (
	"math"
	"testing"
)

func mustPaper(t *testing.T) *Paper {
	t.Helper()
	p, err := LoadPaper([]byte(validPaperJSON))
	if err != nil {
		t.Fatalf("试卷加载失败: %v", err)
	}
	return p
}

func lines(rows ...string) [][]byte {
	out := make([][]byte, len(rows))
	for i, r := range rows {
		out[i] = []byte(r)
	}
	return out
}

func findResult(results []Result, id string) *Result {
	for i := range results {
		if results[i].CandidateID == id {
			return &results[i]
		}
	}
	return nil
}

func TestGradeScoring(t *testing.T) {
	p := mustPaper(t)
	rows := lines(
		// 全对：2 + 4 + 3 = 9，及格
		`{"candidate_id":"C1","answers":{"q1":["A"],"q2":["A","C"],"q3":10}}`,
		// 单选错；多选只选一个正确项，得一半分 2；数值题边界 10.5（容限 0.5）正确
		`{"candidate_id":"C2","answers":{"q1":["B"],"q2":["A"],"q3":10.5}}`,
		// 多选含错误选项 => 0；数值题另一边界 9.5 正确
		`{"candidate_id":"C3","answers":{"q2":["A","D"],"q3":9.5}}`,
		// 数值题差 0.50001 => 零分；多选空列表 => 零分（答错，不是漏答）；单选空列表零分
		`{"candidate_id":"C4","answers":{"q1":[],"q2":[],"q3":10.50001}}`,
	)
	results, summary := GradeFile(p, rows)

	c1 := findResult(results, "C1")
	if !c1.Valid || *c1.TotalScore != 9 || !*c1.Passed {
		t.Fatalf("C1 应满分及格: %+v", c1)
	}

	c2 := findResult(results, "C2")
	if !c2.Valid {
		t.Fatalf("C2 应有效: %+v", c2.Errors)
	}
	// 0 + 2 + 3 = 5，不及格
	if math.Abs(*c2.TotalScore-5) > 1e-12 {
		t.Fatalf("C2 总分应为 5，实际 %v", *c2.TotalScore)
	}
	if *c2.Passed {
		t.Fatalf("C2 应不及格（5 < 6）")
	}
	if c2.Questions[1].Status != StatusPartial || c2.Questions[1].Score != 2 {
		t.Fatalf("C2 q2 应为部分得分 2: %+v", c2.Questions[1])
	}

	c3 := findResult(results, "C3")
	// q1 漏答 0；q2 含错误项 0；q3 边界正确 3 => 3
	if math.Abs(*c3.TotalScore-3) > 1e-12 {
		t.Fatalf("C3 总分应为 3，实际 %v", *c3.TotalScore)
	}
	if c3.Questions[0].Status != StatusUnanswered {
		t.Fatalf("C3 q1 应为漏答")
	}
	if c3.Questions[1].Status != StatusWrong {
		t.Fatalf("C3 q2 应答错")
	}

	c4 := findResult(results, "C4")
	if math.Abs(*c4.TotalScore) > 1e-12 {
		t.Fatalf("C4 总分应为 0，实际 %v", *c4.TotalScore)
	}
	for _, qr := range c4.Questions {
		if qr.Status != StatusWrong {
			t.Fatalf("C4 %s 应为 wrong，实际 %s", qr.QuestionID, qr.Status)
		}
	}

	if summary.ValidCount != 4 || summary.InvalidCount != 0 {
		t.Fatalf("应全部有效: %+v", summary)
	}
	if summary.PassCount != 1 {
		t.Fatalf("及格人数应为 1，实际 %d", summary.PassCount)
	}
	// 均分 (9+5+3+0)/4 = 4.25，基于未舍入分值
	if math.Abs(*summary.AverageScore-4.25) > 1e-12 {
		t.Fatalf("均分应为 4.25，实际 %v", *summary.AverageScore)
	}
	// 满分率: q1 只有 C1，q2 只有 C1，q3 有 C1/C2/C3 => [0.25 0.25 0.75]
	wantRates := map[string]float64{"q1": 0.25, "q2": 0.25, "q3": 0.75}
	for id, want := range wantRates {
		if math.Abs(*summary.QuestionFullRate[id]-want) > 1e-12 {
			t.Fatalf("%s 满分率应为 %v，实际 %v", id, want, *summary.QuestionFullRate[id])
		}
	}
}

func TestGradePassBoundary(t *testing.T) {
	p := mustPaper(t)
	// q1 满分 2 + q2 一半 4*1/2=2 + q3 漏答 = 4
	r, _ := GradeFile(p, lines(`{"candidate_id":"C","answers":{"q1":["A"],"q2":["C"]}}`))
	if *r[0].TotalScore != 4 || *r[0].Passed {
		t.Fatalf("4 分应不及格")
	}
	// 再答对数值题 => 7 >= 6 及格
	r, _ = GradeFile(p, lines(`{"candidate_id":"C","answers":{"q1":["A"],"q2":["C"],"q3":10}}`))
	if *r[0].TotalScore != 7 || !*r[0].Passed {
		t.Fatalf("7 分应及格")
	}
}

func TestGradeInvalidSheets(t *testing.T) {
	p := mustPaper(t)
	rows := lines(
		`{"candidate_id":"B1","answers":{"q9":["A"]}}`,       // 未知题号
		`{"candidate_id":"B2","answers":{"q1":["Z"]}}`,       // 未定义选项
		`{"candidate_id":"B3","answers":{"q2":["A","A"]}}`,   // 重复选项
		`{"candidate_id":"B4","answers":{"q1":["A","B"]}}`,   // 单选多答
		`{"candidate_id":"B5","answers":{"q3":"10"}}`,        // 答案类型错误
		`{"candidate_id":"B6","answers":{"q1":"A"}}`,         // 选择答案非列表
		`{"candidate_id":"","answers":{}}`,                   // 空考生编号
		`{"answers":{}}`,                                     // 缺考生编号
		`{"candidate_id":"B9"}`,                              // 缺 answers
		`{broken json`,                                       // 非法 JSON
		`{"candidate_id":"B10","answers":{"q3":10},"extra":`, // 非法 JSON（有编号）
		`   `, // 空行
	)
	results, summary := GradeFile(p, rows)

	if len(results) != len(rows) {
		t.Fatalf("结果数应与行数一致: %d vs %d", len(results), len(rows))
	}
	for i, r := range results {
		if r.Valid {
			t.Fatalf("第 %d 行应无效", i+1)
		}
		if r.Line != i+1 {
			t.Fatalf("行号应为 %d，实际 %d", i+1, r.Line)
		}
		if len(r.Errors) == 0 {
			t.Fatalf("第 %d 行应给出具体错误", i+1)
		}
	}
	// 非法 JSON 但含可识别编号时必须保留编号
	if results[10].CandidateID != "B10" {
		t.Fatalf("非法 JSON 行应保留可识别编号，实际 %q", results[10].CandidateID)
	}
	if summary.InvalidCount != len(rows) || summary.ValidCount != 0 {
		t.Fatalf("全部应为无效: %+v", summary)
	}
}

func TestGradeDuplicateCandidates(t *testing.T) {
	p := mustPaper(t)
	rows := lines(
		`{"candidate_id":"DUP","answers":{"q1":["A"],"q2":["A","C"],"q3":10}}`,
		`{"candidate_id":"DUP","answers":{}}`,
		`{"candidate_id":"OTHER","answers":{"q1":["A"]}}`,
	)
	results, summary := GradeFile(p, rows)
	for _, r := range results {
		if r.CandidateID == "DUP" {
			if r.Valid || r.TotalScore != nil {
				t.Fatalf("重复考生的答卷都应无效: %+v", r)
			}
		}
	}
	if findResult(results, "OTHER") != nil && !findResult(results, "OTHER").Valid {
		t.Fatalf("未重复的考生不应受牵连")
	}
	if summary.ValidCount != 1 || summary.InvalidCount != 2 {
		t.Fatalf("汇总计数错误: %+v", summary)
	}
}

func TestGradeEmptyFile(t *testing.T) {
	p := mustPaper(t)
	results, summary := GradeFile(p, nil)
	if len(results) != 0 {
		t.Fatalf("空文件不应产生结果行")
	}
	if summary.ValidCount != 0 || summary.InvalidCount != 0 {
		t.Fatalf("计数应为零")
	}
	if summary.AverageScore != nil {
		t.Fatalf("无有效答卷时均分应为 null")
	}
	if summary.PassCount != 0 {
		t.Fatalf("及格人数应为零")
	}
	for _, rate := range summary.QuestionFullRate {
		if rate != nil {
			t.Fatalf("无有效答卷时满分率应为 null")
		}
	}
}

func TestGradeUnscoredDoNotPolluteStats(t *testing.T) {
	p := mustPaper(t)
	rows := lines(
		`{"candidate_id":"OK","answers":{"q1":["A"]}}`,  // 有效，仅 q1 满分
		`{"candidate_id":"BAD","answers":{"q1":["Z"]}}`, // 无效
	)
	results, summary := GradeFile(p, rows)
	if summary.ValidCount != 1 || summary.InvalidCount != 1 {
		t.Fatalf("计数错误: %+v", summary)
	}
	// 满分率分母只含有效答卷 => q1 1/1=1, q2 0/1=0
	if *summary.QuestionFullRate["q1"] != 1 || *summary.QuestionFullRate["q2"] != 0 {
		t.Fatalf("满分率错误: %+v", summary.QuestionFullRate)
	}
	if *results[0].TotalScore != 2 {
		t.Fatalf("OK 漏答两题，总分应为 2")
	}
}
