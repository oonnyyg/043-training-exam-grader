package main

import (
	"flag"
	"fmt"
	"os"

	"example.com/training-exam-grader/grading"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("examgrade", flag.ContinueOnError)
	paperPath := fs.String("paper", "", "试卷 JSON 文件路径")
	answersPath := fs.String("answers", "", "答卷 JSONL 文件路径")
	outDir := fs.String("out", "", "报告输出目录")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "用法: examgrade -paper 试卷.json -answers 答卷.jsonl -out 报告目录\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *paperPath == "" || *answersPath == "" || *outDir == "" {
		fs.Usage()
		return 2
	}

	if _, _, err := grading.Run(*paperPath, *answersPath, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, "阅卷失败:", err)
		return 1
	}
	fmt.Println("阅卷完成，报告已写入", *outDir)
	return 0
}
