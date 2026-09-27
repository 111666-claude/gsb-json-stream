# json-stream

流式读取 JSON 数组，只用 Go 标准库。

```
go test ./...
go run ./cmd/json-stream --text '[{"key":"a","value":9007199254740993}]'
go run ./cmd/json-stream --generate 100000 --counters
```

## 口径（README 为准）

- **逐元素解码**（`json.Decoder` 的 `Token`/`More` 或流式扫描），峰值内存 O(单元素)：
  `--generate 100000 --counters` 现在 `peak=3088891`（等于整份输入），按口径应该与实际元素大小同阶。
- **整数不能经 float64**：`value` 是 JSON 整数时要原样保留，`9007199254740993` 不许变成 `9007199254740992`
  （现在被 `float64` 截断）。
- 元素字段是 `key`（字符串）与 `value`（整数）；缺字段或类型不对要报错并给出元素下标。
- 不变量：`elements` 等于数组长度；`Next()` 按下标递增返回；同一输入重复读取结果相同；
  空数组返回 0 个元素。
- 规模：1 GB 的 JSON 数组要能边读边处理，峰值内存 O(单元素)；不许整份读进内存再 `Unmarshal`。

## 现在的行为

- `New` 先 `json.Unmarshal` 成 `[]map[string]any`：整份进内存（`peak` 等于输入长度）。
- `value` 从 `float64` 转 `int64`：`9007199254740993` 变成 `9007199254740992`。
- 没有流式接口，调用方也拿不到增量结果；`bufio` 都没用上。

## 计划新增的能力

```
--generate N   生成 N 个元素的数组（内存统计用）
--counters     打印峰值内存估计与输入字节数
```

## 输出

```
elements=… last=键:值
peak=… bytes=…
```

## 目录

```
stream.go         流式读取
cmd/json-stream   命令行入口
stream_test.go    go test 用例
```
