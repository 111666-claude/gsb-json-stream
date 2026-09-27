package jsonstream

import "testing"

func TestReadTwoElements(t *testing.T) {
reader, err := New([]byte(`[{"key":"a","value":1},{"key":"b","value":2}]`))
if err != nil {
t.Fatalf("New 出错：%v", err)
}
count := 0
for {
if _, ok := reader.Next(); !ok {
break
}
count++
}
if count != 2 {
t.Fatalf("元素数不对：%d", count)
}
}

func TestEmptyArray(t *testing.T) {
reader, err := New([]byte(`[]`))
if err != nil {
t.Fatalf("New 出错：%v", err)
}
if _, ok := reader.Next(); ok {
t.Fatal("空数组不该有元素")
}
}

func TestInvalidJSON(t *testing.T) {
if _, err := New([]byte(`{`)); err == nil {
t.Fatal("非法 JSON 应该报错")
}
}

func TestPeakIsCounted(t *testing.T) {
reader, err := New([]byte(`[{"key":"a","value":1}]`))
if err != nil {
t.Fatalf("New 出错：%v", err)
}
if reader.Peak() == 0 {
t.Fatal("峰值内存估计应该被统计")
}
}

func TestElementIndexes(t *testing.T) {
reader, err := New([]byte(`[{"key":"a","value":1},{"key":"b","value":2}]`))
if err != nil {
t.Fatalf("New 出错：%v", err)
}
first, _ := reader.Next()
second, _ := reader.Next()
if first.Index != 0 || second.Index != 1 {
t.Fatalf("下标不对：%d %d", first.Index, second.Index)
}
}
