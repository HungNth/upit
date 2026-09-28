package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/HungNth/upit/internal/app"
	"github.com/spf13/pflag"
)

const helpText = `Usage:
  upit upload [flags] <file>

Flags:
  --uploader <name>   use a named Uploader instead of the configured default
  --json              write machine-readable output
  --clipboard         copy the Final URL after a successful upload
  --no-clipboard      disable clipboard copying for this invocation
  --timeout <duration> cancel after a Go duration such as 30s or 10m (default: no deadline)
  -h, --help          show this help
`

type Runner struct {
	HomeDir   func() (string, error)
	Clipboard app.Clipboard
}

func (r Runner) Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if helpRequested(args) {
		if _, err := io.WriteString(stdout, helpText); err != nil {
			fmt.Fprintf(stderr, "write help: %v\n", err)
			return 1
		}
		return 0
	}

	options, err := parseUploadArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		fmt.Fprintln(stderr, "usage: upit upload [flags] <file>")
		return 2
	}

	runContext := ctx
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		runContext, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	clipboardOverride := app.ClipboardFromConfig
	if options.Clipboard {
		clipboardOverride = app.ClipboardEnabled
	} else if options.NoClipboard {
		clipboardOverride = app.ClipboardDisabled
	}
	outcome, err := (app.Service{HomeDir: r.HomeDir, Clipboard: r.Clipboard}).Upload(runContext, app.UploadOptions{
		FilePath:  options.FilePath,
		Uploader:  options.Uploader,
		Clipboard: clipboardOverride,
	})
	if err != nil {
		failure := appFailure(err)
		if options.JSON {
			encoder := json.NewEncoder(stderr)
			encoder.SetEscapeHTML(false)
			_ = encoder.Encode(jsonFailure{
				Success:    false,
				Stage:      failure.Stage,
				Message:    failure.Message,
				StatusCode: failure.StatusCode,
			})
		} else {
			fmt.Fprintln(stderr, "Upload failed")
			fmt.Fprintf(stderr, "Stage: %s\n", failure.Stage)
			if failure.StatusCode != 0 {
				fmt.Fprintf(stderr, "HTTP: %d\n", failure.StatusCode)
			}
			fmt.Fprintf(stderr, "Message: %s\n", failure.Message)
		}
		if errors.Is(err, context.Canceled) && errors.Is(ctx.Err(), context.Canceled) {
			return 130
		}
		return 1
	}

	if options.JSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(jsonSuccess{
			Success:     true,
			OriginalURL: outcome.Result.OriginalURL,
			FinalURL:    outcome.Result.FinalURL,
		}); err != nil {
			fmt.Fprintf(stderr, "write result: %v\n", err)
			return 1
		}
	} else if _, err := fmt.Fprintln(stdout, outcome.Result.FinalURL); err != nil {
		fmt.Fprintf(stderr, "write result: %v\n", err)
		return 1
	}
	for _, warning := range outcome.Warnings {
		fmt.Fprintf(stderr, "Warning: %s\n", warning)
	}
	return 0
}

func helpRequested(args []string) bool {
	if len(args) == 1 {
		switch args[0] {
		case "help", "-h", "--help":
			return true
		}
	}
	if len(args) == 2 && args[0] == "help" && args[1] == "upload" {
		return true
	}
	return len(args) > 1 && args[0] == "upload" && (slices.Contains(args[1:], "-h") || slices.Contains(args[1:], "--help"))
}

type jsonSuccess struct {
	Success     bool   `json:"success"`
	OriginalURL string `json:"originalUrl"`
	FinalURL    string `json:"finalUrl"`
}

type jsonFailure struct {
	Success    bool   `json:"success"`
	Stage      string `json:"stage"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode,omitempty"`
}

func appFailure(err error) *app.Failure {
	var failure *app.Failure
	if errors.As(err, &failure) {
		return failure
	}
	return &app.Failure{Stage: "request", Message: err.Error(), Cause: err}
}

type uploadOptions struct {
	FilePath    string
	Uploader    string
	JSON        bool
	Timeout     time.Duration
	Clipboard   bool
	NoClipboard bool
}

func parseUploadArgs(args []string) (uploadOptions, error) {
	if len(args) == 0 || args[0] != "upload" {
		return uploadOptions{}, fmt.Errorf("unknown command")
	}
	var options uploadOptions
	flags := pflag.NewFlagSet("upload", pflag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.SetInterspersed(true)
	flags.StringVar(&options.Uploader, "uploader", "", "named uploader")
	flags.BoolVar(&options.JSON, "json", false, "write JSON output")
	flags.DurationVar(&options.Timeout, "timeout", 0, "upload timeout")
	flags.BoolVar(&options.Clipboard, "clipboard", false, "copy the final URL")
	flags.BoolVar(&options.NoClipboard, "no-clipboard", false, "do not copy the final URL")
	if err := flags.Parse(args[1:]); err != nil {
		return uploadOptions{}, err
	}
	if options.Timeout < 0 {
		return uploadOptions{}, fmt.Errorf("timeout must not be negative")
	}
	if options.Clipboard && options.NoClipboard {
		return uploadOptions{}, fmt.Errorf("--clipboard and --no-clipboard cannot be used together")
	}
	if flags.NArg() != 1 {
		return uploadOptions{}, fmt.Errorf("upload requires exactly one file")
	}
	options.FilePath = flags.Arg(0)
	return options, nil
}
