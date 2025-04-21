package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/invopop/jsonschema"
	"sync"
)

// FuncDefinition represents a function definition for a tool
type FuncDefinition struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Parameters  *jsonschema.Schema `json:"parameters"`
	Handler     FuncHandler
}

// FuncHandler is the standard tool handler signature
type FuncHandler func(ctx context.Context, args map[string]interface{}) (interface{}, error)

type FuncHandlerV2[T any, R any] func(ctx context.Context, args T) (R, error)

type FuncRegistry interface {
	Register(fn FuncDefinition)
	Execute(ctx context.Context, toolCall ToolCall) (string, error)
	GetFuncDefinitions() []FuncDefinition
}

// FuncRegistryImpl manages tool handlers
type FuncRegistryImpl struct {
	tools map[string]FuncDefinition
	mu    sync.RWMutex
}

func NewFunctionRegistryImpl() *FuncRegistryImpl {
	// Initialize the tool registry
	return &FuncRegistryImpl{tools: make(map[string]FuncDefinition)}
}

// Register registers a function with the given name and definition. Note that
func (r *FuncRegistryImpl) Register(fn FuncDefinition) {
	// Lock to avoid case we want to parallelize the registration in the future
	r.mu.Lock()
	r.tools[fn.Name] = fn
	r.mu.Unlock()
}

func (r *FuncRegistryImpl) Execute(ctx context.Context, toolCall ToolCall) (string, error) {
	// All the tools need to be registered when the app starts up
	// So we don't need to lock here
	fd, exists := r.tools[toolCall.Function.Name]
	if !exists {
		return "", fmt.Errorf("unknown tool: %s", toolCall.Function.Name)
	}
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}
	result, err := fd.Handler(ctx, args)
	if err != nil {
		return "", err
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to serialize result: %v", err)
	}
	return string(resultJSON), nil
}

func (r *FuncRegistryImpl) GetFuncDefinitions() []FuncDefinition {
	funcDefs := make([]FuncDefinition, 0, len(r.tools))
	for _, fd := range r.tools {
		funcDefs = append(funcDefs, fd)
	}
	return funcDefs
}

// FuncWrapper adapts a typed function into a FuncDefinition with inferred schema
// Currently,it only supports functions with a single argument with type struct (DTO)
func FuncWrapper[T any, R any](name string, description string, fn func(ctx context.Context, args T) (R, error)) FuncDefinition {
	// Create a schema reflector for parameter type T
	reflector := jsonschema.Reflector{
		ExpandedStruct: true,
		DoNotReference: true,
	}

	// Infer JSON schema from the type T
	schema := reflector.Reflect(new(T))

	// Create the handler function that adapts the typed function
	handler := func(ctx context.Context, innerArgs map[string]interface{}) (interface{}, error) {
		argsJSON, err := json.Marshal(innerArgs)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal args: %v", err)
		}
		var typedArgs T
		if err := json.Unmarshal(argsJSON, &typedArgs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal args: %v", err)
		}
		return fn(ctx, typedArgs)
	}

	// Return the complete FuncDefinition
	return FuncDefinition{
		Name:        name,
		Description: description,
		Parameters:  schema,
		Handler:     handler,
	}
}
