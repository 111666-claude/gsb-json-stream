// Package jsonstream 流式读取 JSON 数组。
package jsonstream

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

// Element 是流里的一项。
type Element struct {
	Index int
	Key   string
	Value int64
}

// Reader 用 json.Decoder 逐元素流式读数组，峰值内存 O(单元素)。
type Reader struct {
	dec   *json.Decoder
	index int
	peak  int64
	done  bool
}

// New 从内存中的 JSON 文本建 reader。
func New(text []byte) (*Reader, error) {
	return NewReader(bytes.NewReader(text))
}

// NewReader 从 io.Reader 建流式 reader，并预先校验顶层是数组。
func NewReader(source io.Reader) (*Reader, error) {
	dec := json.NewDecoder(bufio.NewReaderSize(source, 64*1024))
	token, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("读取数组开头失败：%w", err)
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '[' {
		return nil, fmt.Errorf("顶层必须是 JSON 数组，实际是 %v", token)
	}
	return &Reader{dec: dec}, nil
}

// Next 流式取下一项；读完返回 io.EOF，解析或校验失败返回带元素下标的错误。
func (r *Reader) Next() (Element, error) {
	if r.done {
		return Element{}, io.EOF
	}
	if !r.dec.More() {
		if _, err := r.dec.Token(); err != nil {
			return Element{}, fmt.Errorf("读取数组结尾失败：%w", err)
		}
		if token, err := r.dec.Token(); err != io.EOF {
			if err == nil {
				return Element{}, fmt.Errorf("数组结束后有多余内容：%v", token)
			}
			return Element{}, fmt.Errorf("数组结束后有多余内容：%w", err)
		}
		r.done = true
		return Element{}, io.EOF
	}
	index := r.index
	start := r.dec.InputOffset()
	var raw struct {
		Key   *string         `json:"key"`
		Value json.RawMessage `json:"value"`
	}
	if err := r.dec.Decode(&raw); err != nil {
		return Element{}, fmt.Errorf("元素 %d 解析失败：%w", index, err)
	}
	if size := r.dec.InputOffset() - start; size > r.peak {
		r.peak = size
	}
	if raw.Key == nil {
		return Element{}, fmt.Errorf("元素 %d 缺少 key 字段", index)
	}
	if raw.Value == nil {
		return Element{}, fmt.Errorf("元素 %d 缺少 value 字段", index)
	}
	if raw.Value[0] == '"' {
		return Element{}, fmt.Errorf("元素 %d 的 value 类型不对：应该是整数，实际是字符串", index)
	}
	value, err := strconv.ParseInt(string(raw.Value), 10, 64)
	if err != nil {
		return Element{}, fmt.Errorf("元素 %d 的 value 不是 int64 整数：%w", index, err)
	}
	r.index++
	return Element{Index: index, Key: *raw.Key, Value: value}, nil
}

// ReadN 一次最多取 n 项；流结束时返回已读到的项，一项都没有时返回 io.EOF。
func (r *Reader) ReadN(n int) ([]Element, error) {
	if n <= 0 {
		return nil, nil
	}
	elements := make([]Element, 0, n)
	for len(elements) < n {
		element, err := r.Next()
		if err == io.EOF {
			if len(elements) == 0 {
				return nil, io.EOF
			}
			return elements, nil
		}
		if err != nil {
			return elements, err
		}
		elements = append(elements, element)
	}
	return elements, nil
}

// Each 回调式遍历，调用方不需要把结果全存下来；回调返回错误会中断遍历。
func (r *Reader) Each(fn func(Element) error) error {
	for {
		element, err := r.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := fn(element); err != nil {
			return err
		}
	}
}

// Peak 是峰值内存估计（迄今最大单元素的字节数，统计用）。
func (r *Reader) Peak() int64 { return r.peak }
