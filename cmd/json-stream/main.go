// Command json-stream 流式读一份 JSON 数组。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"example.com/jsonstream"
)

// countingReader 统计流过的字节数（counters 用）。
type countingReader struct {
	source io.Reader
	bytes  int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.source.Read(p)
	c.bytes += n
	return n, err
}

// generator 边读边生成 N 个元素的 JSON 数组，峰值内存 O(单元素)。
type generator struct {
	total int
	pos   int // -1 未开始；0..total-1 下一个元素；total 收尾
	chunk []byte
}

func newGenerator(total int) *generator { return &generator{total: total, pos: -1} }

func (g *generator) Read(p []byte) (int, error) {
	for len(g.chunk) == 0 {
		switch {
		case g.pos < 0:
			g.chunk = []byte("[")
			g.pos = 0
		case g.pos < g.total:
			prefix := ","
			if g.pos == 0 {
				prefix = ""
			}
			g.chunk = []byte(fmt.Sprintf("%s{\"key\":\"k%05d\",\"value\":%d}", prefix, g.pos, g.pos))
			g.pos++
		case g.pos == g.total:
			g.chunk = []byte("]")
			g.pos++
		default:
			return 0, io.EOF
		}
	}
	n := copy(p, g.chunk)
	g.chunk = g.chunk[n:]
	return n, nil
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
		source = newGenerator(*generate)
	default:
		fmt.Fprintln(stderr, "用法：json-stream --file x.json | --text '[...]' | --generate 100000")
		return 2
	}
	counter := &countingReader{source: source}
	reader, err := jsonstream.New(counter)
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
	if err := reader.Err(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "elements=%d last=%s:%d\n", count, last.Key, last.Value)
	if *counters {
		fmt.Fprintf(stdout, "peak=%d bytes=%d\n", reader.Peak(), counter.bytes)
	}
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
