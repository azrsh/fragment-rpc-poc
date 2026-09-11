package demo_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/azrsh/fragment-colocation-with-grpc/generated/appv1"
	"github.com/azrsh/fragment-colocation-with-grpc/generated/appv1/appv1connect"
	"github.com/azrsh/fragment-colocation-with-grpc/internal/compiler"
	"github.com/azrsh/fragment-colocation-with-grpc/internal/network"
	"google.golang.org/protobuf/proto"
)

func TestFragmentEditChangesStandaloneRPCAndPreservesOldClient(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, entry := range []string{"app", "proto", "schema.graphql", "go.mod", "go.sum", "internal", "cmd/server", "generated/schema.lock.json"} {
		err := filepath.WalkDir(filepath.Join(root, entry), func(path string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if e.IsDir() {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			target := filepath.Join(dir, relative)
			if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			return os.WriteFile(target, data, 0644)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if err = os.Symlink(filepath.Join(root, "node_modules"), filepath.Join(dir, "node_modules")); err != nil {
		t.Fatal(err)
	}
	fragment := filepath.Join(dir, "app/components/UserCard/UserCard.graphql")
	data, err := os.ReadFile(fragment)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(fragment, []byte(strings.Replace(string(data), "  name\n", "  name\n  email\n", 1)), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = compiler.Generate(dir); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(filepath.Join(dir, "generated/app.proto"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("string email = 5;")) {
		t.Fatalf("missing new field: %s", data)
	}
	data, err = os.ReadFile(filepath.Join(dir, "generated/fragments.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("email: string")) {
		t.Fatal("missing fragment field")
	}
	deps := exec.Command("go", "list", "-deps", "./cmd/server")
	deps.Dir = dir
	data, err = deps.CombinedOutput()
	if err != nil {
		t.Fatalf("dependencies: %v: %s", err, data)
	}
	if bytes.Contains(data, []byte("gqlparser")) || bytes.Contains(data, []byte("/internal/compiler")) {
		t.Fatal("runtime imports GraphQL compiler")
	}
	runtimeDir := t.TempDir()
	binary := filepath.Join(runtimeDir, "fragment-rpc")
	build := exec.Command("go", "build", "-o", binary, "./cmd/server")
	build.Dir = dir
	if data, err = build.CombinedOutput(); err != nil {
		t.Fatalf("build evolved server: %v: %s", err, data)
	}
	// The executable must serve the new contract after all generation inputs are gone.
	if err = os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary)
	cmd.Dir = runtimeDir
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "PATH=") && !strings.HasPrefix(env, "PORT=") && !strings.HasPrefix(env, "GRPC_PORT=") {
			cmd.Env = append(cmd.Env, env)
		}
	}
	cmd.Env = append(cmd.Env, "PATH="+runtimeDir, "PORT=0", "GRPC_PORT=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	lines := make(chan string, 8)
	scanDone := make(chan struct{})
	go func() {
		defer close(lines)
		defer close(scanDone)
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Demo:") || strings.HasPrefix(line, "Generated gRPC API:") {
				lines <- line
			}
		}
	}()
	t.Cleanup(func() {
		cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("server exit: %v: %s", err, stderr.String())
			}
		case <-time.After(5 * time.Second):
			cmd.Process.Kill()
			<-done
			t.Error("server did not stop")
		}
		<-scanDone
	})
	webURL, grpcURL := "", ""
	timeout := time.After(10 * time.Second)
	for webURL == "" || grpcURL == "" {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatal("server exited before startup")
			}
			if strings.HasPrefix(line, "Demo: ") {
				webURL = strings.TrimPrefix(line, "Demo: ")
			}
			if strings.HasPrefix(line, "Generated gRPC API: ") {
				grpcURL = strings.TrimSuffix(strings.TrimPrefix(line, "Generated gRPC API: "), "/app.v1.AppService")
			}
		case <-timeout:
			t.Fatal("server startup timed out")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webURL+"/app.v1.AppService/GetUserPage", strings.NewReader(`{"id":"u1"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal(response.Status)
	}
	var payload struct{ User struct{ Email string } }
	if err = json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.User.Email != "aki@example.test" {
		t.Fatalf("evolved response: %+v", payload)
	}
	httpClient := network.NewGRPCClient()
	defer httpClient.CloseIdleConnections()
	oldClient := appv1connect.NewAppServiceClient(httpClient, grpcURL, connect.WithGRPC())
	old, err := oldClient.GetUserPage(ctx, connect.NewRequest(&appv1.GetUserPageRequest{Id: proto.String("u1")}))
	if err != nil {
		t.Fatal(err)
	}
	if old.Msg.GetUser().GetName() != "Aki Tanaka" || old.Msg.GetUser().GetOrganization().GetName() != "Northstar Studio" {
		t.Fatal(old.Msg)
	}
	if old.Msg.User.ProtoReflect().Descriptor().Fields().ByName("email") != nil {
		t.Fatal("old client unexpectedly has new field")
	}
}
