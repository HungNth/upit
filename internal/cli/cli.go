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
  --uploader <name>    use a named Uploader instead of the configured default
  --shortener <name>   shorten the uploaded URL with a named Shortener
  --no-shorten         disable URL shortening for this invocation
  --json               write machine-readable output
  --clipboard          copy the Final URL after a successful upload
  --no-clipboard       disable clipboard copying for this invocation
  --timeout <duration> cancel after a Go duration such as 30s or 10m (default: no deadline)
  -h, --help           show this help
`

const globalHelpText = `Usage:
  upit upload [flags] <file>
  upit config <command>

Commands:
  upload  upload one file
  config  inspect and validate the Configuration Set

` + helpText

const configHelpText = `Usage:
  upit config <command>

Commands:
  path                         print the Configuration Set directory
  show                         show Global Configuration values
  list-uploaders               list configured Uploaders
  list-shorteners              list configured Shorteners
  validate                     validate the complete Configuration Set
  set-default-uploader <name>  select the default Uploader
  set-default-shortener <name> select the default Shortener
  clear-default-shortener      clear the default Shortener
  enable-clipboard             enable default clipboard copying
  disable-clipboard            disable default clipboard copying

  -h, --help                   show this help
`

type Runner struct {
	HomeDir   func() (string, error)
	Clipboard app.Clipboard
}

func (r Runner) Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if help, ok := helpFor(args); ok {
		if _, err := io.WriteString(stdout, help); err != nil {
			fmt.Fprintf(stderr, "write help: %v\n", err)
			return 1
		}
		return 0
	}
	if len(args) > 0 && args[0] == "config" {
		return r.runConfig(args[1:], stdout, stderr)
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
		FilePath:          options.FilePath,
		Uploader:          options.Uploader,
		Shortener:         options.Shortener,
		DisableShortening: options.NoShorten,
		Clipboard:         clipboardOverride,
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

func helpFor(args []string) (string, bool) {
	if len(args) == 1 {
		switch args[0] {
		case "help", "-h", "--help":
			return globalHelpText, true
		}
	}
	if len(args) > 1 {
		switch args[0] {
		case "upload":
			if slices.Contains(args[1:], "-h") || slices.Contains(args[1:], "--help") {
				return helpText, true
			}
		case "config":
			if len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
				return configHelpText, true
			}
		}
	}
	return "", false
}

func (r Runner) runConfig(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return configUsage(stderr)
	}
	service := app.Service{HomeDir: r.HomeDir}
	switch args[0] {
	case "path":
		if len(args) != 1 {
			return configUsage(stderr)
		}
		path, err := service.ConfigurationPath()
		if err != nil {
			return writeConfigurationError(stderr, err)
		}
		fmt.Fprintln(stdout, path)
		return 0
	case "show":
		if len(args) != 1 {
			return configUsage(stderr)
		}
		view, err := service.ShowConfiguration()
		if err != nil {
			return writeConfigurationError(stderr, err)
		}
		fmt.Fprintf(stdout, "Default Uploader: %s\n", quoteJSON(view.DefaultUploader))
		if view.DefaultShortener == "" {
			fmt.Fprintln(stdout, "Default Shortener: none")
		} else {
			fmt.Fprintf(stdout, "Default Shortener: %s\n", quoteJSON(view.DefaultShortener))
		}
		if view.CopyToClipboard {
			fmt.Fprintln(stdout, "Copy to Clipboard: enabled")
		} else {
			fmt.Fprintln(stdout, "Copy to Clipboard: disabled")
		}
		return 0
	case "list-uploaders":
		if len(args) != 1 {
			return configUsage(stderr)
		}
		names, err := service.ListUploaders()
		if err != nil {
			return writeConfigurationError(stderr, err)
		}
		writeNameList(stdout, "Uploaders:", names)
		return 0
	case "list-shorteners":
		if len(args) != 1 {
			return configUsage(stderr)
		}
		names, present, err := service.ListShorteners()
		if err != nil {
			return writeConfigurationError(stderr, err)
		}
		if !present {
			fmt.Fprintln(stdout, "No Shorteners configured.")
			return 0
		}
		writeNameList(stdout, "Shorteners:", names)
		return 0
	case "set-default-uploader":
		if len(args) != 2 {
			return configUsage(stderr)
		}
		changed, err := service.SetDefaultUploader(args[1])
		if err != nil {
			return writeConfigurationError(stderr, err)
		}
		if changed {
			fmt.Fprintf(stdout, "Default Uploader set to %s.\n", quoteJSON(args[1]))
		} else {
			fmt.Fprintf(stdout, "Default Uploader is already %s.\n", quoteJSON(args[1]))
		}
		return 0
	case "set-default-shortener":
		if len(args) != 2 {
			return configUsage(stderr)
		}
		changed, err := service.SetDefaultShortener(args[1])
		if err != nil {
			return writeConfigurationError(stderr, err)
		}
		if changed {
			fmt.Fprintf(stdout, "Default Shortener set to %s.\n", quoteJSON(args[1]))
		} else {
			fmt.Fprintf(stdout, "Default Shortener is already %s.\n", quoteJSON(args[1]))
		}
		return 0
	case "clear-default-shortener":
		if len(args) != 1 {
			return configUsage(stderr)
		}
		changed, err := service.ClearDefaultShortener()
		if err != nil {
			return writeConfigurationError(stderr, err)
		}
		if changed {
			fmt.Fprintln(stdout, "Default Shortener cleared.")
		} else {
			fmt.Fprintln(stdout, "Default Shortener is already clear.")
		}
		return 0
	case "enable-clipboard", "disable-clipboard":
		if len(args) != 1 {
			return configUsage(stderr)
		}
		enabled := args[0] == "enable-clipboard"
		changed, err := service.SetClipboardCopying(enabled)
		if err != nil {
			return writeConfigurationError(stderr, err)
		}
		if changed {
			if enabled {
				fmt.Fprintln(stdout, "Clipboard copying enabled.")
			} else {
				fmt.Fprintln(stdout, "Clipboard copying disabled.")
			}
		} else if enabled {
			fmt.Fprintln(stdout, "Clipboard copying is already enabled.")
		} else {
			fmt.Fprintln(stdout, "Clipboard copying is already disabled.")
		}
		return 0
	case "validate":
		if len(args) != 1 {
			return configUsage(stderr)
		}
		if err := service.ValidateConfiguration(); err != nil {
			return writeConfigurationError(stderr, err)
		}
		fmt.Fprintln(stdout, "Configuration is valid.")
		return 0
	default:
		return configUsage(stderr)
	}
}

func configUsage(stderr io.Writer) int {
	fmt.Fprintln(stderr, "usage: upit config <command>")
	fmt.Fprint(stderr, configHelpText)
	return 2
}

func writeConfigurationError(stderr io.Writer, err error) int {
	failure := appFailure(err)
	fmt.Fprintln(stderr, "Configuration command failed")
	fmt.Fprintf(stderr, "Stage: %s\nMessage: %s\n", failure.Stage, failure.Message)
	return 1
}

func writeNameList(stdout io.Writer, heading string, names []string) {
	fmt.Fprintln(stdout, heading)
	for _, name := range names {
		fmt.Fprintf(stdout, "- %s\n", quoteJSON(name))
	}
}

func quoteJSON(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
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
	Shortener   string
	NoShorten   bool
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
	flags.StringVar(&options.Shortener, "shortener", "", "named Shortener")
	flags.BoolVar(&options.NoShorten, "no-shorten", false, "disable URL shortening")
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
	if options.Shortener != "" && options.NoShorten {
		return uploadOptions{}, fmt.Errorf("--shortener and --no-shorten cannot be used together")
	}
	if flags.NArg() != 1 {
		return uploadOptions{}, fmt.Errorf("upload requires exactly one file")
	}
	options.FilePath = flags.Arg(0)
	return options, nil
}
