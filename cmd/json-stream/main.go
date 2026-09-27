// Command json-stream 流式读一份 JSON 数组。
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"example.com/jsonstream"
)

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("json-stream", flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("file", "", "JSON 文件")
	text := flags.String("text", "", "直接给 JSON 文本")
	generate := flags.Int("generate", 0, "生成这么大的数组（内存统计用）")
	counters := flags.Bool("counters", false, "打印峰值内存估计")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	var raw []byte
	switch {
	case *text != "":
		raw = []byte(*text)
	case *path != "":
		content, err := os.ReadFile(*path)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		raw = content
	case *generate > 0:
		var builder strings.Builder
		builder.WriteString("[")
		for index := 0; index < *generate; index++ {
			if index > 0 {
				builder.WriteString(",")
			}
			builder.WriteString(fmt.Sprintf("{\"key\":\"k%05d\",\"value\":%d}", index, index))
		}
		builder.WriteString("]")
		raw = []byte(builder.String())
	default:
		fmt.Fprintln(stderr, "用法：json-stream --file x.json | --text '[...]' | --generate 100000")
		return 2
	}
	reader, err := jsonstream.New(raw)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	count := 0
	last := jsonstream.Element{}
	for {
		element, ok := reader.Next()
		if !ok {
			break
		}
		count++
		last = element
	}
	fmt.Fprintf(stdout, "elements=%d last=%s:%d\n", count, last.Key, last.Value)
	if *counters {
		fmt.Fprintf(stdout, "peak=%d bytes=%d\n", reader.Peak(), len(raw))
	}
	_ = bufio.NewScanner
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
