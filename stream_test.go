package jsonstream

import (
	"strings"
	"testing"
)

func readAll(t *testing.T, text string) []Element {
	t.Helper()
	reader, err := NewBytes([]byte(text))
	if err != nil {
		t.Fatalf("New 出错：%v", err)
	}
	var elements []Element
	for {
		element, ok := reader.Next()
		if !ok {
			break
		}
		elements = append(elements, element)
	}
	if err := reader.Err(); err != nil {
		t.Fatalf("读取出错：%v", err)
	}
	return elements
}

func expectErr(t *testing.T, text string, want string) {
	t.Helper()
	reader, err := NewBytes([]byte(text))
	if err != nil {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("错误 %q 不含 %q", err, want)
		}
		return
	}
	for {
		if _, ok := reader.Next(); !ok {
			break
		}
	}
	if reader.Err() == nil {
		t.Fatalf("%s 应该报错", text)
	}
	if !strings.Contains(reader.Err().Error(), want) {
		t.Fatalf("错误 %q 不含 %q", reader.Err(), want)
	}
}

func TestReadTwoElements(t *testing.T) {
	elements := readAll(t, `[{"key":"a","value":1},{"key":"b","value":2}]`)
	if len(elements) != 2 {
		t.Fatalf("元素数不对：%d", len(elements))
	}
	if elements[0].Key != "a" || elements[0].Value != 1 || elements[1].Key != "b" || elements[1].Value != 2 {
		t.Fatalf("内容不对：%+v", elements)
	}
}

func TestEmptyArray(t *testing.T) {
	reader, err := NewBytes([]byte(`[]`))
	if err != nil {
		t.Fatalf("New 出错：%v", err)
	}
	if _, ok := reader.Next(); ok {
		t.Fatal("空数组不该有元素")
	}
	if err := reader.Err(); err != nil {
		t.Fatalf("空数组不该报错：%v", err)
	}
}

func TestInvalidJSON(t *testing.T) {
	if _, err := NewBytes([]byte(`{`)); err == nil {
		t.Fatal("非法 JSON 应该报错")
	}
	expectErr(t, `[{"key":"a","value":1},`, "元素 1")
	expectErr(t, `[{"key":"a","value":1}] 垃圾`, "解析失败")
	expectErr(t, `[{"key":"a","value":1}] 2`, "多余内容")
}

func TestTopLevelMustBeArray(t *testing.T) {
	if _, err := NewBytes([]byte(`{"key":"a","value":1}`)); err == nil {
		t.Fatal("顶层不是数组应该报错")
	}
}

func TestBigIntegerPreserved(t *testing.T) {
	elements := readAll(t, `[{"key":"a","value":9007199254740993}]`)
	if elements[0].Value != 9007199254740993 {
		t.Fatalf("大整数被截断：%d", elements[0].Value)
	}
}

func TestInt64Overflow(t *testing.T) {
	expectErr(t, `[{"key":"a","value":9223372036854775808}]`, "元素 0")
}

func TestMissingFields(t *testing.T) {
	expectErr(t, `[{"key":"a","value":1},{"value":2}]`, "元素 1")
	expectErr(t, `[{"key":"a","value":1},{"key":"b"}]`, "元素 1")
}

func TestWrongTypes(t *testing.T) {
	expectErr(t, `[{"key":"a","value":"x"}]`, "元素 0")
	expectErr(t, `[{"key":"a","value":1.5}]`, "元素 0")
	expectErr(t, `[1]`, "元素 0")
}

func TestPeakIsCounted(t *testing.T) {
	reader, err := NewBytes([]byte(`[{"key":"a","value":1}]`))
	if err != nil {
		t.Fatalf("New 出错：%v", err)
	}
	if _, ok := reader.Next(); !ok {
		t.Fatalf("读取出错：%v", reader.Err())
	}
	if reader.Peak() == 0 {
		t.Fatal("峰值内存估计应该被统计")
	}
	if reader.Peak() >= len(`[{"key":"a","value":1}]`) {
		t.Fatalf("流式峰值应是单元素量级，不是整份输入：%d", reader.Peak())
	}
}

func TestElementIndexes(t *testing.T) {
	elements := readAll(t, `[{"key":"a","value":1},{"key":"b","value":2}]`)
	if elements[0].Index != 0 || elements[1].Index != 1 {
		t.Fatalf("下标不对：%d %d", elements[0].Index, elements[1].Index)
	}
}

func TestDeterministic(t *testing.T) {
	text := `[{"key":"a","value":1},{"key":"b","value":9007199254740993}]`
	first := readAll(t, text)
	second := readAll(t, text)
	if len(first) != len(second) {
		t.Fatal("重复读取结果不一致")
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("第 %d 项不一致：%+v != %+v", i, first[i], second[i])
		}
	}
}

func TestNextN(t *testing.T) {
	reader, err := NewBytes([]byte(`[{"key":"a","value":1},{"key":"b","value":2},{"key":"c","value":3}]`))
	if err != nil {
		t.Fatalf("New 出错：%v", err)
	}
	batch, err := reader.NextN(2)
	if err != nil || len(batch) != 2 || batch[1].Key != "b" {
		t.Fatalf("NextN(2) 不对：%+v %v", batch, err)
	}
	rest, err := reader.NextN(2)
	if err != nil || len(rest) != 1 || rest[0].Key != "c" {
		t.Fatalf("NextN 剩余不对：%+v %v", rest, err)
	}
}

func TestForEach(t *testing.T) {
	reader, err := NewBytes([]byte(`[{"key":"a","value":1},{"key":"b","value":2}]`))
	if err != nil {
		t.Fatalf("New 出错：%v", err)
	}
	count := 0
	var sum int64
	err = reader.ForEach(func(element Element) error {
		if element.Index != count {
			t.Fatalf("下标不递增：%d", element.Index)
		}
		count++
		sum += element.Value
		return nil
	})
	if err != nil {
		t.Fatalf("ForEach 出错：%v", err)
	}
	if count != 2 || sum != 3 {
		t.Fatalf("ForEach 结果不对：%d %d", count, sum)
	}
}
