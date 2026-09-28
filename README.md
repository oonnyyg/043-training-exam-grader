# training-exam-grader

企业培训离线阅卷工具。读取一份 JSON 试卷和批量 JSONL 答卷，生成可复核的逐题成绩（`results.jsonl`）与考试统计（`summary.json`）。仅使用 Go 标准库，不依赖任何在线服务。

## 构建与运行

```sh
go build ./cmd/examgrade
go run ./cmd/examgrade -paper examples/paper.json -answers examples/answers.jsonl -out report
```

- `-paper`：JSON 试卷路径
- `-answers`：JSONL 答卷路径（每行一条，空行跳过，行号按物理行计算）
- `-out`：输出目录，不存在时创建

运行测试：

```sh
go test ./...
```

## 试卷格式（JSON）

```json
{
  "title": "安全生产培训测验",
  "pass_score": 6,
  "questions": [
    {"id": "q1", "type": "single",   "points": 2,
     "options": [{"id": "A"}, {"id": "B"}], "correct": ["A"]},
    {"id": "q2", "type": "multiple", "points": 4,
     "options": [{"id": "A"}, {"id": "B"}, {"id": "C"}], "correct": ["A", "B"]},
    {"id": "q3", "type": "numeric",  "points": 4, "value": 36.8, "tolerance": 0.3}
  ]
}
```

校验规则（任一不满足即拒绝阅卷、非零退出，且不生成或覆盖报告）：

- 至少一道题；题号非空且唯一；每题分值为正整数。
- 选择题（`single` / `multiple`）：选项编号非空且唯一；标准答案必须是已定义选项且不重复；单选恰有一个正确选项；多选至少一个正确选项。
- 数值题（`numeric`）：`value` 为有限数值，`tolerance` 为有限、非负数值。
- `pass_score` 在 0 到试卷满分（含）之间。

## 答卷格式（JSONL）

每行一个 JSON 对象：

```json
{"student_id": "S001", "answers": {"q1": ["A"], "q2": ["A", "B"], "q3": 36.9}}
```

- `student_id` 非空字符串；`answers` 按题号关联，选择题答案为选项编号列表，数值题答案为单个数值。
- 漏答（缺键或 `null`）记零分，状态为 `unanswered`；选择题提交空列表也按漏答记零分。

## 评分规则

- 单选：选中唯一正确项得满分，否则零分。
- 多选：包含任何错误选项得零分；否则按「选中正确项数 / 全部正确项数 × 本题分值」给分，全部选对为 `correct`，部分正确为 `partial`。
- 数值题：答案为有限数值，且 `|答案 - 标准值| <= tolerance`（边界包含）得满分，否则零分。
- 总分等于各题未舍入得分之和；`总分 >= pass_score`（含等于）判定及格。统计同样基于未舍入分值。

## 无效答卷（不把非法输入当作答错）

以下情况使整条答卷无效，结果中给出具体错误，且不输出逐题得分：

- 非法 JSON 行（单独报错，其他行继续处理）；缺失或空的 `student_id`
- 未知题号；引用未定义选项；选项重复；单选提交多个选项
- 答案类型错误（如选择题不是列表、数值题不是数值、数值为 NaN/Infinity）
- 重复考生编号：涉及的所有答卷均无效（不是先到先得）

## 输出

`results.jsonl` 按答卷行顺序逐行输出，包含 `line`、`student_id`、`valid`、`errors`，有效答卷额外含按试卷顺序排列的 `questions`（题号、状态、满分、得分）、`total` 与 `passed`。无法解析考生编号时该字段为空字符串但行号保留。

`summary.json` 包含：

- `total_sheets` / `valid_sheets` / `invalid_sheets`
- `mean_score`：有效答卷均分；无有效答卷时为 `null`
- `pass_count`：及格人数；无有效答卷时为 0
- `question_stats`：各题满分人数与 `full_rate`（满分率），漏答计入分母，分母只含有效答卷；无有效答卷时为 `null`

`examples/report/` 是对 `examples/answers.jsonl` 的样例输出，涵盖部分得分、容限边界、漏答、重复考生、未知题号、非法 JSON 等情况。
