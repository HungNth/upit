package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

var (
	// Can be set via -ldflags "-X github.com/HungNth/upit/internal/version.version=<val>"
	version = "0.9.0"
	commit  = ""
	date    = ""
)

// Info contains complete product version and runtime build metadata.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Get returns the current build and runtime version info.
func Get() Info {
	v := version
	if v == "" {
		v = "0.9.0"
	}

	c := commit
	d := date

	if c == "" || d == "" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			var rev string
			var modified bool
			var vcsTime string

			for _, s := range bi.Settings {
				switch s.Key {
				case "vcs.revision":
					rev = s.Value
				case "vcs.modified":
					modified = (s.Value == "true")
				case "vcs.time":
					vcsTime = s.Value
				}
			}

			if c == "" && rev != "" {
				if len(rev) > 7 {
					c = rev[:7]
				} else {
					c = rev
				}
				if modified {
					c += "-dirty"
				}
			}

			if d == "" && vcsTime != "" {
				d = vcsTime
			}
		}
	}

	if c == "" {
		c = "unknown"
	}
	if d == "" {
		d = "unknown"
	}

	return Info{
		Version:   v,
		Commit:    c,
		Date:      d,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String formats the version info as human-readable text:
// upit <version> (commit: <sha>, built: <date>)
func (i Info) String() string {
	d := i.Date
	if parsed, err := time.Parse(time.RFC3339, d); err == nil {
		d = parsed.UTC().Format("2006-01-02")
	} else if idx := strings.IndexByte(d, 'T'); idx != -1 {
		d = d[:idx]
	}
	return fmt.Sprintf("upit %s (commit: %s, built: %s)", i.Version, i.Commit, d)
}
