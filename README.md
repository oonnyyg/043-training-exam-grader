# 企业培训离线阅卷工具（examgrade）

把同一份试卷的批量答卷（JSONL）转换为可复核的逐题成绩（`results.jsonl`）与考试统计（`summary.json`）。
仅使用 Go 标准库，完全离线运行，不依赖任何在线服务。

## 构建与运行

```powershell
go build ./cmd/examgrade
go run ./cmd/examgrade -paper examples/paper.json -answers examples/answers.jsonl -out examples/report
```

命令行参数：

- `-paper`：试卷 JSON 文件路径
- `-answers`：答卷 JSONL 文件路径（每行一条答卷；空文件合法，表示无人作答）
- `-out`：报告输出目录（不存在会自动创建）

试卷非法、文件缺失等情况会打印错误并以非零码退出，**不会生成或覆盖**任何报告。

## 试卷格式（JSON）

```json
{
  "pass_score": 6,
  "questions": [
    {"id": "q1", "type": "single",   "score": 2, "options": ["A","B","C"], "correct_options": ["A"]},
    {"id": "q2", "type": "multiple", "score": 4, "options": ["A","B","C","D"], "correct_options": ["A","C"]},
    {"id": "q3", "type": "number",   "score": 3, "correct_value": 10, "tolerance": 0.5}
  ]
}
```

校验规则：

- 至少一道题；题号非空且在卷内唯一。
- 每题 `score` 为正整数。
- 选择题（`single`/`multiple`）：选项编号非空且唯一；`correct_options` 必须全部属于选项且不重复；
  单选恰有 1 个正确选项，多选至少 1 个。
- 数值题（`number`）：`correct_value` 为有限数值，`tolerance` 为有限非负数。
- `pass_score` 在 `0` 到试卷满分之间（含端点）。

## 答卷格式（JSONL）

每行一条 JSON 答卷：

```json
{"candidate_id":"E001","answers":{"q1":["A"],"q2":["A","C"],"q3":10}}
```

- `candidate_id`：非空考生编号。
- `answers`：按题号关联的答案。选择题答案为选项编号字符串列表；数值题答案为单个数值。

### 评分规则

- 单选：选中唯一正确项得满分，否则零分（空列表也算答错）。
- 多选：包含任何错误选项即零分；否则按“选中正确项数 / 全部正确项数”比例给分
  （可能得到小数分，统计时不做取整）。
- 数值题：`|答案 - 标准值| <= tolerance` 得满分（**含边界**），否则零分。
- 漏答（`answers` 中没有该题号）记零分，状态为 `unanswered`，并参与统计。
- 总分 >= `pass_score` 即为及格（**含等于**）。

### 整条答卷无效（记为“无法评分”，不算答错）

- 非法 JSON 行、空行；缺少或类型错误的 `candidate_id` / `answers`。
- 答案引用未知题号、未定义选项、重复选项。
- 单选题提交多个选项；答案类型错误（如选择题给字符串、数值题给字符串）。
- 同一考生编号出现多次时，**涉及的所有答卷均无效**（不采用先到先得）。

非法行不会中断处理，其他行继续阅卷。

## 输出

`results.jsonl`：与输入行一一对应、保持行序，每行包含行号、考生编号（能识别时保留）、
`valid` 状态与具体 `errors`；有效答卷另含按试卷顺序排列的逐题得分、`total_score` 和 `passed`。

`summary.json`：

- `valid_count` / `invalid_count`：有效与无效答卷数。
- `average_score`：有效答卷均分；没有有效答卷时为 `null`。
- `pass_count`：及格人数；没有有效答卷时为 `0`。
- `question_full_rate`：各题满分率（得分等于该题满分的人数 / 有效答卷数，漏答计入分母）；
  没有有效答卷时各题均为 `null`。

总分、及格判断与全部统计都基于未舍入的分值计算。

## 测试

```powershell
go test ./...
```

测试覆盖：试卷各类非法配置、单选/多选/数值题评分、多选部分得分、容限边界、
漏答与空选项列表、各类无效答卷隔离、重复考生编号、非法 JSON 行、空答卷文件，
以及报告写出与“非法试卷不覆盖已有报告”的端到端场景。
