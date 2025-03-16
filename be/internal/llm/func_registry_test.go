package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestFuncRegistry_RegisterAndExecute_WithTypedFunction(t *testing.T) {
	// Define strongly typed request/response structures
	type WeatherRequest struct {
		City    string `json:"city"`
		Country string `json:"country"`
		Units   string `json:"units"`
	}

	type WeatherResponse struct {
		Temperature float64 `json:"temperature"`
		Conditions  string  `json:"conditions"`
		Humidity    int     `json:"humidity"`
	}

	// Create a typed function
	weatherFn := func(ctx context.Context, args WeatherRequest) (WeatherResponse, error) {
		if args.City == "Paris" && args.Country == "France" {
			return WeatherResponse{
				Temperature: 22.5,
				Conditions:  "Sunny",
				Humidity:    65,
			}, nil
		}
		return WeatherResponse{}, fmt.Errorf("location not found")
	}

	// Register the wrapped function
	registry := NewFunctionRegistryImpl()
	registry.Register(FuncWrapper("getWeather", "", weatherFn))

	// Execute the tool
	toolCall := ToolCall{
		Function: &FunctionCall{
			Name:      "getWeather",
			Arguments: `{"city": "Paris", "country": "France", "units": "metric"}`,
		},
	}

	result, err := registry.Execute(context.Background(), toolCall)
	require.NoError(t, err)
	assert.JSONEq(t, `{"temperature":22.5,"conditions":"Sunny","humidity":65}`, result)
}

func TestFuncRegistry_Execute_FailureWithTypedFunction(t *testing.T) {
	type SearchRequest struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}

	type SearchResponse struct {
		Results []string `json:"results"`
		Total   int      `json:"total"`
	}

	searchFn := func(ctx context.Context, args SearchRequest) (SearchResponse, error) {
		if args.Query == "" {
			return SearchResponse{}, errors.New("empty search query")
		}
		return SearchResponse{
			Results: []string{"result1", "result2"},
			Total:   2,
		}, nil
	}

	registry := NewFunctionRegistryImpl()
	registry.Register(FuncWrapper("search", "", searchFn))

	toolCall := ToolCall{
		Function: &FunctionCall{
			Name:      "search",
			Arguments: `{"query": "", "limit": 10}`,
		},
	}

	_, err := registry.Execute(context.Background(), toolCall)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty search query")
}

func TestFuncRegistry_Execute_TypeConversionError(t *testing.T) {
	type UserRequest struct {
		ID       int       `json:"id"`
		JoinDate time.Time `json:"join_date"`
	}

	getUserFn := func(ctx context.Context, args UserRequest) (map[string]interface{}, error) {
		return map[string]interface{}{
			"username": fmt.Sprintf("user%d", args.ID),
			"joined":   args.JoinDate.Format(time.RFC3339),
		}, nil
	}

	registry := NewFunctionRegistryImpl()
	registry.Register(FuncWrapper("getUser", "", getUserFn))

	// Send invalid type for ID (string instead of int)
	toolCall := ToolCall{
		Function: &FunctionCall{
			Name:      "getUser",
			Arguments: `{"id": "not-a-number", "join_date": "2023-01-01T00:00:00Z"}`,
		},
	}

	_, err := registry.Execute(context.Background(), toolCall)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal args")
}

func TestFuncRegistry_Execute_ComplexNestedStructure(t *testing.T) {
	type OrderItem struct {
		ProductID string  `json:"product_id"`
		Quantity  int     `json:"quantity"`
		UnitPrice float64 `json:"unit_price"`
	}

	type CreateOrderRequest struct {
		CustomerID      string      `json:"customer_id"`
		Items           []OrderItem `json:"items"`
		ShippingAddress struct {
			Street  string `json:"street"`
			City    string `json:"city"`
			Country string `json:"country"`
			Zip     string `json:"zip"`
		} `json:"shipping_address"`
	}

	type OrderResponse struct {
		OrderID           string  `json:"order_id"`
		TotalAmount       float64 `json:"total_amount"`
		EstimatedDelivery string  `json:"estimated_delivery"`
	}

	createOrderFn := func(ctx context.Context, req CreateOrderRequest) (OrderResponse, error) {
		var total float64
		for _, item := range req.Items {
			total += item.UnitPrice * float64(item.Quantity)
		}

		return OrderResponse{
			OrderID:           "ord-12345",
			TotalAmount:       total,
			EstimatedDelivery: "2023-05-15",
		}, nil
	}

	registry := NewFunctionRegistryImpl()
	registry.Register(FuncWrapper("createOrder", "", createOrderFn))

	toolCall := ToolCall{
		Function: &FunctionCall{
			Name: "createOrder",
			Arguments: `{
                "customer_id": "cust-789",
                "items": [
                    {"product_id": "prod-123", "quantity": 2, "unit_price": 29.99},
                    {"product_id": "prod-456", "quantity": 1, "unit_price": 49.99}
                ],
                "shipping_address": {
                    "street": "123 Main St",
                    "city": "Boston",
                    "country": "USA",
                    "zip": "02108"
                }
            }`,
		},
	}

	result, err := registry.Execute(context.Background(), toolCall)
	require.NoError(t, err)
	assert.JSONEq(t, `{"order_id":"ord-12345","total_amount":109.97,"estimated_delivery":"2023-05-15"}`, result)
}

func TestFuncWrapper_ComplexArgumentStructure(t *testing.T) {
	type NestedStruct struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	}

	type ComplexArgs struct {
		Name    string       `json:"name"`
		Enabled bool         `json:"enabled"`
		Nested  NestedStruct `json:"nested"`
		Tags    []string     `json:"tags"`
	}

	type ComplexResult struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	fn := func(ctx context.Context, args ComplexArgs) (ComplexResult, error) {
		message := fmt.Sprintf("Processed %s with %d tags", args.Name, len(args.Tags))
		return ComplexResult{Success: args.Enabled, Message: message}, nil
	}

	funcDefinition := FuncWrapper("testFn", "description", fn)

	args := map[string]interface{}{
		"name":    "test-item",
		"enabled": true,
		"nested":  map[string]interface{}{"value": "test-value", "count": 42},
		"tags":    []interface{}{"tag1", "tag2", "tag3"},
	}

	assert.Equal(t, "testFn", funcDefinition.Name)
	assert.Equal(t, "description", funcDefinition.Description)

	result, err := funcDefinition.Handler(context.Background(), args)
	require.NoError(t, err)

	resultJSON, _ := json.Marshal(result)
	assert.JSONEq(t, `{"success":true,"message":"Processed test-item with 3 tags"}`, string(resultJSON))
}

func TestFuncWrapper_MissingRequiredField(t *testing.T) {
	type RequiredArgs struct {
		Required int    `json:"required"`
		Optional string `json:"optional"`
	}

	fn := func(ctx context.Context, args RequiredArgs) (string, error) {
		return fmt.Sprintf("Got required=%d", args.Required), nil
	}

	funcDef := FuncWrapper("requiredTest", "Test required fields", fn)
	assert.Equal(t, "requiredTest", funcDef.Name)
	assert.Equal(t, "Test required fields", funcDef.Description)

	// Missing the required field
	args := map[string]interface{}{
		"optional": "test",
	}

	result, err := funcDef.Handler(context.Background(), args)
	// Go's json unmarshaler doesn't error on missing fields, it just uses zero values
	require.NoError(t, err)
	assert.Equal(t, "Got required=0", result)
}

func TestFuncWrapper_ErrorFromWrappedFunction(t *testing.T) {
	type Args struct {
		ShouldFail bool `json:"should_fail"`
	}

	fn := func(ctx context.Context, args Args) (interface{}, error) {
		if args.ShouldFail {
			return nil, errors.New("function failed as requested")
		}
		return "success", nil
	}

	funcDef := FuncWrapper("failableFunction", "A function that might fail", fn)

	args := map[string]interface{}{
		"should_fail": true,
	}

	result, err := funcDef.Handler(context.Background(), args)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "function failed as requested")
	assert.Nil(t, result)
}

func TestFuncWrapper_TypeMismatch(t *testing.T) {
	type StrictArgs struct {
		Count int `json:"count"`
	}

	fn := func(ctx context.Context, args StrictArgs) (string, error) {
		return fmt.Sprintf("Count: %d", args.Count), nil
	}

	funcDef := FuncWrapper("strictTypeFunction", "Function with strict type requirements", fn)

	// String value for int field will cause type conversion during unmarshaling
	args := map[string]interface{}{
		"count": "not-a-number",
	}

	_, err := funcDef.Handler(context.Background(), args)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal args")
}

func TestFuncWrapper_WithCustomTimeFormat(t *testing.T) {
	type TimeArgs struct {
		Timestamp time.Time `json:"timestamp"`
	}

	fn := func(ctx context.Context, args TimeArgs) (string, error) {
		return args.Timestamp.Format(time.RFC3339), nil
	}

	funcDef := FuncWrapper("timeFunction", "Function handling time values", fn)

	timeStr := "2023-04-15T12:30:45Z"
	args := map[string]interface{}{
		"timestamp": timeStr,
	}

	result, err := funcDef.Handler(context.Background(), args)
	require.NoError(t, err)
	assert.Equal(t, timeStr, result)
}

// Test MethodWrapper with a method that has a pointer receiver

type UserService struct {
	prefix string
}
type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	Country string `json:"country"`
}

type ComplexArgs struct {
	Name        string   `json:"name"`
	Age         int      `json:"age"`
	IsActive    bool     `json:"is_active"`
	Tags        []string `json:"tags"`
	HomeAddress Address  `json:"home_address"`
}

func (s *UserService) ProcessUser(ctx context.Context, args ComplexArgs) (string, error) {
	return fmt.Sprintf("%s: %s, %d, %v, %s, %s",
		s.prefix, args.Name, args.Age, args.IsActive,
		strings.Join(args.Tags, ","), args.HomeAddress.City), nil
}

func TestFuncWrapper_WithMethodReceiver(t *testing.T) {
	// Create a UserService instance with a prefix
	userService := &UserService{
		prefix: "User",
	}

	// Create the function wrapper using the ProcessUser method
	funcDef := FuncWrapper("processUser", "Process user information", userService.ProcessUser)

	// Verify function definition properties
	assert.Equal(t, "processUser", funcDef.Name)
	assert.Equal(t, "Process user information", funcDef.Description)

	// Create test arguments as a map
	args := map[string]interface{}{
		"name":      "John Doe",
		"age":       30,
		"is_active": true,
		"tags":      []interface{}{"customer", "premium"},
		"home_address": map[string]interface{}{
			"street":  "123 Main St",
			"city":    "Boston",
			"country": "USA",
		},
	}

	// Execute the handler
	result, err := funcDef.Handler(context.Background(), args)

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, "User: John Doe, 30, true, customer,premium, Boston", result)
}
