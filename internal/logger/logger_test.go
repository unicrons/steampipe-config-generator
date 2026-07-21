package logger

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestNew_DefaultFormatIsText(t *testing.T) {
	out := captureStderr(t, func() {
		log := New("default")
		log.Info("hello", "key", "value")
	})

	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("expected plain text output for \"default\", got what looks like JSON: %s", out)
	}
	if !strings.Contains(out, "msg=hello") {
		t.Errorf("expected slog text handler output (msg=hello), got: %s", out)
	}
}

func TestNew_JSONFormat(t *testing.T) {
	out := captureStderr(t, func() {
		log := New("json")
		log.Info("hello", "key", "value")
	})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("expected valid JSON output, got %q: %v", out, err)
	}
	if parsed["msg"] != "hello" {
		t.Errorf(`parsed["msg"] = %v, want "hello"`, parsed["msg"])
	}
}

func TestLevelFromEnv(t *testing.T) {
	tests := []struct {
		name   string
		envVal string
		want   slog.Level
	}{
		{name: "valid level", envVal: "DEBUG", want: slog.LevelDebug},
		{name: "valid level, different case", envVal: "warn", want: slog.LevelWarn},
		{name: "invalid value falls back to info", envVal: "bogus", want: slog.LevelInfo},
		{name: "unset falls back to info", envVal: "", want: slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", tt.envVal)
			if got := levelFromEnv(); got != tt.want {
				t.Errorf("levelFromEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

// captureStderr swaps os.Stderr for a pipe for the duration of fn, restoring it afterward, and
// returns everything fn wrote. logger.New writes to os.Stderr directly (there's no io.Writer
// injection point for a CLI-only logger), so this is the only way to observe its output.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}

	original := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = original }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe writer: %v", err)
	}

	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return strings.Join(lines, "\n")
}
