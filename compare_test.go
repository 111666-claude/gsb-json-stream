package jsonstream

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// TestStreamingMatchesReference 流式结果与整份 Unmarshal 参考实现逐元素比对。
func TestStreamingMatchesReference(t *testing.T) {
	samples := []string{
		`[]`,
		`[{"key":"a","value":9007199254740993}]`,
		`[{"key":"a","value":1},{"key":"b","value":-2},{"key":"c","value":0}]`,
	}
	random := rand.New(rand.NewSource(42))
	for i := 0; i < 1000; i++ {
		var builder strings.Builder
		builder.WriteString("[")
		length := random.Intn(20)
		for index := 0; index < length; index++ {
			if index > 0 {
				builder.WriteString(",")
			}
			fmt.Fprintf(&builder, `{"key":"k%d","value":%d}`, index, random.Int63())
		}
		builder.WriteString("]")
		samples = append(samples, builder.String())
	}
	for _, sample := range samples {
		want, err := UnmarshalAll([]byte(sample))
		if err != nil {
			t.Fatalf("参考实现出错：%v", err)
		}
		got := readAll(t, sample)
		if len(got) != len(want) {
			t.Fatalf("%s：元素数 %d != %d", sample, len(got), len(want))
		}
		for index := range want {
			if got[index] != want[index] {
				t.Fatalf("%s：第 %d 项 %+v != %+v", sample, index, got[index], want[index])
			}
		}
	}
}

// TestPeakMagnitude 流式峰值是单元素量级，参考实现峰值是整份输入量级。
func TestPeakMagnitude(t *testing.T) {
	var builder strings.Builder
	builder.WriteString("[")
	for index := 0; index < 20000; index++ {
		if index > 0 {
			builder.WriteString(",")
		}
		fmt.Fprintf(&builder, `{"key":"k%05d","value":%d}`, index, index)
	}
	builder.WriteString("]")
	text := builder.String()

	reader, err := NewBytes([]byte(text))
	if err != nil {
		t.Fatalf("New 出错：%v", err)
	}
	count := 0
	if err := reader.ForEach(func(Element) error { count++; return nil }); err != nil {
		t.Fatalf("读取出错：%v", err)
	}
	if count != 20000 {
		t.Fatalf("元素数不对：%d", count)
	}
	streamingPeak := reader.Peak()
	referencePeak := len(text) // 整份 Unmarshal 的峰值量级
	if streamingPeak >= referencePeak/100 {
		t.Fatalf("流式峰值 %d 应远小于整份输入 %d", streamingPeak, referencePeak)
	}
}
