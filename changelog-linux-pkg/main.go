package main

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Project    string  `yaml:"project"`
	Maintainer string  `yaml:"maintaner"`
	Changelog  []Entry `yaml:"changelog"`
}

type Entry struct {
	Version string    `yaml:"version"`
	Urgency string    `yaml:"urgency"`
	Stable  bool      `yaml:"stable"`
	Date    LocalTime `yaml:"date"`
	Changes []Change  `yaml:"changes"`
}

type Change struct {
	Desc   string `yaml:"desc"`
	Closes []any  `yaml:"closes"`
}

// LocalTime wraps time.Time to parse custom date format.
type LocalTime struct {
	time.Time
}

func (l LocalTime) Cmp(j LocalTime) int {
	if l.Time.Equal(j.Time) {
		return 0
	}
	if l.Time.Before(j.Time) {
		return 1
	}
	return -1
}

func (lt *LocalTime) UnmarshalYAML(value *yaml.Node) error {
	// Expect format "2006-01-02 15:04:05 +0400"
	parsed, err := time.Parse("2006-01-02 15:04:05 -0700", value.Value)
	if err != nil {
		return err
	}
	lt.Time = parsed.Local()
	return nil
}

func stable(stable bool) string {
	if stable {
		return "stable"
	}
	return "unstable"
}

func readLog() (*Config, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	slices.SortFunc(cfg.Changelog, func(a, b Entry) int { return a.Date.Cmp(b.Date) })

	return &cfg, nil
}

func doDebChangelog(cfg *Config) error {

	for _, entry := range cfg.Changelog {
		fmt.Printf("%s (%s) %s; urgency=%s\n\n",
			cfg.Project,
			entry.Version,
			stable(entry.Stable),
			entry.Urgency,
		)

		for _, change := range entry.Changes {
			fmt.Printf("  * %s\n", change.Desc)
			if len(change.Closes) > 0 {
				var closes []string
				for _, c := range change.Closes {
					switch v := c.(type) {
					case string:
						closes = append(closes, v)
					case int:
						closes = append(closes, fmt.Sprintf("#%d", v))
					}
				}
				closesStr := strings.Join(closes, ", ")
				fmt.Printf("  Closes: %s\n", closesStr)
			}
		}
		fmt.Printf("\n -- %s  %s\n\n", cfg.Maintainer, entry.Date.Format(time.RFC1123Z))
	}
	return nil
}

func doRpmChangelog(cfg *Config) error {

	for _, entry := range cfg.Changelog {
		fmt.Printf("* %s %s - %s\n",
			entry.Date.Format("Mon Jan 2 2006"),
			cfg.Maintainer,
			entry.Version,
		)
		for _, change := range entry.Changes {
			fmt.Printf("- %s", change.Desc)
			if len(change.Closes) > 0 {
				var closes []string
				for _, c := range change.Closes {
					switch v := c.(type) {
					case string:
						closes = append(closes, v)
					case int:
						closes = append(closes, fmt.Sprintf("#%d", v))
					}
				}
				closesStr := strings.Join(closes, ", ")
				fmt.Printf("  (Closes: %s)", closesStr)
			}
			fmt.Println()
		}
		fmt.Println()
	}
	return nil

}

func main() {
	err := mainInner()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func mainInner() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("Usage: %s <rpm|deb> < changelog.yaml", os.Args[0])
	}
	cfg, err := readLog()
	if err != nil {
		return err
	}

	switch os.Args[1] {
	case "rpm":
		err = doRpmChangelog(cfg)
	case "deb":
		err = doDebChangelog(cfg)
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", os.Args[1])
	}
	if err != nil {
		return err
	}

	return nil
}
