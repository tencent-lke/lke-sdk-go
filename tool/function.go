package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// FunctionTool is a tool implemented as a Go function
type FunctionTool struct {
	name        string
	description string
	function    interface{}
	schema      map[string]interface{}
}

// NewFunctionTool creates a new function tool
func NewFunctionTool(name, description string, fn interface{}, schema map[string]interface{}) (
	*FunctionTool, error) {
	// Validate that fn is a function
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("fn is not fucntion, function tool must be a function")
	}

	// Generate schema from function signature
	autoSchema, err := generateSchemaFromFunction(fnType)
	if err != nil {
		return nil, err
	}

	// unable to auto parse schema, schema should not be nil
	if len(schema) == 0 && len(autoSchema) == 0 {
		return nil, fmt.Errorf("requires a custom schema")
	}
	// using user define first
	if len(schema) != 0 {
		return &FunctionTool{
			name:        name,
			description: description,
			function:    fn,
			schema:      schema,
		}, nil
	}
	return &FunctionTool{
		name:        name,
		description: description,
		function:    fn,
		schema:      autoSchema,
	}, nil
}

// GetName returns the name of the tool
func (t *FunctionTool) GetName() string {
	return t.name
}

// GetDescription returns the description of the tool
func (t *FunctionTool) GetDescription() string {
	return t.description
}

// GetParametersSchema returns the JSON schema for the tool parameters
func (t *FunctionTool) GetParametersSchema() map[string]interface{} {
	return t.schema
}

// Execute executes the tool with the given parameters
func (t *FunctionTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	fnType := reflect.TypeOf(t.function)
	fnValue := reflect.ValueOf(t.function)

	// Check if the function accepts a context as the first parameter
	hasContext := fnType.NumIn() > 0 && fnType.In(0).Implements(reflect.TypeOf((*context.Context)(nil)).Elem())

	// Prepare arguments
	args := make([]reflect.Value, fnType.NumIn())

	// Set context if the function accepts it
	argIndex := 0
	if hasContext {
		args[0] = reflect.ValueOf(ctx)
		argIndex = 1
	}
	// Set parameters based on function signature
	for i := argIndex; i < fnType.NumIn(); i++ {
		paramType := fnType.In(i)
		// If the function expects a map[string]interface{} directly
		if i == argIndex && paramType.Kind() == reflect.Map &&
			paramType.Key().Kind() == reflect.String &&
			paramType.Elem().Kind() == reflect.Interface {
			args[i] = reflect.ValueOf(params)
			continue
		}
		if paramType.Kind() == reflect.Struct {
			structValue := reflect.New(paramType).Elem()
			err := convertToStruct(structValue, params, paramType)
			if err != nil {
				return nil, err
			}
			args[i] = structValue
			continue
		}
		// For a single parameter function with a primitive type, try to use the first parameter or a parameter with the same name
		paramName := ""
		// Only try to access struct fields if the parameter type is a struct
		// if paramType.Kind() == reflect.Struct {
		// 	for j := 0; j < paramType.NumField(); j++ {
		// 		field := paramType.Field(j)
		// 		jsonTag := field.Tag.Get("json")
		// 		if jsonTag != "" {
		// 			parts := strings.Split(jsonTag, ",")
		// 			jsonTag = parts[0]
		// 			if _, ok := params[jsonTag]; ok {
		// 				paramName = jsonTag
		// 				break
		// 			}
		// 		}
		// 	}
		// }

		if paramName == "" && len(params) > 0 {
			// Just use the first parameter
			for name := range params {
				paramName = name
				break
			}
		}

		if paramName != "" {
			if paramValue, ok := params[paramName]; ok {
				// Try to convert the parameter value to the expected type
				convertedValue, err := convertToType(paramValue, paramType)
				if err != nil {
					return nil, fmt.Errorf("failed to convert parameter %s: %w", paramName, err)
				}

				args[i] = reflect.ValueOf(convertedValue)
				continue
			}
		}

		// If we couldn't find a parameter, use the zero value for the type
		args[i] = reflect.Zero(paramType)
	}

	// Call the function
	results := fnValue.Call(args)

	// Handle return values
	if len(results) == 0 {
		return nil, nil
	} else if len(results) == 1 {
		return results[0].Interface(), nil
	} else {
		// Assume the last result is an error
		errVal := results[len(results)-1]
		if errVal.IsNil() {
			return results[0].Interface(), nil
		}
		return results[0].Interface(), errVal.Interface().(error)
	}
}

func convertToStruct(structValue reflect.Value, value map[string]interface{},
	paramType reflect.Type) error {
	if value == nil {
		// default struct value
		return nil
	}
	// Handle struct parameter - map params to struct fields
	if paramType.Kind() == reflect.Struct {
		// For each field in the struct, check if we have a corresponding parameter
		for j := 0; j < paramType.NumField(); j++ {
			field := paramType.Field(j)

			// Get the JSON tag if available
			jsonTag := field.Tag.Get("json")
			if jsonTag == "" {
				jsonTag = field.Name
			} else {
				// Handle json tag options like `json:"name,omitempty"`
				parts := strings.Split(jsonTag, ",")
				jsonTag = parts[0]
			}

			// Check if we have a parameter with this name
			if paramValue, ok := value[jsonTag]; ok {
				// Try to set the field
				fieldValue := structValue.Field(j)
				if fieldValue.CanSet() {
					if field.Type.Kind() != reflect.Struct {
						// Convert the parameter value to the field type
						convertedValue, err := convertToType(paramValue, field.Type)
						if err != nil {
							return fmt.Errorf("failed to convert parameter %s: %w", jsonTag, err)
						}
						fieldValue.Set(reflect.ValueOf(convertedValue))
					} else {
						convertToStruct(fieldValue, paramValue.(map[string]interface{}), field.Type)
					}
				}
			}
		}
		return nil
	}
	return fmt.Errorf("paramType not struct")
}

// convertToType attempts to convert a value to the specified type
func convertToType(value interface{}, targetType reflect.Type) (interface{}, error) {
	// Handle nil special case
	if value == nil {
		return reflect.Zero(targetType).Interface(), nil
	}

	// Get the value's type
	valueType := reflect.TypeOf(value)

	// If the value is already assignable to the target type, return it
	if valueType.AssignableTo(targetType) {
		return value, nil
	}

	// Handle some common conversions
	switch targetType.Kind() {
	case reflect.String:
		// Convert to string
		return fmt.Sprintf("%v", value), nil

	case reflect.Bool:
		// Try to convert to bool
		switch v := value.(type) {
		case bool:
			return v, nil
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			return reflect.ValueOf(v).Int() != 0, nil
		case string:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return false, fmt.Errorf("cannot convert %v to bool: %w", value, err)
			}
			return b, nil
		default:
			return false, fmt.Errorf("cannot convert %v to bool", value)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Try to convert to int
		switch v := value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			intVal := reflect.ValueOf(v).Int()
			return reflect.ValueOf(intVal).Convert(targetType).Interface(), nil
		case float32, float64:
			floatVal := reflect.ValueOf(v).Float()
			return reflect.ValueOf(int64(floatVal)).Convert(targetType).Interface(), nil
		case string:
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("cannot convert %v to int: %w", value, err)
			}
			return reflect.ValueOf(i).Convert(targetType).Interface(), nil
		default:
			return 0, fmt.Errorf("cannot convert %v to int", value)
		}

	case reflect.Float32, reflect.Float64:
		// Try to convert to float
		switch v := value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			intVal := reflect.ValueOf(v).Int()
			return reflect.ValueOf(float64(intVal)).Convert(targetType).Interface(), nil
		case float32, float64:
			floatVal := reflect.ValueOf(v).Float()
			return reflect.ValueOf(floatVal).Convert(targetType).Interface(), nil
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return 0.0, fmt.Errorf("cannot convert %v to float: %w", value, err)
			}
			return reflect.ValueOf(f).Convert(targetType).Interface(), nil
		default:
			return 0.0, fmt.Errorf("cannot convert %v to float", value)
		}

	case reflect.Slice:
		// Try to convert to slice
		switch v := value.(type) {
		case []interface{}:
			elemType := targetType.Elem()
			sliceValue := reflect.MakeSlice(targetType, len(v), len(v))

			for i, elem := range v {
				convertedElem, err := convertToType(elem, elemType)
				if err != nil {
					return nil, fmt.Errorf("cannot convert slice element %d: %w", i, err)
				}
				sliceValue.Index(i).Set(reflect.ValueOf(convertedElem))
			}

			return sliceValue.Interface(), nil
		default:
			return nil, fmt.Errorf("cannot convert %v to slice", value)
		}

	case reflect.Map:
		// Try to convert to map
		if targetType.Key().Kind() == reflect.String {
			switch v := value.(type) {
			case map[string]interface{}:
				elemType := targetType.Elem()
				mapValue := reflect.MakeMap(targetType)

				for key, elem := range v {
					convertedElem, err := convertToType(elem, elemType)
					if err != nil {
						return nil, fmt.Errorf("cannot convert map element %s: %w", key, err)
					}
					mapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(convertedElem))
				}

				return mapValue.Interface(), nil
			default:
				return nil, fmt.Errorf("cannot convert %v to map", value)
			}
		}

	case reflect.Struct:
		// struct recursion parse
		mapValue, ok := value.(map[string]interface{})
		if ok {
			structValue := reflect.New(targetType).Elem()
			return convertToStruct(structValue, mapValue, targetType), nil
		}
	}

	// If we couldn't convert, return an error
	return nil, fmt.Errorf("cannot convert %v (type %T) to %v", value, value, targetType)
}

// generateSchemaFromFunction generates a JSON schema from a function signature
func generateSchemaFromFunction(fnType reflect.Type) (map[string]interface{}, error) {
	// Initialize schema
	schema := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}

	// Check if the function accepts a context as the first parameter
	hasContext := fnType.NumIn() > 0 && fnType.In(0).Implements(reflect.TypeOf((*context.Context)(nil)).Elem())

	// Start from the first non-context parameter
	startIndex := 0
	if hasContext {
		startIndex = 1
	}

	// If the function has no parameters beyond context, return empty schema
	if fnType.NumIn() <= startIndex {
		return schema, nil
	}

	if fnType.NumIn() > startIndex+1 {
		return nil, fmt.Errorf("excluding context, a function can have at most one parameter")
	}
	// Get the first parameter type after context (if any)
	paramType := fnType.In(startIndex)

	// If the parameter is a map[string]interface{}, we can't infer the schema, user should define schema
	if paramType.Kind() == reflect.Map && paramType.Key().Kind() == reflect.String &&
		paramType.Elem().Kind() == reflect.Interface {
		return nil, nil
	}

	// If the parameter is a struct, create a schema from its fields
	if paramType.Kind() == reflect.Struct {
		for i := 0; i < paramType.NumField(); i++ {
			field := paramType.Field(i)

			// Skip unexported fields
			if field.PkgPath != "" {
				continue
			}

			// Get the field name from JSON tag or fallback to field name
			fieldName := field.Name
			jsonTag := field.Tag.Get("json")
			if jsonTag != "" {
				// Handle json tag options like `json:"name,omitempty"`
				parts := strings.Split(jsonTag, ",")
				fieldName = parts[0]

				// Skip if the field is explicitly omitted with "-"
				if fieldName == "-" {
					continue
				}

				// Check if the field is required (not marked as omitempty)
				isRequired := true
				for _, part := range parts[1:] {
					if part == "omitempty" {
						isRequired = false
						break
					}
				}

				if isRequired {
					schema["required"] = append(schema["required"].([]string), fieldName)
				}
			} else {
				// If no JSON tag, assume it's required
				schema["required"] = append(schema["required"].([]string), fieldName)
			}

			// Get the field schema
			fieldSchema := getTypeSchema(field.Type)

			// Add description from doc tag if available
			if docTag := field.Tag.Get("doc"); docTag != "" {
				fieldSchema["description"] = docTag
			}

			// Add the field to properties
			schema["properties"].(map[string]interface{})[fieldName] = fieldSchema
		}
	} else {
		// For other parameter types, not supported
		return nil, fmt.Errorf("unsupported function definition, please refer to the sdk documentation")
	}

	return schema, nil
}

// getTypeSchema returns the JSON schema for a Go type
func getTypeSchema(t reflect.Type) map[string]interface{} {
	schema := make(map[string]interface{})

	// Handle pointers
	if t.Kind() == reflect.Ptr {
		elemSchema := getTypeSchema(t.Elem())

		// For pointers, the field is nullable
		if enum, ok := elemSchema["enum"]; ok {
			// If the schema has enum values, add null to the enum
			enumValues := enum.([]interface{})
			enumValues = append(enumValues, nil)
			elemSchema["enum"] = enumValues
		}

		return elemSchema
	}

	// Handle different types
	switch t.Kind() {
	case reflect.Bool:
		schema["type"] = "boolean"

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema["type"] = "integer"

	case reflect.Float32, reflect.Float64:
		schema["type"] = "number"

	case reflect.String:
		schema["type"] = "string"

	case reflect.Slice, reflect.Array:
		schema["type"] = "array"
		schema["items"] = getTypeSchema(t.Elem())

	case reflect.Map:
		schema["type"] = "object"
		if t.Key().Kind() == reflect.String {
			schema["additionalProperties"] = getTypeSchema(t.Elem())
		} else {
			// Non-string keyed maps are not well represented in JSON Schema
			schema["additionalProperties"] = true
		}

	case reflect.Struct:
		schema["type"] = "object"
		schema["properties"] = make(map[string]interface{})
		schema["required"] = []string{}

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)

			// Skip unexported fields
			if field.PkgPath != "" {
				continue
			}

			// Get the field name from JSON tag or fallback to field name
			fieldName := field.Name
			jsonTag := field.Tag.Get("json")
			if jsonTag != "" {
				// Handle json tag options like `json:"name,omitempty"`
				parts := strings.Split(jsonTag, ",")
				fieldName = parts[0]

				// Skip if the field is explicitly omitted with "-"
				if fieldName == "-" {
					continue
				}

				// Check if the field is required (not marked as omitempty)
				isRequired := true
				for _, part := range parts[1:] {
					if part == "omitempty" {
						isRequired = false
						break
					}
				}

				if isRequired {
					schema["required"] = append(schema["required"].([]string), fieldName)
				}
			} else {
				// If no JSON tag, assume it's required
				schema["required"] = append(schema["required"].([]string), fieldName)
			}

			// Get the field schema
			fieldSchema := getTypeSchema(field.Type)

			// Add description from doc tag if available
			if docTag := field.Tag.Get("doc"); docTag != "" {
				fieldSchema["description"] = docTag
			}

			// Add the field to properties
			schema["properties"].(map[string]interface{})[fieldName] = fieldSchema
		}

	default:
		// For unknown types, fallback to string
		schema["type"] = "string"
	}

	return schema
}

// WithSchema sets a custom schema for the tool parameters
func (t *FunctionTool) WithSchema(schema map[string]interface{}) *FunctionTool {
	t.schema = schema
	return t
}

// WithDescription updates the description of the tool
func (t *FunctionTool) WithDescription(description string) *FunctionTool {
	t.description = description
	return t
}

// WithName updates the name of the tool
func (t *FunctionTool) WithName(name string) *FunctionTool {
	t.name = name
	return t
}

// InterfaceToString 转换成 string
func InterfaceToString(i interface{}) (string, error) {
	// 使用反射判断类型
	val := reflect.ValueOf(i)

	// 如果是指针，获取其指向的值
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// 判断是否是 map 或者结构体
	if val.Kind() == reflect.Map || val.Kind() == reflect.Struct {
		// 将其转换为 JSON 字符串
		jsonBytes, err := json.Marshal(val.Interface())
		if err != nil {
			return "", err
		}
		return string(jsonBytes), nil
	}

	// 对于其他类型，直接转换为字符串
	return fmt.Sprintf("%v", i), nil
}

// GenerateRandomData 根据 map 结构的 Schema 生成随机数据
func GenerateRandomSchema(schema map[string]interface{}) interface{} {
	// 优先处理 enum
	if enum, ok := schema["enum"].([]interface{}); ok && len(enum) > 0 {
		return enum[rand.Intn(len(enum))]
	}

	typeName, _ := schema["type"].(string)
	switch typeName {
	case "object":
		return generateObject(schema)
	case "array":
		return generateArray(schema)
	case "string":
		return generateString(schema)
	case "number", "integer":
		return generateNumber(schema)
	case "boolean":
		return rand.Intn(2) == 1
	default:
		return nil
	}
}

func generateObject(schema map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for name, propSchema := range properties {
			if childSchema, ok := propSchema.(map[string]interface{}); ok {
				result[name] = GenerateRandomSchema(childSchema)
			}
		}
	}
	return result
}

func generateArray(schema map[string]interface{}) []interface{} {
	length := rand.Intn(5) + 1 // 生成1-5个元素
	if minItems, ok := schema["minItems"].(float64); ok {
		length = int(minItems) + rand.Intn(5)
	}

	arr := make([]interface{}, length)
	if items, ok := schema["items"].(map[string]interface{}); ok {
		for i := range arr {
			arr[i] = GenerateRandomSchema(items)
		}
	}
	return arr
}

func generateString(schema map[string]interface{}) string {
	// 处理可能的格式约束
	if format, ok := schema["format"].(string); ok {
		switch format {
		case "date-time":
			return time.Now().Format(time.RFC3339)
		case "email":
			return fmt.Sprintf("%s@%s.com", randomString(8), randomString(5))
		}
	}

	// 处理长度约束
	minLength := 5
	if ml, ok := schema["minLength"].(float64); ok {
		minLength = int(ml)
	}
	maxLength := minLength + 5
	if ml, ok := schema["maxLength"].(float64); ok {
		maxLength = int(ml)
	}
	return randomString(minLength + rand.Intn(maxLength-minLength+1))
}

func generateNumber(schema map[string]interface{}) float64 {
	min := 0.0
	max := 100.0

	if mi, ok := schema["minimum"].(float64); ok {
		min = mi
	}
	if ma, ok := schema["maximum"].(float64); ok {
		max = ma
	}

	// 处理整数类型
	if schema["type"] == "integer" {
		return float64(int(min) + rand.Intn(int(max-min)+1))
	}

	return min + rand.Float64()*(max-min)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
