package compiler

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

func CollectSources(root string, roots []string) ([]SourceFile, error) {
	if roots == nil {
		roots = []string{"app", "mobile/ios/Sources", "mobile/android/app/src/main"}
	}
	var files []string
	for _, dir := range roots {
		base := dir
		if !filepath.IsAbs(base) {
			base = filepath.Join(root, dir)
		}
		if _, err := os.Stat(base); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(base, func(path string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !e.IsDir() {
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				files = append(files, filepath.ToSlash(rel))
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sources := []SourceFile{}
	for _, name := range files {
		if filepath.Ext(name) == ".graphql" {
			body, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				return nil, err
			}
			sources = append(sources, SourceFile{Name: name, Body: string(body)})
		}
	}
	for _, tool := range []struct{ ext, executable string }{
		{".swift", "mobile/swift-graphql/.build/debug/extract-graphql"},
		{".kt", "mobile/android/graphql-tools/build/install/graphql-tools/bin/graphql-tools"},
	} {
		var native []string
		for _, name := range files {
			if filepath.Ext(name) == tool.ext {
				native = append(native, name)
			}
		}
		if len(native) == 0 {
			continue
		}
		executable, err := filepath.Abs(filepath.Join(root, tool.executable))
		if err != nil {
			return nil, err
		}
		if _, err = os.Stat(executable); err != nil {
			return nil, fmt.Errorf("Run bash scripts/build-native-tools.sh before generating native contracts: %w", err)
		}
		cmd := exec.Command(executable, native...)
		cmd.Dir = root
		cmd.Env = os.Environ()
		if os.Getenv("JAVA_HOME") == "" {
			home, err := filepath.Abs(filepath.Join(root, ".local/mobile-tools/jdk/Contents/Home"))
			if err != nil {
				return nil, err
			}
			cmd.Env = append(cmd.Env, "JAVA_HOME="+home)
		}
		data, err := cmd.Output()
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				return nil, fmt.Errorf("extract %s: %s", tool.ext, e.Stderr)
			}
			return nil, err
		}
		var extracted []SourceFile
		if err = json.Unmarshal(data, &extracted); err != nil {
			return nil, fmt.Errorf("decode %s extraction: %w", tool.ext, err)
		}
		sources = append(sources, extracted...)
	}
	return sources, nil
}
