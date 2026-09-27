package jsonstream

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

func collect(t *testing.T, reader *Reader) []Element {
	t.Helper()
	var elements []Element
	for {
		element, err := reader.Next()
		if err == io.EOF {
			return elements
		}
		if err != nil {
			t.Fatalf("Next 出错：%v", err)
		}
		elements = append(elements, element)
	}
}

func mustNew(t *testing.T, text string) *Reader {
	t.Helper()
	reader, err := New([]byte(text))
	if err != nil {
		t.Fatalf("New 出错：%v", err)
	}
	return reader
}

func TestReadTwoElements(t *testing.T) {
	elements := collect(t, mustNew(t, `[{"key":"a","value":1},{"key":"b","value":2}]`))
	if len(elements) != 2 {
		t.Fatalf("元素数不对：%d", len(elements))
	}
	if elements[0].Key != "a" || elements[0].Value != 1 || elements[1].Key != "b" || elements[1].Value != 2 {
		t.Fatalf("元素内容不对：%+v", elements)
	}
}

func TestEmptyArray(t *testing.T) {
	reader := mustNew(t, `[]`)
	if _, err := reader.Next(); err != io.EOF {
		t.Fatalf("空数组应该直接返回 io.EOF，得到 %v", err)
	}
}

func TestInvalidJSON(t *testing.T) {
	if _, err := New([]byte(`{`)); err == nil {
		t.Fatal("非法 JSON 应该报错")
	}
}

func TestTopLevelNotArray(t *testing.T) {
	if _, err := New([]byte(`{"key":"a","value":1}`)); err == nil {
		t.Fatal("顶层不是数组应该报错")
	}
}

func TestTruncatedJSON(t *testing.T) {
	reader := mustNew(t, `[{"key":"a","value":1},`)
	if _, err := reader.Next(); err != nil {
		t.Fatalf("第一个元素应该读成功：%v", err)
	}
	if _, err := reader.Next(); err == nil || err == io.EOF {
		t.Fatalf("截断的 JSON 应该报解析错误，得到 %v", err)
	}
}

func TestBigIntegerPreserved(t *testing.T) {
	elements := collect(t, mustNew(t, `[{"key":"a","value":9007199254740993}]`))
	if len(elements) != 1 || elements[0].Value != 9007199254740993 {
		t.Fatalf("大整数必须原样保留：%+v", elements)
	}
}

func TestIntegerOverflow(t *testing.T) {
	reader := mustNew(t, `[{"key":"a","value":9223372036854775808}]`)
	_, err := reader.Next()
	if err == nil || !strings.Contains(err.Error(), "元素 0") {
		t.Fatalf("超出 int64 应该报带下标的错误，得到 %v", err)
	}
}

func TestMissingKey(t *testing.T) {
	reader := mustNew(t, `[{"key":"a","value":1},{"value":2}]`)
	if _, err := reader.Next(); err != nil {
		t.Fatalf("第一个元素应该读成功：%v", err)
	}
	_, err := reader.Next()
	if err == nil || !strings.Contains(err.Error(), "元素 1") || !strings.Contains(err.Error(), "key") {
		t.Fatalf("缺 key 应该报带下标的错误，得到 %v", err)
	}
}

func TestMissingValue(t *testing.T) {
	reader := mustNew(t, `[{"key":"a"}]`)
	_, err := reader.Next()
	if err == nil || !strings.Contains(err.Error(), "元素 0") || !strings.Contains(err.Error(), "value") {
		t.Fatalf("缺 value 应该报带下标的错误，得到 %v", err)
	}
}

func TestValueWrongType(t *testing.T) {
	reader := mustNew(t, `[{"key":"a","value":"1"}]`)
	_, err := reader.Next()
	if err == nil || !strings.Contains(err.Error(), "元素 0") {
		t.Fatalf("value 是字符串应该报带下标的错误，得到 %v", err)
	}
}

func TestValueNotInteger(t *testing.T) {
	reader := mustNew(t, `[{"key":"a","value":1.5}]`)
	if _, err := reader.Next(); err == nil {
		t.Fatal("value 是小数应该报错")
	}
}

func TestPeakIsCounted(t *testing.T) {
	reader := mustNew(t, `[{"key":"a","value":1}]`)
	collect(t, reader)
	if reader.Peak() == 0 {
		t.Fatal("峰值内存估计应该被统计")
	}
}

func TestElementIndexes(t *testing.T) {
	elements := collect(t, mustNew(t, `[{"key":"a","value":1},{"key":"b","value":2}]`))
	for index, element := range elements {
		if element.Index != index {
			t.Fatalf("下标应该递增：第 %d 个是 %d", index, element.Index)
		}
	}
}

func TestRepeatedReadsSameResult(t *testing.T) {
	text := `[{"key":"a","value":9007199254740993},{"key":"b","value":-7}]`
	first := collect(t, mustNew(t, text))
	second := collect(t, mustNew(t, text))
	if fmt.Sprintf("%v", first) != fmt.Sprintf("%v", second) {
		t.Fatalf("同一输入重复读取结果应该相同：%v vs %v", first, second)
	}
}

func TestReadN(t *testing.T) {
	reader := mustNew(t, `[{"key":"a","value":1},{"key":"b","value":2},{"key":"c","value":3},{"key":"d","value":4},{"key":"e","value":5}]`)
	var sizes []int
	var total int
	for {
		batch, err := reader.ReadN(2)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("ReadN 出错：%v", err)
		}
		sizes = append(sizes, len(batch))
		total += len(batch)
	}
	if total != 5 || fmt.Sprint(sizes) != "[2 2 1]" {
		t.Fatalf("ReadN 分批不对：%v 共 %d", sizes, total)
	}
}

func TestEachCallback(t *testing.T) {
	reader := mustNew(t, `[{"key":"a","value":1},{"key":"b","value":2},{"key":"c","value":3}]`)
	count := 0
	var sum int64
	err := reader.Each(func(element Element) error {
		if element.Index != count {
			t.Fatalf("下标应该递增：%d", element.Index)
		}
		count++
		sum += element.Value
		return nil
	})
	if err != nil || count != 3 || sum != 6 {
		t.Fatalf("Each 遍历不对：count=%d sum=%d err=%v", count, sum, err)
	}
}

func TestEachCallbackStop(t *testing.T) {
	reader := mustNew(t, `[{"key":"a","value":1},{"key":"b","value":2}]`)
	stop := errors.New("停")
	visited := 0
	err := reader.Each(func(element Element) error {
		visited++
		return stop
	})
	if err != stop || visited != 1 {
		t.Fatalf("回调返回错误应该中断遍历：visited=%d err=%v", visited, err)
	}
}

func TestStreamingPeakSmall(t *testing.T) {
	var builder strings.Builder
	builder.WriteString("[")
	for index := 0; index < 100000; index++ {
		if index > 0 {
			builder.WriteString(",")
		}
		fmt.Fprintf(&builder, `{"key":"k%05d","value":%d}`, index, index)
	}
	builder.WriteString("]")
	text := builder.String()
	reader := mustNew(t, text)
	elements := collect(t, reader)
	if len(elements) != 100000 {
		t.Fatalf("元素数不对：%d", len(elements))
	}
	if reader.Peak() >= 1024 || reader.Peak()*1000 >= int64(len(text)) {
		t.Fatalf("峰值应该与单元素同阶而非整份输入：peak=%d bytes=%d", reader.Peak(), len(text))
	}
}
