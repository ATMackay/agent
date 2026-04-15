package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ATMackay/agent/state"
)

// ---- edit_file tests --------------------------------------------------------

func TestEditFile_ReplaceOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("hello world\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := newFakeToolContext(map[string]any{state.StateWorkDir: dir})
	result, err := newEditFileTool()(ctx, EditFileArgs{
		Path:      "file.txt",
		OldString: "world",
		NewString: "Go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Replaced != 1 {
		t.Errorf("replaced = %d, want 1", result.Replaced)
	}

	got, _ := os.ReadFile(path)
	if string(got) != "hello Go\n" {
		t.Errorf("file content = %q, want %q", got, "hello Go\n")
	}
}

func TestEditFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello world\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := newFakeToolContext(map[string]any{state.StateWorkDir: dir})
	_, err := newEditFileTool()(ctx, EditFileArgs{
		Path:      "file.txt",
		OldString: "notpresent",
		NewString: "x",
	})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got %v", err)
	}
}

func TestEditFile_AmbiguousWithoutReplaceAll(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("aa aa aa\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := newFakeToolContext(map[string]any{state.StateWorkDir: dir})
	_, err := newEditFileTool()(ctx, EditFileArgs{
		Path:      "file.txt",
		OldString: "aa",
		NewString: "bb",
	})
	if err == nil || !strings.Contains(err.Error(), "matches") {
		t.Errorf("expected ambiguous match error, got %v", err)
	}
}

func TestEditFile_ReplaceAll(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("aa aa aa\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := newFakeToolContext(map[string]any{state.StateWorkDir: dir})
	result, err := newEditFileTool()(ctx, EditFileArgs{
		Path:       "file.txt",
		OldString:  "aa",
		NewString:  "bb",
		ReplaceAll: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Replaced != 3 {
		t.Errorf("replaced = %d, want 3", result.Replaced)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "file.txt"))
	if string(got) != "bb bb bb\n" {
		t.Errorf("file content = %q, want %q", got, "bb bb bb\n")
	}
}

// ---- list_dir tests ---------------------------------------------------------

func TestListDir_Basic(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.go"), []byte("package sub"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := newFakeToolContext(map[string]any{state.StateWorkDir: dir})
	result, err := newListDirTool()(ctx, ListDirArgs{Path: "."})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EntryCount < 3 { // a.go, sub/, sub/b.go
		t.Errorf("entry_count = %d, want >= 3", result.EntryCount)
	}
}

func TestListDir_NotADirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := newFakeToolContext(map[string]any{state.StateWorkDir: dir})
	_, err := newListDirTool()(ctx, ListDirArgs{Path: "file.txt"})
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("expected 'not a directory' error, got %v", err)
	}
}

// ---- read_local_file tests --------------------------------------------------

func TestReadLocalFile_Snippet(t *testing.T) {
	dir := t.TempDir()
	lines := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := newFakeToolContext(map[string]any{state.StateWorkDir: dir})
	result, err := newReadLocalFileTool()(ctx, ReadLocalFileArgs{
		Path:      "f.txt",
		StartLine: 2,
		EndLine:   4,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StartLine != 2 || result.EndLine != 4 {
		t.Errorf("lines = %d-%d, want 2-4", result.StartLine, result.EndLine)
	}
	if !strings.Contains(result.Content, "line2") || !strings.Contains(result.Content, "line4") {
		t.Errorf("content missing expected lines: %q", result.Content)
	}
}

func TestReadLocalFile_MissingFile(t *testing.T) {
	dir := t.TempDir()

	ctx := newFakeToolContext(map[string]any{state.StateWorkDir: dir})
	_, err := newReadLocalFileTool()(ctx, ReadLocalFileArgs{Path: "nope.txt"})
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

// ---- exec_command tests -----------------------------------------------------

func TestExecCommand_Success(t *testing.T) {
	ctx := newFakeToolContext(nil)
	result, err := newExecCommandTool()(ctx, ExecCommandArgs{
		Command: "echo",
		Args:    []string{"hello"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit_code = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "hello") {
		t.Errorf("stdout = %q, want to contain 'hello'", result.Stdout)
	}
}

func TestExecCommand_NonZeroExit(t *testing.T) {
	ctx := newFakeToolContext(nil)
	result, err := newExecCommandTool()(ctx, ExecCommandArgs{
		Command: "sh",
		Args:    []string{"-c", "exit 42"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 42 {
		t.Errorf("exit_code = %d, want 42", result.ExitCode)
	}
}

func TestExecCommand_EmptyCommand(t *testing.T) {
	ctx := newFakeToolContext(nil)
	_, err := newExecCommandTool()(ctx, ExecCommandArgs{Command: ""})
	if err == nil || !strings.Contains(err.Error(), "command is required") {
		t.Errorf("expected 'command is required' error, got %v", err)
	}
}
