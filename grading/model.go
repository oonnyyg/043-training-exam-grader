package grading

// QuestionType 表示试题类型。
type QuestionType string

const (
	// Single 单选题。
	Single QuestionType = "single"
	// Multiple 多选题。
	Multiple QuestionType = "multiple"
	// Number 数值题。
	Number QuestionType = "number"
)

// Question 是经过校验的一道试题。
type Question struct {
	ID             string
	Type           QuestionType
	Score          float64
	Options        []string // 选择题的选项编号
	CorrectOptions []string // 选择题的正确选项编号
	CorrectValue   float64  // 数值题标准值
	Tolerance      float64  // 数值题绝对误差容限
}

// Paper 是经过校验的试卷。
type Paper struct {
	PassScore float64
	Questions []*Question
	index     map[string]*Question
}

// Question 返回指定题号的试题，不存在时返回 nil。
func (p *Paper) Question(id string) *Question {
	return p.index[id]
}

// TotalScore 返回试卷满分。
func (p *Paper) TotalScore() float64 {
	total := 0.0
	for _, q := range p.Questions {
		total += q.Score
	}
	return total
}

// 逐题作答状态。
const (
	StatusCorrect    = "correct"    // 得满分
	StatusPartial    = "partial"    // 多选部分得分
	StatusWrong      = "wrong"      // 答错（含选择题空选项列表）
	StatusUnanswered = "unanswered" // 漏答
)

// QuestionResult 是有效答卷的逐题结果。
type QuestionResult struct {
	QuestionID string  `json:"question_id"`
	Score      float64 `json:"score"`
	Status     string  `json:"status"`
}

// Result 是一行答卷的阅卷结果。
type Result struct {
	Line        int              `json:"line"`
	CandidateID string           `json:"candidate_id,omitempty"`
	Valid       bool             `json:"valid"`
	Errors      []string         `json:"errors,omitempty"`
	Questions   []QuestionResult `json:"questions,omitempty"`
	TotalScore  *float64         `json:"total_score,omitempty"`
	Passed      *bool            `json:"passed,omitempty"`
}

// Summary 是全部答卷的汇总统计。
type Summary struct {
	ValidCount       int                 `json:"valid_count"`
	InvalidCount     int                 `json:"invalid_count"`
	AverageScore     *float64            `json:"average_score"`
	PassCount        int                 `json:"pass_count"`
	QuestionFullRate map[string]*float64 `json:"question_full_rate"`
}
