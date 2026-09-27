// Package jsonstream 流式读取 JSON 数组。
package jsonstream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Element 是流里的一项。
type Element struct {
	Index int
	Key   string
	Value int64
}

// rawElement 是单个元素的中间表示；value 用 json.Number 保住整数精度。
type rawElement struct {
	Key   *string      `json:"key"`
	Value *json.Number `json:"value"`
}

// Reader 流式读数组：json.Decoder 逐元素解码，峰值内存 O(单元素)。
type Reader struct {
	decoder *json.Decoder
	next    int
	peak    int
	done    bool
	err     error
}

// New 从 io.Reader 建流式 reader，并校验顶层是数组。
func New(source io.Reader) (*Reader, error) {
	decoder := json.NewDecoder(source)
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("解析失败：%v", err)
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '[' {
		return nil, fmt.Errorf("解析失败：顶层必须是 JSON 数组")
	}
	return &Reader{decoder: decoder}, nil
}

// NewBytes 从内存中的 JSON 文本建 reader。
func NewBytes(text []byte) (*Reader, error) {
	return New(bytes.NewReader(text))
}

// Next 取下一项；ok 为 false 时用 Err 区分正常结束与解析错误。
func (r *Reader) Next() (Element, bool) {
	if r.done || r.err != nil {
		return Element{}, false
	}
	if !r.decoder.More() {
		r.done = true
		if _, err := r.decoder.Token(); err != nil {
			r.err = fmt.Errorf("解析失败：%v", err)
			return Element{}, false
		}
		if _, err := r.decoder.Token(); err != io.EOF {
			if err == nil {
				r.err = fmt.Errorf("解析失败：数组结束后还有多余内容")
			} else {
				r.err = fmt.Errorf("解析失败：%v", err)
			}
		}
		return Element{}, false
	}
	var raw json.RawMessage
	if err := r.decoder.Decode(&raw); err != nil {
		r.err = fmt.Errorf("元素 %d：%v", r.next, err)
		return Element{}, false
	}
	if len(raw) > r.peak {
		r.peak = len(raw)
	}
	element, err := parseElement(raw, r.next)
	if err != nil {
		r.err = err
		return Element{}, false
	}
	r.next++
	return element, true
}

// Err 返回迭代中遇到的错误（正常结束为 nil）。
func (r *Reader) Err() error { return r.err }

// Peak 是峰值内存估计：迄今最大单个元素的字节数。
func (r *Reader) Peak() int { return r.peak }

// NextN 一次取最多 n 项；剩余不足 n 项时返回全部剩余项。
func (r *Reader) NextN(n int) ([]Element, error) {
	elements := make([]Element, 0, n)
	for len(elements) < n {
		element, ok := r.Next()
		if !ok {
			return elements, r.Err()
		}
		elements = append(elements, element)
	}
	return elements, nil
}

// ForEach 回调式逐元素处理，调用方无需保存全部结果。
func (r *Reader) ForEach(fn func(Element) error) error {
	for {
		element, ok := r.Next()
		if !ok {
			return r.Err()
		}
		if err := fn(element); err != nil {
			return err
		}
	}
}

// parseElement 校验并转换单个元素，错误信息带元素下标。
func parseElement(raw json.RawMessage, index int) (Element, error) {
	var item rawElement
	if err := json.Unmarshal(raw, &item); err != nil {
		return Element{}, fmt.Errorf("元素 %d：%v", index, err)
	}
	if item.Key == nil {
		return Element{}, fmt.Errorf("元素 %d：缺 key 字段", index)
	}
	if item.Value == nil {
		return Element{}, fmt.Errorf("元素 %d：缺 value 字段", index)
	}
	value, err := item.Value.Int64()
	if err != nil {
		return Element{}, fmt.Errorf("元素 %d：value 不是 int64 整数：%v", index, err)
	}
	return Element{Index: index, Key: *item.Key, Value: value}, nil
}
