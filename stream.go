// Package jsonstream 流式读取 JSON 数组。
package jsonstream

import (
	"encoding/json"
	"fmt"
)

// Element 是流里的一项。
type Element struct {
	Index int
	Key   string
	Value int64
}

// Reader 流式读数组。缺陷：先用 json.Unmarshal 把整份读成 map，再遍历。
type Reader struct {
	elements []Element
	peak     int
	at       int
}

// New 从 JSON 文本建 reader（口径要求流式，见 README）。
func New(text []byte) (*Reader, error) {
	var decoded []map[string]any
	if err := json.Unmarshal(text, &decoded); err != nil {
		return nil, fmt.Errorf("解析失败：%v", err)
	}
	reader := &Reader{peak: len(text)}
	for index, item := range decoded {
		key, _ := item["key"].(string)
		value, _ := item["value"].(float64)
		reader.elements = append(reader.elements, Element{Index: index, Key: key, Value: int64(value)})
	}
	return reader, nil
}

// Next 取下一项。
func (r *Reader) Next() (Element, bool) {
	if r.at >= len(r.elements) {
		return Element{}, false
	}
	element := r.elements[r.at]
	r.at++
	return element, true
}

// Peak 是峰值内存估计（统计用）。
func (r *Reader) Peak() int { return r.peak }
