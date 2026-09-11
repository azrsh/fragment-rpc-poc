package gateway_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/azrsh/fragment-colocation-with-grpc/generated"
	"github.com/azrsh/fragment-colocation-with-grpc/generated/appv1"
	"github.com/azrsh/fragment-colocation-with-grpc/generated/appv1/appv1connect"
	"github.com/azrsh/fragment-colocation-with-grpc/internal/compiler"
	"github.com/azrsh/fragment-colocation-with-grpc/internal/demo"
	"github.com/azrsh/fragment-colocation-with-grpc/internal/gateway"
	"github.com/azrsh/fragment-colocation-with-grpc/internal/network"
	"github.com/azrsh/fragment-colocation-with-grpc/internal/plan"
	"google.golang.org/protobuf/proto"
)

func makePlan(t *testing.T, sdl, body string) plan.Operation {
	t.Helper()
	out, err := compiler.Compile(sdl, []compiler.SourceFile{{Name: "test.graphql", Body: body}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return out.Operations[0]
}
func clientFor(t *testing.T, resolvers gateway.Resolvers) appv1connect.AppServiceClient {
	t.Helper()
	path, handler, err := generated.NewHandler(resolvers)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	s, err := network.Serve(mux, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	httpClient := network.NewGRPCClient()
	t.Cleanup(httpClient.CloseIdleConnections)
	return appv1connect.NewAppServiceClient(httpClient, s.URL, connect.WithGRPC())
}
func request(id string) *connect.Request[appv1.GetUserPageRequest] {
	return connect.NewRequest(&appv1.GetUserPageRequest{Id: proto.String(id)})
}

func TestGRPCProjectionAndDeduplication(t *testing.T) {
	var mu sync.Mutex
	calls := []string{}
	backends, err := demo.StartBackends(func(service, id string) { mu.Lock(); defer mu.Unlock(); calls = append(calls, service+":"+id) })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { backends.Close() })
	httpClient := network.NewGRPCClient()
	t.Cleanup(httpClient.CloseIdleConnections)
	resolvers := gateway.NewResolvers(httpClient, backends.User.URL, backends.Organization.URL)
	client := clientFor(t, resolvers)
	checkCalls := func(expected ...string) {
		t.Helper()
		mu.Lock()
		defer mu.Unlock()
		if !reflect.DeepEqual(calls, expected) && !(len(calls) == 0 && len(expected) == 0) {
			t.Fatalf("calls: %v, expected: %v", calls, expected)
		}
		calls = nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	page, err := client.GetUserPage(ctx, request("u1"))
	if err != nil {
		t.Fatal(err)
	}
	if page.Msg.GetUser().GetName() != "Aki Tanaka" || page.Msg.GetUser().GetOrganization().GetName() != "Northstar Studio" {
		t.Fatal(page.Msg)
	}
	if page.Msg.User.ProtoReflect().Descriptor().Fields().ByName("email") != nil {
		t.Fatal("leaked unselected email")
	}
	checkCalls("UserService.GetUser:u1", "OrganizationService.GetOrganization:o1")
	summary, err := client.GetUserSummary(ctx, connect.NewRequest(&appv1.GetUserSummaryRequest{Id: proto.String("u2")}))
	if err != nil {
		t.Fatal(err)
	}
	if summary.Msg.GetUser().GetName() != "Ren Sato" {
		t.Fatal(summary.Msg)
	}
	checkCalls("UserService.GetUser:u2")
	page, err = client.GetUserPage(ctx, request("u3"))
	if err != nil {
		t.Fatal(err)
	}
	if page.Msg.User == nil || page.Msg.User.Organization != nil {
		t.Fatal(page.Msg)
	}
	checkCalls("UserService.GetUser:u3")
	page, err = client.GetUserPage(ctx, request("missing"))
	if err != nil {
		t.Fatal(err)
	}
	if page.Msg.User != nil {
		t.Fatal(page.Msg)
	}
	checkCalls("UserService.GetUser:missing")
	_, err = client.GetUserPage(ctx, connect.NewRequest(&appv1.GetUserPageRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid argument: %v", err)
	}
	checkCalls()
	sdl := `type Query{user(id:ID!):User} type User{id:ID! name:String! organization:Organization} type Organization{name:String! website:String!}`
	p := makePlan(t, sdl, `query Duplicate($id:ID!){first:user(id:$id){name organization{name}} second:user(id:$id){id organization{website}}}`)
	_, err = gateway.Execute(ctx, p, map[string]any{"id": "u1"}, resolvers)
	if err != nil {
		t.Fatal(err)
	}
	checkCalls("UserService.GetUser:u1", "OrganizationService.GetOrganization:o1")
	_, err = gateway.Execute(ctx, p, map[string]any{"id": "u1"}, resolvers)
	if err != nil {
		t.Fatal(err)
	}
	checkCalls("UserService.GetUser:u1", "OrganizationService.GetOrganization:o1")
}

func TestNumericIDLiteralAfterPlanDecoding(t *testing.T) {
	called := make(chan string, 1)
	b, err := demo.StartBackends(func(service, id string) { called <- id })
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	client := network.NewGRPCClient()
	defer client.CloseIdleConnections()
	p := makePlan(t, `type Query{user(id:ID!):User} type User{name:String!}`, `query Numeric{user(id:1234567){name}}`)
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	_, err = gateway.Execute(context.Background(), p, nil, gateway.NewResolvers(client, b.User.URL, b.Organization.URL))
	if err != nil {
		t.Fatal(err)
	}
	if id := <-called; id != "1234567" {
		t.Fatalf("numeric ID changed to %q", id)
	}
}

func TestFailureCancelsSiblingAndPreservesOriginalError(t *testing.T) {
	p := makePlan(t, `type Query{broken:String waiting:String}`, `query Fail{broken waiting}`)
	started, cancelled := make(chan struct{}), make(chan struct{})
	wanted := connect.NewError(connect.CodeUnavailable, errors.New("backend failed"))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := gateway.Execute(ctx, p, nil, gateway.Resolvers{
		"Query.broken": func(_, _ map[string]any, ctx *gateway.ExecutionContext) (any, error) {
			select {
			case <-started:
				return nil, wanted
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
		"Query.waiting": func(_, _ map[string]any, ctx *gateway.ExecutionContext) (any, error) {
			close(started)
			<-ctx.Done()
			close(cancelled)
			return nil, ctx.Err()
		},
	})
	if !errors.Is(err, wanted) {
		t.Fatalf("lost original failure: %v", err)
	}
	if ctx.Err() != nil {
		t.Fatal("waited for caller deadline instead of cancelling sibling")
	}
	select {
	case <-cancelled:
	default:
		t.Fatal("sibling was not cancelled")
	}
}

func TestListItemResolversRunConcurrently(t *testing.T) {
	p := makePlan(t, `type Query{items:[Item!]!} type Item{value:String!}`, `query Items{items{value}}`)
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() {
		for range 2 {
			select {
			case <-started:
			case <-ctx.Done():
				return
			}
		}
		close(release)
	}()
	out, err := gateway.Execute(ctx, p, nil, gateway.Resolvers{
		"Query.items": func(_, _ map[string]any, _ *gateway.ExecutionContext) (any, error) {
			return []any{map[string]any{"value": "first"}, map[string]any{"value": "second"}}, nil
		},
		"Item.value": func(source, _ map[string]any, ctx *gateway.ExecutionContext) (any, error) {
			started <- struct{}{}
			select {
			case <-release:
				return source["value"], nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"items": []any{map[string]any{"value": "first"}, map[string]any{"value": "second"}}}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("list order changed: %#v", out)
	}
}

func TestRuntimeValidationAndPresence(t *testing.T) {
	p := makePlan(t, `type Query{item(enabled:Boolean!,count:Int!):Item} type Item{enabled:Boolean! count:Int! name:String}`, `query GetItem($enabled:Boolean!,$count:Int!){item(enabled:$enabled,count:$count){enabled count name}}`)
	args := map[string]any{"enabled": false, "count": 0}
	output, err := gateway.Execute(context.Background(), p, args, gateway.Resolvers{"Query.item": func(_, args map[string]any, _ *gateway.ExecutionContext) (any, error) { return args, nil }})
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]any{"item": map[string]any{"count": float64(0), "enabled": false, "name": nil}}
	if !reflect.DeepEqual(output, expected) {
		t.Fatalf("presence: %#v", output)
	}
	_, err = gateway.Execute(context.Background(), p, args, gateway.Resolvers{"Query.item": func(_, _ map[string]any, _ *gateway.ExecutionContext) (any, error) {
		return map[string]any{"count": 0}, nil
	}})
	if err == nil || !strings.Contains(err.Error(), "Null for required field") {
		t.Fatal(err)
	}
	args["count"] = 1.5
	_, err = gateway.Execute(context.Background(), p, args, nil)
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatal(err)
	}
	args["count"] = 0
	_, err = gateway.Execute(context.Background(), p, args, nil)
	if err == nil || !strings.Contains(err.Error(), "Missing resolver") {
		t.Fatal(err)
	}
	list := makePlan(t, `type Query{values:[Int!]!}`, `query List{values}`)
	for _, value := range []any{[]any{0, 1}, []any{}, []any{nil}, []any{1.5}, "not a list", nil} {
		_, err := gateway.Execute(context.Background(), list, nil, gateway.Resolvers{"Query.values": func(_, _ map[string]any, _ *gateway.ExecutionContext) (any, error) { return value, nil }})
		valid := reflect.DeepEqual(value, []any{0, 1}) || reflect.DeepEqual(value, []any{})
		if (err == nil) != valid {
			t.Fatalf("list %#v: %v", value, err)
		}
	}
}

func TestDownstreamFailureIsIsolated(t *testing.T) {
	b, err := demo.StartBackends(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { b.Close() })
	httpClient := network.NewGRPCClient()
	t.Cleanup(httpClient.CloseIdleConnections)
	client := clientFor(t, gateway.NewResolvers(httpClient, b.User.URL, b.Organization.URL))
	b.Organization.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = client.GetUserPage(ctx, request("u1"))
	if code := connect.CodeOf(err); code != connect.CodeUnavailable && code != connect.CodeDeadlineExceeded {
		t.Fatalf("unexpected failure: %v", err)
	}
	response, err := client.GetUserSummary(ctx, connect.NewRequest(&appv1.GetUserSummaryRequest{Id: proto.String("u1")}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetUser().GetName() != "Aki Tanaka" {
		t.Fatal(response.Msg)
	}
}

func TestCancellationReachesResolver(t *testing.T) {
	started, cancelled := make(chan struct{}), make(chan struct{})
	client := clientFor(t, gateway.Resolvers{"Query.user": func(_, _ map[string]any, ctx *gateway.ExecutionContext) (any, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return nil, ctx.Err()
	}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { _, err := client.GetUserPage(ctx, request("u1")); result <- err }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("resolver never started")
	}
	cancel()
	select {
	case err := <-result:
		if connect.CodeOf(err) != connect.CodeCanceled && !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("client did not cancel")
	}
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("resolver did not cancel")
	}
}
