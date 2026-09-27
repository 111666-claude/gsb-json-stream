package jsonstream

import (
	"encoding/json"
	"fmt"
)

// UnmarshalAll 朴素参考实现：整份 json.Unmarshal 进内存，仅用于对照测试。
// 峰值内存 O(整份输入)，与流式 Reader 的 O(单元素) 形成对照。
func UnmarshalAll(text []byte) ([]Element, error) {
	var raws []json.RawMessage
	if err := json.Unmarshal(text, &raws); err != nil {
		return nil, fmt.Errorf("解析失败：%v", err)
	}
	elements := make([]Element, 0, len(raws))
	for index, raw := range raws {
		element, err := parseElement(raw, index)
		if err != nil {
			return nil, err
		}
		elements = append(elements, element)
	}
	return elements, nil
}
