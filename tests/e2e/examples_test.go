package e2e_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func runGo(t *testing.T, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		t.Fatalf("go %s failed: %v\n%s", strings.Join(args, " "), err, out.String())
	}
	return out.String()
}

func buildExample(t *testing.T, example string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), example)
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	runGo(t, "build", "-o", bin, "./examples/"+example)
	return bin
}

func runBinary(t *testing.T, bin string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %s failed: %v\n%s", bin, strings.Join(args, " "), err, out.String())
	}
	return out.String()
}

func runBinaryWithSignal(t *testing.T, bin string, args ...string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("signal process group e2e uses POSIX semantics")
	}

	cmd := exec.Command(bin, args...)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Start(); err != nil {
		t.Fatalf("start %s: %v", bin, err)
	}

	timer := time.NewTimer(2 * time.Second)
	<-timer.C
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGINT); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("send SIGINT to %s: %v", bin, err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("%s did not exit cleanly after SIGINT: %v\n%s", bin, err, out.String())
		}
	case <-time.After(20 * time.Second):
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		t.Fatalf("%s timed out after SIGINT\n%s", bin, out.String())
	}

	return out.String()
}

func assertContains(t *testing.T, out string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(out, part) {
			t.Fatalf("output missing %q\n%s", part, out)
		}
	}
}

func assertNotContains(t *testing.T, out string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if strings.Contains(out, part) {
			t.Fatalf("output should not contain %q\n%s", part, out)
		}
	}
}

func TestExamplesE2E_CliTool(t *testing.T) {
	bin := buildExample(t, "cli-tool")

	help := runBinary(t, bin, "--help")
	assertContains(t, help, "Recommended CLI-only example", "status", "echo", "--profile")

	status := runBinary(t, bin, "status", "--profile", "prod")
	assertContains(t, status, "init hook completed", "profile: prod", "command:")

	echo := runBinary(t, bin, "echo", "hello", "world")
	assertContains(t, echo, "hello world")
}

func TestExamplesE2E_FlagExport(t *testing.T) {
	bin := buildExample(t, "flag-export")
	out := runBinary(t, bin, "inspect", "--region", "eu-west-1", "--output", "yaml", "--verbose")
	assertContains(t, out, "system flags (23)", "bindable flag names", "region = eu-west-1")
}

func TestExamplesE2E_ServiceConfig(t *testing.T) {
	bin := buildExample(t, "service-config")
	out := runBinary(t, bin, "inspect")
	assertContains(t, out, `"name": "payments-agent"`, `"stop_timeout": "30s"`, `"AllowSudoFallback": true`)
}

func TestExamplesE2E_GracefulSignalShutdown(t *testing.T) {
	tests := []struct {
		name         string
		want         []string
		forbidden    []string
		cleanupsOnce bool
	}{
		{
			name:      "function-service",
			want:      []string{"mailer worker started", "mailer worker received shutdown signal", "mailer worker stopped cleanly"},
			forbidden: []string{"Error:", "Usage:", "context canceled"},
		},
		{
			name:      "service-runner",
			want:      []string{"queue worker started", "context cancelled, worker exiting", "queue worker stop hook completed"},
			forbidden: []string{"Error:", "Usage:", "context canceled"},
		},
		{
			name:         "complete",
			want:         []string{"服务启动", "收到停止信号，优雅退出中", "生命周期: BeforeStop", "生命周期: AfterStop"},
			forbidden:    []string{"Error:", "Usage:", "context canceled", "命令执行失败"},
			cleanupsOnce: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin := buildExample(t, tt.name)
			out := runBinaryWithSignal(t, bin)
			assertContains(t, out, tt.want...)
			assertNotContains(t, out, tt.forbidden...)
			if tt.cleanupsOnce && strings.Count(out, "服务清理完成") != 1 {
				t.Fatalf("expected service cleanup to run once\n%s", out)
			}
		})
	}
}
