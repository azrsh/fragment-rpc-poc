package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

	"connectrpc.com/connect"
	"github.com/azrsh/fragment-colocation-with-grpc/internal/plan"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type cached struct {
	done  chan struct{}
	value any
	err   error
}
type ExecutionContext struct {
	context.Context
	mu    sync.Mutex
	cache map[string]*cached
}
type Resolver func(source, args map[string]any, ctx *ExecutionContext) (any, error)
type Resolvers map[string]Resolver

func (ctx *ExecutionContext) Once(key string, load func() (any, error)) (any, error) {
	ctx.mu.Lock()
	if entry, ok := ctx.cache[key]; ok {
		ctx.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-entry.done:
			return entry.value, entry.err
		}
	}
	entry := &cached{done: make(chan struct{})}
	ctx.cache[key] = entry
	ctx.mu.Unlock()
	entry.value, entry.err = load()
	close(entry.done)
	return entry.value, entry.err
}

func failure(code connect.Code, format string, args ...any) error {
	return connect.NewError(code, fmt.Errorf(format, args...))
}

func scalar(value any, typ, path string, code connect.Code) (any, error) {
	if s, ok := value.(string); ok && (typ == "String" || typ == "ID") {
		return s, nil
	}
	if b, ok := value.(bool); ok && typ == "Boolean" {
		return b, nil
	}
	var n float64
	numeric := true
	switch v := value.(type) {
	case float64:
		n = v
	case int:
		n = float64(v)
	case int32:
		n = float64(v)
	case int64:
		n = float64(v)
	default:
		numeric = false
	}
	if numeric && !math.IsNaN(n) && !math.IsInf(n, 0) {
		if typ == "Float" {
			return n, nil
		}
		if math.Trunc(n) == n {
			if typ == "ID" {
				return strconv.FormatFloat(n, 'f', -1, 64), nil
			}
			if typ == "Int" && n >= math.MinInt32 && n <= math.MaxInt32 {
				return n, nil
			}
		}
	}
	return nil, failure(code, "Invalid %s at %s", typ, path)
}

func Execute(ctx context.Context, p plan.Operation, request map[string]any, resolvers Resolvers) (map[string]any, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	execution := &ExecutionContext{Context: ctx, cache: map[string]*cached{}}
	variables := map[string]any{}
	for _, v := range p.Variables {
		value := request[v.Name]
		if value == nil {
			if v.Required {
				return nil, failure(connect.CodeInvalidArgument, "Missing variable $%s", v.Name)
			}
			continue
		}
		converted, err := scalar(value, v.Type, "$"+v.Name, connect.CodeInvalidArgument)
		if err != nil {
			return nil, err
		}
		variables[v.Name] = converted
	}
	var firstFailure sync.Once
	var executionError error
	failed := make(chan struct{})
	parallel := func(count int, work func(int) (any, error)) ([]any, error) {
		values := make([]any, count)
		var wg sync.WaitGroup
		for i := range count {
			wg.Add(1)
			go func() {
				defer wg.Done()
				value, err := work(i)
				if err != nil {
					firstFailure.Do(func() {
						executionError = err
						close(failed)
						cancel()
					})
				} else {
					values[i] = value
				}
			}()
		}
		wg.Wait()
		select {
		case <-failed:
			return nil, executionError
		default:
			return values, nil
		}
	}
	var project func(any, []plan.Field, string) (map[string]any, error)
	project = func(source any, fields []plan.Field, path string) (map[string]any, error) {
		record, ok := source.(map[string]any)
		if !ok || record == nil {
			return nil, failure(connect.CodeInternal, "Expected object at %s", path)
		}
		values, err := parallel(len(fields), func(i int) (any, error) {
			field := fields[i]
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			fieldPath := path + "." + field.ResponseName
			args := map[string]any{}
			for name, b := range field.Args {
				if b.Variable != "" {
					args[name] = variables[b.Variable]
				} else {
					args[name] = b.Literal
				}
			}
			resolver := resolvers[field.Coordinate]
			if resolver == nil && strings.HasPrefix(field.Coordinate, "Query.") {
				return nil, failure(connect.CodeInternal, "Missing resolver: %s", field.Coordinate)
			}
			value := record[field.Name]
			if resolver != nil {
				var err error
				value, err = resolver(record, args, execution)
				if err != nil {
					return nil, err
				}
			}
			if value == nil {
				if field.Required {
					return nil, failure(connect.CodeInternal, "Null for required field %s", fieldPath)
				}
				return nil, nil
			}
			item := func(value any, itemPath string) (any, error) {
				if value == nil {
					return nil, failure(connect.CodeInternal, "Null list item at %s", itemPath)
				}
				if field.Fields != nil {
					return project(value, field.Fields, itemPath)
				}
				return scalar(value, field.Type, itemPath, connect.CodeInternal)
			}
			if field.List {
				list, ok := value.([]any)
				if !ok {
					return nil, failure(connect.CodeInternal, "Expected list at %s", fieldPath)
				}
				return parallel(len(list), func(j int) (any, error) {
					return item(list[j], fmt.Sprintf("%s[%d]", fieldPath, j))
				})
			}
			return item(value, fieldPath)
		})
		if err != nil {
			return nil, err
		}
		out := map[string]any{}
		for i, f := range fields {
			out[f.ResponseName] = values[i]
		}
		return out, nil
	}
	return project(map[string]any{}, p.Fields, p.Name)
}

func ExecuteMessage(ctx context.Context, p plan.Operation, request, response proto.Message, resolvers Resolvers) error {
	data, err := protojson.Marshal(request)
	if err != nil {
		return failure(connect.CodeInternal, "encode request: %v", err)
	}
	var variables map[string]any
	if err = json.Unmarshal(data, &variables); err != nil {
		return failure(connect.CodeInternal, "decode request: %v", err)
	}
	output, err := Execute(ctx, p, variables, resolvers)
	if err != nil {
		return err
	}
	data, err = json.Marshal(output)
	if err != nil {
		return failure(connect.CodeInternal, "encode response: %v", err)
	}
	if err = protojson.Unmarshal(data, response); err != nil {
		return failure(connect.CodeInternal, "decode response: %v", err)
	}
	return nil
}
