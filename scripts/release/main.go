// Package main provides release helper tasks powered by goyek.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/goyek/goyek/v2"
)

var semverTagPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)

type version struct {
	major int
	minor int
	patch int
}

func (v version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}

func (v version) Tag() string {
	return "v" + v.String()
}

func parseSemverTag(raw string) (version, bool) {
	matches := semverTagPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(matches) != 4 {
		return version{}, false
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return version{}, false
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return version{}, false
	}
	patch, err := strconv.Atoi(matches[3])
	if err != nil {
		return version{}, false
	}

	return version{major: major, minor: minor, patch: patch}, true
}

func (v version) lessThan(other version) bool {
	if v.major != other.major {
		return v.major < other.major
	}
	if v.minor != other.minor {
		return v.minor < other.minor
	}
	return v.patch < other.patch
}

func latestVersion(repoRoot string) (version, error) {
	repo, err := git.PlainOpenWithOptions(repoRoot, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return version{}, fmt.Errorf("open git repository: %w", err)
	}

	iter, err := repo.Tags()
	if err != nil {
		return version{}, fmt.Errorf("list git tags: %w", err)
	}
	defer iter.Close()

	latest := version{}
	found := false

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		v, ok := parseSemverTag(ref.Name().Short())
		if !ok {
			return nil
		}
		if !found || latest.lessThan(v) {
			latest = v
			found = true
		}
		return nil
	})
	if err != nil {
		return version{}, fmt.Errorf("iterate git tags: %w", err)
	}

	if !found {
		return version{major: 0, minor: 0, patch: 0}, nil
	}
	return latest, nil
}

func bump(v version, kind string) (version, error) {
	switch kind {
	case "major":
		return version{major: v.major + 1, minor: 0, patch: 0}, nil
	case "minor":
		return version{major: v.major, minor: v.minor + 1, patch: 0}, nil
	case "patch":
		return version{major: v.major, minor: v.minor, patch: v.patch + 1}, nil
	default:
		return version{}, errors.New("unsupported bump type: " + kind)
	}
}

func emitLatest(a *goyek.A, withPrefix bool) {
	current, err := latestVersion(".")
	if err != nil {
		a.Fatalf("failed to read git repository tags: %v", err)
	}

	if withPrefix {
		writeLineOrFatal(a, a.Output(), current.Tag())
		return
	}
	writeLineOrFatal(a, a.Output(), current.String())
}

func emitBump(a *goyek.A, kind string) {
	current, err := latestVersion(".")
	if err != nil {
		a.Fatalf("failed to read git repository tags: %v", err)
	}
	next, bumpErr := bump(current, kind)
	if bumpErr != nil {
		a.Fatal(bumpErr.Error())
	}
	writeLineOrFatal(a, a.Output(), next.String())
}

func writeLineOrFatal(a *goyek.A, w io.Writer, value string) {
	if _, err := fmt.Fprintln(w, value); err != nil {
		a.Fatalf("failed to write output: %v", err)
	}
}

func defineTasks() {
	goyek.Define(goyek.Task{
		Name: "latest",
		Action: func(a *goyek.A) {
			emitLatest(a, true)
		},
	})

	goyek.Define(goyek.Task{
		Name: "latest-no-v",
		Action: func(a *goyek.A) {
			emitLatest(a, false)
		},
	})

	goyek.Define(goyek.Task{
		Name: "bump-major",
		Action: func(a *goyek.A) {
			emitBump(a, "major")
		},
	})

	goyek.Define(goyek.Task{
		Name: "bump-minor",
		Action: func(a *goyek.A) {
			emitBump(a, "minor")
		},
	})

	goyek.Define(goyek.Task{
		Name: "bump-patch",
		Action: func(a *goyek.A) {
			emitBump(a, "patch")
		},
	})
}

func usage() {
	if _, err := os.Stderr.WriteString("usage: go run ./scripts/release <latest|latest-no-v|bump-major|bump-minor|bump-patch>\n"); err != nil {
		panic(fmt.Errorf("write usage: %w", err))
	}
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	target := os.Args[1]
	switch target {
	case "latest", "latest-no-v", "bump-major", "bump-minor", "bump-patch":
	default:
		usage()
		os.Exit(1)
	}

	defineTasks()

	if err := goyek.Execute(context.Background(), []string{target}); err != nil {
		if _, writeErr := os.Stderr.WriteString(err.Error() + "\n"); writeErr != nil {
			panic(fmt.Errorf("write execute error: %w", writeErr))
		}
		os.Exit(1)
	}
}
