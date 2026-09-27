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

## 实现方式

- `NewReader(io.Reader)` 用 `json.Decoder` 逐元素解码（`Token`/`More` + 单元素 `Decode`），
  峰值内存 O(单元素)；`New([]byte)` 是其内存版便捷封装。
- `value` 按 `json.RawMessage` 取出后用 `strconv.ParseInt` 解析：整数原样保留，
  字符串、小数、超出 int64 都报带元素下标的错误。
- 增量接口：`Next()` 逐项（读完返回 `io.EOF`）、`ReadN(n)` 一次取 n 项、
  `Each(fn)` 回调式遍历，调用方不需要把结果全存下来。
- `peak` 统计迄今最大单元素的字节数；`bytes` 是流过的输入字节数。
- `reference_test.go` 里有「整份 Unmarshal」的朴素参考实现，测试对 1000 份随机数组
  逐元素比对键、值与下标，并检查峰值内存远小于整份输入。

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
stream.go         流式读取（Next / ReadN / Each）
cmd/json-stream   命令行入口（--file/--generate 也是流式）
stream_test.go    go test 用例
reference_test.go 整份 Unmarshal 参考实现与对照测试
```
