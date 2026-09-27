package jsonstream

import (
	"encoding/json"
	"io"
	"math/rand"
	"testing"
)

// referenceDecode 是「整份 Unmarshal」的朴素参考实现，用来对照流式实现。
func referenceDecode(text []byte) ([]Element, error) {
	var raw []struct {
		Key   string      `json:"key"`
		Value json.Number `json:"value"`
	}
	if err := json.Unmarshal(text, &raw); err != nil {
		return nil, err
	}
	elements := make([]Element, 0, len(raw))
	for index, item := range raw {
		value, err := item.Value.Int64()
		if err != nil {
			return nil, err
		}
		elements = append(elements, Element{Index: index, Key: item.Key, Value: value})
	}
	return elements, nil
}

func streamDecode(text []byte) ([]Element, int64, error) {
	reader, err := New(text)
	if err != nil {
		return nil, 0, err
	}
	var elements []Element
	for {
		element, err := reader.Next()
		if err == io.EOF {
			return elements, reader.Peak(), nil
		}
		if err != nil {
			return nil, 0, err
		}
		elements = append(elements, element)
	}
}

func randomValue(rng *rand.Rand) int64 {
	switch rng.Intn(4) {
	case 0:
		return int64(rng.Intn(100))
	case 1:
		return (1 << 53) + rng.Int63n(1<<10) // float64 无法精确表示的区间
	case 2:
		return -rng.Int63()
	default:
		return rng.Int63()
	}
}

func TestMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for round := 0; round < 1000; round++ {
		length := rng.Intn(41)
		items := make([]map[string]any, 0, length)
		for index := 0; index < length; index++ {
			key := make([]byte, rng.Intn(9))
			for i := range key {
				key[i] = byte('a' + rng.Intn(26))
			}
			items = append(items, map[string]any{"key": string(key), "value": randomValue(rng)})
		}
		text, err := json.Marshal(items)
		if err != nil {
			t.Fatalf("第 %d 份：生成失败：%v", round, err)
		}
		want, err := referenceDecode(text)
		if err != nil {
			t.Fatalf("第 %d 份：参考实现失败：%v", round, err)
		}
		got, peak, err := streamDecode(text)
		if err != nil {
			t.Fatalf("第 %d 份：流式实现失败：%v", round, err)
		}
		if len(got) != len(want) {
			t.Fatalf("第 %d 份：元素数不一致：%d vs %d", round, len(got), len(want))
		}
		for index := range want {
			if got[index] != want[index] {
				t.Fatalf("第 %d 份第 %d 项不一致：%+v vs %+v", round, index, got[index], want[index])
			}
		}
		if length >= 10 && peak*2 >= int64(len(text)) {
			t.Fatalf("第 %d 份：峰值应远小于整份输入：peak=%d bytes=%d", round, peak, len(text))
		}
	}
}
