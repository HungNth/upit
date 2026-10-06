package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/HungNth/upit/internal/cli"
)

func TestVersionCommandsAndFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
		isJSON     bool
	}{
		{
			name:     "version subcommand",
			args:     []string{"version"},
			wantCode: 0,
		},
		{
			name:     "long version flag",
			args:     []string{"--version"},
			wantCode: 0,
		},
		{
			name:     "short version flag",
			args:     []string{"-v"},
			wantCode: 0,
		},
		{
			name:     "version subcommand with json flag",
			args:     []string{"version", "--json"},
			wantCode: 0,
			isJSON:   true,
		},
		{
			name:     "long version flag with json flag",
			args:     []string{"--version", "--json"},
			wantCode: 0,
			isJSON:   true,
		},
		{
			name:     "short version flag with json flag",
			args:     []string{"-v", "--json"},
			wantCode: 0,
			isJSON:   true,
		},
		{
			name:       "version with unexpected positional argument",
			args:       []string{"version", "extra"},
			wantCode:   2,
			wantStderr: "unexpected argument",
		},
		{
			name:       "long version flag with unexpected positional argument",
			args:       []string{"--version", "extra"},
			wantCode:   2,
			wantStderr: "unexpected argument",
		},
		{
			name:       "version with unknown flag",
			args:       []string{"version", "--unknown"},
			wantCode:   2,
			wantStderr: "unknown flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			runner := cli.Runner{}
			code := runner.Run(context.Background(), tt.args, &stdout, &stderr)

			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d; stderr: %q", code, tt.wantCode, stderr.String())
			}

			if tt.wantCode == 0 {
				if stderr.Len() != 0 {
					t.Errorf("stderr = %q, want empty", stderr.String())
				}

				if tt.isJSON {
					var decoded map[string]string
					if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
						t.Fatalf("failed to decode JSON output: %v; raw: %q", err, stdout.String())
					}
					if decoded["version"] != "0.9.0" {
						t.Errorf("JSON version = %q, want %q", decoded["version"], "0.9.0")
					}
					for _, key := range []string{"commit", "date", "goVersion", "os", "arch"} {
						if val, ok := decoded[key]; !ok || val == "" {
							t.Errorf("JSON missing or empty field %q in %s", key, stdout.String())
						}
					}
				} else {
					out := stdout.String()
					if !strings.HasPrefix(out, "upit 0.9.0 (commit: ") || !strings.Contains(out, ", built: ") {
						t.Errorf("stdout = %q, want version output starting with 'upit 0.9.0 (commit: '", out)
					}
				}
			} else {
				if stdout.Len() != 0 {
					t.Errorf("stdout = %q, want empty on error", stdout.String())
				}
				if tt.wantStderr != "" && !strings.Contains(stderr.String(), tt.wantStderr) {
					t.Errorf("stderr = %q, want to contain %q", stderr.String(), tt.wantStderr)
				}
			}
		})
	}
}
