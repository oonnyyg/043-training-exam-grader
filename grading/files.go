package grading

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ReadPaperFile 从文件读取并校验试卷。
func ReadPaperFile(path string) (*Paper, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取试卷失败: %w", err)
	}
	return LoadPaper(data)
}

// ReadAnswerLines 读取 JSONL 答卷，按行返回原始内容。
// 文件不存在时返回错误；空文件返回空切片（不是错误）。
func ReadAnswerLines(path string) ([][]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("读取答卷失败: %w", err)
	}
	defer f.Close()

	var lines [][]byte
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		lines = append(lines, bytes.Clone(scanner.Bytes()))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取答卷失败: %w", err)
	}
	return lines, nil
}

// Run 执行完整的离线阅卷流程并写出报告。
// 仅在全部处理成功后才写入输出目录，试卷非法等情况下不会生成或覆盖报告。
func Run(paperPath, answersPath, outDir string) ([]Result, Summary, error) {
	paper, err := ReadPaperFile(paperPath)
	if err != nil {
		return nil, Summary{}, err
	}
	lines, err := ReadAnswerLines(answersPath)
	if err != nil {
		return nil, Summary{}, err
	}

	results, summary := GradeFile(paper, lines)
	if err := WriteReports(outDir, results, summary); err != nil {
		return nil, Summary{}, err
	}
	return results, summary, nil
}

// WriteReports 将逐条结果与汇总写入输出目录。
func WriteReports(outDir string, results []Result, summary Summary) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	var resultBuf bytes.Buffer
	enc := json.NewEncoder(&resultBuf)
	enc.SetEscapeHTML(false)
	for _, r := range results {
		if err := enc.Encode(r); err != nil {
			return fmt.Errorf("序列化结果失败: %w", err)
		}
	}

	summaryBytes, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化汇总失败: %w", err)
	}
	summaryBytes = append(summaryBytes, '\n')

	// 先写临时文件再重命名，避免半成品报告覆盖已有结果。
	if err := writeAtomic(filepath.Join(outDir, "results.jsonl"), resultBuf.Bytes()); err != nil {
		return err
	}
	if err := writeAtomic(filepath.Join(outDir, "summary.json"), summaryBytes); err != nil {
		return err
	}
	return nil
}

func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("写入 %s 失败: %w", path, err)
	}
	return nil
}
