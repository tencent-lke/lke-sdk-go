package tool_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/tencent-lke/lke-sdk-go/tool"
)

// CompareMaps 判断两个 map[string]interface{} 是否相同
func CompareMaps(a, b map[string]interface{}) bool {
	// 首先检查两个 map 的长度是否相同
	if len(a) != len(b) {
		return false
	}

	// 遍历第一个 map，检查每个键值对是否在第二个 map 中存在
	for key, valueA := range a {
		valueB, exists := b[key]
		if !exists {
			return false // 如果 b 中没有这个键，返回 false
		}

		// 使用 reflect.DeepEqual 比较值
		if !reflect.DeepEqual(valueA, valueB) {
			return false // 如果值不相等，返回 false
		}
	}

	// 如果所有键值对都匹配，返回 true
	return true
}

func assertMap(t *testing.T, except, actual map[string]interface{}) bool {
	if CompareMaps(except, actual) {
		return true
	}
	bs1, _ := json.MarshalIndent(except, "  ", "  ")
	bs2, _ := json.MarshalIndent(actual, "  ", "  ")
	t.Fatalf("except schema: %s\n, actual schema:%s", string(bs1), string(bs2))
	return false
}

func TestFunctionSchema1(t *testing.T) {
	// struct
	type Add struct {
		A int `json:"a" doc:"number a"`
		B int `json:"b" doc:"number b"`
	}
	structAdd := func(param Add) int {
		return param.A + param.B
	}
	excepetSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{
				"type":        "integer",
				"description": "number a",
			},
			"b": map[string]interface{}{
				"type":        "integer",
				"description": "number b",
			},
		},
		"required": []string{"a", "b"},
	}

	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, nil)
	if err != nil {
		t.Fatal(err)
	}

	assertMap(t, excepetSchema, to.GetParametersSchema())

	structAddCtx := func(ctx context.Context, param Add) int {
		return param.A + param.B
	}
	to, err = tool.NewFunctionTool("add", "两个数的和", structAddCtx, nil)
	if err != nil {
		t.Fatal(err)
	}

	assertMap(t, excepetSchema, to.GetParametersSchema())
}

func TestFunctionSchema2(t *testing.T) {
	// 部分 tag 丢失，会缺省
	type Add struct {
		A int `doc:"number a"`
		B int `json:"b"`
	}
	structAdd := func(param Add) int {
		return param.A + param.B
	}

	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, nil)
	if err != nil {
		t.Fatal(err)
	}
	excepetSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"A": map[string]interface{}{
				"type":        "integer",
				"description": "number a",
			},
			"b": map[string]interface{}{
				"type": "integer",
			},
		},
		"required": []string{"A", "b"},
	}
	assertMap(t, excepetSchema, to.GetParametersSchema())

	structAddCtx := func(ctx context.Context, param Add) int {
		return param.A + param.B
	}
	to, err = tool.NewFunctionTool("add", "两个数的和", structAddCtx, nil)
	if err != nil {
		t.Fatal(err)
	}

	assertMap(t, excepetSchema, to.GetParametersSchema())
}

func TestFunctionSchema3(t *testing.T) {
	// 用用户自定义的 schema
	type Add struct {
		A int `doc:"number a"`
		B int `json:"b"`
	}
	structAdd := func(param Add) int {
		return param.A + param.B
	}

	excepetSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"A": map[string]interface{}{
				"type":        "integer",
				"description": "number a",
			},
			"b": map[string]interface{}{
				"type": "integer",
			},
		},
		"required": []string{"A", "b"},
	}

	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, excepetSchema)
	if err != nil {
		t.Fatal(err)
	}

	assertMap(t, excepetSchema, to.GetParametersSchema())

	structAddCtx := func(ctx context.Context, param Add) int {
		return param.A + param.B
	}
	to, err = tool.NewFunctionTool("add", "两个数的和", structAddCtx, nil)
	if err != nil {
		t.Fatal(err)
	}

	assertMap(t, excepetSchema, to.GetParametersSchema())
}

func TestFunctionSchema4(t *testing.T) {
	// 嵌套的 struct
	type B struct {
		B string `json:"b" doc:"string b"`
	}
	type Add struct {
		A string `json:"a" doc:"string a"`
		B B      `json:"b" doc:"struct b"`
	}
	structAdd := func(param Add) string {
		return fmt.Sprintf("%s+%s\n", param.A, param.B.B)
	}

	excepetSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{
				"type":        "string",
				"description": "string a",
			},
			"b": map[string]interface{}{
				"type":        "object",
				"description": "struct b",
				"properties": map[string]interface{}{
					"b": map[string]interface{}{
						"description": "string b",
						"type":        "string",
					},
				},
				"required": []string{"b"},
			},
		},
		"required": []string{"a", "b"},
	}

	to, err := tool.NewFunctionTool("add", "连接2个字符串", structAdd, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertMap(t, excepetSchema, to.GetParametersSchema())
}

func TestFunctionSchema5(t *testing.T) {
	// 嵌套 interface{}
	type Add struct {
		A int         `json:"a" doc:"number a"`
		B interface{} `json:"b" doc:"number b"`
	}
	structAdd := func(param Add) int {
		return param.A + param.B.(int)
	}

	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, nil)
	if err != nil {
		t.Fatal(err)
	}
	excepetSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{
				"type":        "integer",
				"description": "number a",
			},
			"b": map[string]interface{}{
				"type":        "string",
				"description": "number b",
			},
		},
		"required": []string{"a", "b"},
	}
	assertMap(t, excepetSchema, to.GetParametersSchema())

	structAddCtx := func(ctx context.Context, param Add) int {
		return param.A + param.B.(int)
	}
	to, err = tool.NewFunctionTool("add", "两个数的和", structAddCtx, nil)
	if err != nil {
		t.Fatal(err)
	}

	assertMap(t, excepetSchema, to.GetParametersSchema())

}

func TestFunctionSchema6(t *testing.T) {
	// 不需要参数
	structAdd := func() int {
		return 0
	}
	excepetSchema := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}

	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, nil)
	if err != nil {
		t.Fatal(err)
	}

	assertMap(t, excepetSchema, to.GetParametersSchema())

	structAddCtx := func(ctx context.Context) int {
		return 0
	}
	to, err = tool.NewFunctionTool("add", "两个数的和", structAddCtx, nil)
	if err != nil {
		t.Fatal(err)
	}

	assertMap(t, excepetSchema, to.GetParametersSchema())
}

func TestFunctionSchema7(t *testing.T) {
	// map[string]interface{} 函数
	structAdd := func(params map[string]interface{}) int {
		a, oka := params["a"].(int)
		b, okb := params["b"].(int)
		if oka && okb {
			return a + b
		}
		return 0
	}
	excepetSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{
				"type":        "integer",
				"description": "number a",
			},
			"b": map[string]interface{}{
				"type":        "integer",
				"description": "number b",
			},
		},
		"required": []string{"a", "b"},
	}
	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, excepetSchema)
	if err != nil {
		t.Fatal(err)
	}

	assertMap(t, excepetSchema, to.GetParametersSchema())

	structAddCtx := func(ctx context.Context, params map[string]interface{}) int {
		a, oka := params["a"].(int)
		b, okb := params["b"].(int)
		if oka && okb {
			return a + b
		}
		return 0
	}
	to, err = tool.NewFunctionTool("add", "两个数的和", structAddCtx, excepetSchema)
	if err != nil {
		t.Fatal(err)
	}

	assertMap(t, excepetSchema, to.GetParametersSchema())
}

func TestFunctionSchema8(t *testing.T) {
	// map[string]interface{} 函数
	structAdd := func(params map[string]interface{}) int {
		a, oka := params["a"].(int)
		b, okb := params["b"].(int)
		if oka && okb {
			return a + b
		}
		return 0
	}
	_, err := tool.NewFunctionTool("add", "两个数的和", structAdd, nil)
	if err == nil {
		t.Fatal("except error")
	}
}

func TestFunctionSchema9(t *testing.T) {
	// 不支持的函数
	func1 := func(ctx context.Context, x int) int {
		return 0
	}
	_, err := tool.NewFunctionTool("add", "两个数的和", func1, nil)
	if err == nil {
		t.Fatal("except error")
	}
	func2 := func(x int) int {
		return 0
	}
	_, err = tool.NewFunctionTool("add", "两个数的和", func2, nil)
	if err == nil {
		t.Fatal("except error")
	}
	func3 := func(a, b int, c string) int {
		return 0
	}
	_, err = tool.NewFunctionTool("add", "两个数的和", func3, nil)
	if err == nil {
		t.Fatal("except error")
	}
	excepetSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{
				"type":        "integer",
				"description": "number a",
			},
			"b": map[string]interface{}{
				"type":        "integer",
				"description": "number b",
			},
		},
		"required": []string{"a", "b"},
	}
	func4 := func(params map[string]map[int]string) int {
		return 0
	}
	_, err = tool.NewFunctionTool("add", "两个数的和", func4, excepetSchema)
	if err == nil {
		t.Fatal("except error")
	}
	type Add struct {
		A int         `json:"a" doc:"number a"`
		B interface{} `json:"b" doc:"number b"`
	}
	structAdd := func(ctx context.Context, p1, p2 Add) int {
		return p1.A + p2.A
	}
	_, err = tool.NewFunctionTool("add", "两个数的和", structAdd, nil)
	if err == nil {
		t.Fatal("except error")
	}
}

func TestFunctionSchema10(t *testing.T) {
	// struct 中包含数组
	type Add struct {
		A int   `json:"a" doc:"number a"`
		B []int `json:"b" doc:"number b"`
	}
	structAdd := func(param Add) int {
		sum := param.A
		for _, b := range param.B {
			sum += b
		}
		return sum
	}
	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, nil)
	if err != nil {
		t.Fatal(err)
	}

	excepetSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{
				"type":        "integer",
				"description": "number a",
			},
			"b": map[string]interface{}{
				"type":        "array",
				"description": "number b",
				"items": map[string]interface{}{
					"type": "integer",
				},
			},
		},
		"required": []string{"a", "b"},
	}

	assertMap(t, excepetSchema, to.GetParametersSchema())

	structAddCtx := func(ctx context.Context, param Add) int {
		sum := param.A
		for _, b := range param.B {
			sum += b
		}
		return sum
	}
	to, err = tool.NewFunctionTool("add", "两个数的和", structAddCtx, nil)
	if err != nil {
		t.Fatal(err)
	}

	assertMap(t, excepetSchema, to.GetParametersSchema())
}

func TestFunctionSchema11(t *testing.T) {
	// struct 中包含结构体数组
	type B struct {
		B int `json:"b" doc:"number b"`
	}
	type Add struct {
		A int `json:"a" doc:"number a"`
		B []B `json:"b" doc:"arr struct b"`
	}

	structAdd := func(param Add) int {
		sum := param.A
		for _, b := range param.B {
			sum += b.B
		}
		return sum
	}
	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, nil)
	if err != nil {
		t.Fatal(err)
	}

	excepetSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{
				"type":        "integer",
				"description": "number a",
			},
			"b": map[string]interface{}{
				"type":        "array",
				"description": "arr struct b",
				"items": map[string]interface{}{
					"properties": map[string]interface{}{
						"b": map[string]interface{}{
							"description": "number b",
							"type":        "integer",
						},
					},
					"required": []string{"b"},
					"type":     "object",
				},
			},
		},
		"required": []string{"a", "b"},
	}

	assertMap(t, excepetSchema, to.GetParametersSchema())

	structAddCtx := func(ctx context.Context, param Add) int {
		sum := param.A
		for _, b := range param.B {
			sum += b.B
		}
		return sum
	}
	to, err = tool.NewFunctionTool("add", "两个数的和", structAddCtx, nil)
	if err != nil {
		t.Fatal(err)
	}

	assertMap(t, excepetSchema, to.GetParametersSchema())
}

func TestFunctionExec1(t *testing.T) {
	// struct 正常完全
	ctx := context.Background()
	type Add struct {
		A string `json:"a" doc:"string a"`
		B string `json:"b" doc:"string b"`
	}
	structAdd := func(param Add) string {
		return param.A + param.B
	}

	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, nil)
	if err != nil {
		t.Fatal(err)
	}

	a, b := "a", "b"
	input := map[string]interface{}{
		"a": a,
		"b": b,
	}
	v, err := to.Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if x, ok := v.(string); !ok {
		t.Fatalf("返回值非预期, type: %s", reflect.TypeOf(v).Name())
	} else {
		if x != a+b {
			t.Fatalf("返回值错误, except:%s real:%s", a+b, x)
		}
	}
	// 测试 int to string
	a1, b1 := 1, 2
	input = map[string]interface{}{
		"a": a1,
		"b": b1,
	}
	v, err = to.Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if x, ok := v.(string); !ok {
		t.Fatalf("返回值非预期, type: %s", reflect.TypeOf(v).Name())
	} else {
		ex := fmt.Sprintf("%d%d", a1, b1)
		if x != ex {
			t.Fatalf("返回值错误, except:%s real:%s", ex, x)
		}
	}

	// 测试 float to string
	f1, f2 := 1.4, 2.7
	input = map[string]interface{}{
		"a": f1,
		"b": f2,
	}
	v, err = to.Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if x, ok := v.(string); !ok {
		t.Fatalf("返回值非预期, type: %s", reflect.TypeOf(v).Name())
	} else {
		ex := fmt.Sprintf("%.1f%.1f", f1, f2)
		if x != ex {
			t.Fatalf("返回值错误, except:%s real:%s", ex, x)
		}
	}

	type AddInt struct {
		A int `json:"a" doc:"int a"`
		B int `json:"b" doc:"int b"`
	}
	intAdd := func(param AddInt) int {
		return param.A + param.B
	}

	to, err = tool.NewFunctionTool("add", "两个数的和", intAdd, nil)
	if err != nil {
		t.Fatal(err)
	}
	input = map[string]interface{}{
		"a": a1,
		"b": b1,
	}
	v, err = to.Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if x, ok := v.(int); !ok {
		t.Fatalf("返回值非预期, type: %s", reflect.TypeOf(v).Name())
	} else {
		if x != a1+b1 {
			t.Fatalf("返回值错误, except:%d real:%d", a1+b1, x)
		}
	}
	// float 给 int
	input = map[string]interface{}{
		"a": f1,
		"b": f2,
	}
	v, err = to.Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if x, ok := v.(int); !ok {
		t.Fatalf("返回值非预期, type: %s", reflect.TypeOf(v).Name())
	} else {
		ex := int(f1) + int(f2)
		if x != ex {
			t.Fatalf("返回值错误, except:%d real:%d", ex, x)
		}
	}

	// string 给 int
	input = map[string]interface{}{
		"a": a,
		"b": b,
	}
	_, err = to.Execute(ctx, input)
	if err == nil {
		t.Fatal("except error")
	}
	input = map[string]interface{}{
		"a": fmt.Sprintf("%d", a1),
		"b": fmt.Sprintf("%d", b1),
	}
	v, err = to.Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if x, ok := v.(int); !ok {
		t.Fatalf("返回值非预期, type: %s", reflect.TypeOf(v).Name())
	} else {
		ex := a1 + b1
		if x != ex {
			t.Fatalf("返回值错误, except:%d real:%d", ex, x)
		}
	}
}

func TestFunctionExec2(t *testing.T) {
	ctx := context.Background()
	// 部分 tag 丢失，会缺省
	type Add struct {
		A int `doc:"number a"`
		B int `json:"b"`
	}
	structAdd := func(param Add) int {
		return param.A + param.B
	}

	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, b := 100, 2300
	input := map[string]interface{}{
		"A": a,
		"b": b,
	}
	v, err := to.Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if x, ok := v.(int); !ok {
		t.Fatalf("返回值非预期, type: %s", reflect.TypeOf(v).Name())
	} else {
		ex := a + b
		if x != ex {
			t.Fatalf("返回值错误, except:%d real:%d", ex, x)
		}
	}
}

func TestFunctionExec3(t *testing.T) {
	ctx := context.Background()
	// 用用户自定义的 schema和解析对应不上，用默认值
	type Add struct {
		A int `doc:"number a"`
		B int `json:"b"`
	}
	structAdd := func(param Add) int {
		return param.A + param.B
	}

	excepetSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"A": map[string]interface{}{
				"type":        "integer",
				"description": "number a",
			},
			"b": map[string]interface{}{
				"type": "integer",
			},
		},
		"required": []string{"A", "b"},
	}

	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, excepetSchema)
	if err != nil {
		t.Fatal(err)
	}
	a, b := 100, 2300
	input := map[string]interface{}{
		"A": a,
		"b": b,
	}
	v, err := to.Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if x, ok := v.(int); !ok {
		t.Fatalf("返回值非预期, type: %s", reflect.TypeOf(v).Name())
	} else {
		ex := a + b
		if x != ex {
			t.Fatalf("返回值错误, except:%d real:%d", ex, x)
		}
	}
}

func TestFunctionExec4(t *testing.T) {
	ctx := context.Background()
	// 嵌套的 struct
	type B struct {
		B string `json:"b" doc:"string b"`
	}
	type Add struct {
		A string `json:"a" doc:"string a"`
		B B      `json:"b" doc:"struct b"`
	}
	structAdd := func(param Add) string {
		return fmt.Sprintf("%s%s", param.A, param.B.B)
	}

	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, nil)
	if err != nil {
		t.Fatal(err)
	}
	a, b := "a", "b"
	input := map[string]interface{}{
		"a": a,
		"b": map[string]interface{}{
			"b": b,
		},
	}
	v, err := to.Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if x, ok := v.(string); !ok {
		t.Fatalf("返回值非预期, type: %s", reflect.TypeOf(v).Name())
	} else {
		ex := a + b
		if x != ex {
			t.Fatalf("返回值错误, except:%s real:%s", ex, x)
		}
	}
}

func TestFunctionExec5(t *testing.T) {
	ctx := context.Background()
	// 嵌套 interface{}
	type Add struct {
		A int         `json:"a" doc:"number a"`
		B interface{} `json:"b" doc:"number b"`
	}
	structAdd := func(param Add) int {
		return param.A + param.B.(int)
	}

	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, nil)
	if err != nil {
		t.Fatal(err)
	}

	a, b := 1, 2
	input := map[string]interface{}{
		"a": a,
		"b": b,
	}
	v, err := to.Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if x, ok := v.(int); !ok {
		t.Fatalf("返回值非预期, type: %s", reflect.TypeOf(v).Name())
	} else {
		ex := a + b
		if x != ex {
			t.Fatalf("返回值错误, except:%d real:%d", ex, x)
		}
	}
}

func TestFunctionExec6(t *testing.T) {
	ctx := context.Background()
	// 不需要参数
	structAdd := func() int {
		return 0
	}

	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, nil)
	if err != nil {
		t.Fatal(err)
	}

	a, b := 1, 2
	input := map[string]interface{}{
		"a": a,
		"b": b,
	}
	v, err := to.Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if x, ok := v.(int); !ok {
		t.Fatalf("返回值非预期, type: %s", reflect.TypeOf(v).Name())
	} else {
		ex := 0
		if x != ex {
			t.Fatalf("返回值错误, except:%d real:%d", ex, x)
		}
	}
}

func TestFunctionExec7(t *testing.T) {
	ctx := context.Background()
	// map[string]interface{} 函数
	structAdd := func(params map[string]interface{}) int {
		a, oka := params["a"].(int)
		b, okb := params["b"].(int)
		if oka && okb {
			return a + b
		}
		return 0
	}
	excepetSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{
				"type":        "integer",
				"description": "number a",
			},
			"b": map[string]interface{}{
				"type":        "integer",
				"description": "number b",
			},
		},
		"required": []string{"a", "b"},
	}
	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, excepetSchema)
	if err != nil {
		t.Fatal(err)
	}

	a, b := 1, 2
	input := map[string]interface{}{
		"a": a,
		"b": b,
	}
	v, err := to.Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if x, ok := v.(int); !ok {
		t.Fatalf("返回值非预期, type: %s", reflect.TypeOf(v).Name())
	} else {
		ex := a + b
		if x != ex {
			t.Fatalf("返回值错误, except:%d real:%d", ex, x)
		}
	}
}

func TestFunctionExec10(t *testing.T) {
	ctx := context.Background()
	// struct 中包含数组
	type Add struct {
		A int   `json:"a" doc:"number a"`
		B []int `json:"b" doc:"number b"`
	}
	structAdd := func(param Add) int {
		sum := param.A
		for _, b := range param.B {
			sum += b
		}
		return sum
	}
	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, nil)
	if err != nil {
		t.Fatal(err)
	}

	a, b := 1, 2
	input := map[string]interface{}{
		"a": a,
		"b": []int{b, b},
	}
	v, err := to.Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if x, ok := v.(int); !ok {
		t.Fatalf("返回值非预期, type: %s", reflect.TypeOf(v).Name())
	} else {
		ex := a + 2*b
		if x != ex {
			t.Fatalf("返回值错误, except:%d real:%d", ex, x)
		}
	}
}

func TestFunctionExec11(t *testing.T) {
	ctx := context.Background()
	// struct 中包含结构体数组
	type B struct {
		B int `json:"b" doc:"number b"`
	}
	type Add struct {
		A int `json:"a" doc:"number a"`
		B []B `json:"b" doc:"arr struct b"`
	}

	structAdd := func(param Add) int {
		sum := param.A
		for _, b := range param.B {
			sum += b.B
		}
		return sum
	}
	to, err := tool.NewFunctionTool("add", "两个数的和", structAdd, nil)
	if err != nil {
		t.Fatal(err)
	}

	a, b := 1, 2
	input := map[string]interface{}{
		"a": a,
		"b": []B{
			{
				B: b,
			},
			{
				B: b,
			},
		},
	}
	v, err := to.Execute(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if x, ok := v.(int); !ok {
		t.Fatalf("返回值非预期, type: %s", reflect.TypeOf(v).Name())
	} else {
		ex := a + 2*b
		if x != ex {
			t.Fatalf("返回值错误, except:%d real:%d", ex, x)
		}
	}
}
