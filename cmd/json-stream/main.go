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

// countingReader 统计流过的字节数。
type countingReader struct {
	source io.Reader
	total  int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.source.Read(p)
	c.total += int64(n)
	return n, err
}

// generateArray 流式生成 N 个元素的 JSON 数组，不整份进内存。
func generateArray(count int) io.Reader {
	reader, writer := io.Pipe()
	go func() {
		buffered := bufio.NewWriterSize(writer, 64*1024)
		buffered.WriteString("[")
		for index := 0; index < count; index++ {
			if index > 0 {
				buffered.WriteString(",")
			}
			fmt.Fprintf(buffered, "{\"key\":\"k%05d\",\"value\":%d}", index, index)
		}
		buffered.WriteString("]")
		buffered.Flush()
		writer.Close()
	}()
	return reader
}

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
	var source io.Reader
	switch {
	case *text != "":
		source = strings.NewReader(*text)
	case *path != "":
		file, err := os.Open(*path)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		defer file.Close()
		source = file
	case *generate > 0:
		source = generateArray(*generate)
	default:
		fmt.Fprintln(stderr, "用法：json-stream --file x.json | --text '[...]' | --generate 100000")
		return 2
	}
	counter := &countingReader{source: source}
	reader, err := jsonstream.NewReader(counter)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	count := 0
	last := jsonstream.Element{}
	err = reader.Each(func(element jsonstream.Element) error {
		count++
		last = element
		return nil
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "elements=%d last=%s:%d\n", count, last.Key, last.Value)
	if *counters {
		fmt.Fprintf(stdout, "peak=%d bytes=%d\n", reader.Peak(), counter.total)
	}
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
