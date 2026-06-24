package main

import (
	"bytes"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func TestRun(t *testing.T) {
	cases := []struct {
		name       string
		version    string
		args       []string
		dirs       []string
		files      []string
		wantOut    []string
		wantCode   int
		wantErrSub string
	}{
		{
			name:    "default walks whole tree",
			args:    []string{"find", "/root"},
			dirs:    []string{"/root/sub"},
			files:   []string{"/root/a.txt", "/root/sub/b.txt"},
			wantOut: []string{"/root", "/root/a.txt", "/root/sub", "/root/sub/b.txt"},
		},
		{
			name:    "type f selects files only",
			args:    []string{"find", "-type", "f", "/root"},
			dirs:    []string{"/root/sub"},
			files:   []string{"/root/a.txt", "/root/sub/b.txt"},
			wantOut: []string{"/root/a.txt", "/root/sub/b.txt"},
		},
		{
			name:    "type d selects directories only",
			args:    []string{"find", "--type", "d", "/root"},
			dirs:    []string{"/root/sub"},
			files:   []string{"/root/a.txt", "/root/sub/b.txt"},
			wantOut: []string{"/root", "/root/sub"},
		},
		{
			name:    "maxdepth limits walk depth",
			args:    []string{"find", "-maxdepth", "1", "/root"},
			dirs:    []string{"/root/sub"},
			files:   []string{"/root/a.txt", "/root/sub/b.txt"},
			wantOut: []string{"/root", "/root/a.txt", "/root/sub"},
		},
		{
			name:    "type and maxdepth combine",
			args:    []string{"find", "-type", "f", "-maxdepth", "1", "/root"},
			dirs:    []string{"/root/sub"},
			files:   []string{"/root/a.txt", "/root/sub/b.txt"},
			wantOut: []string{"/root/a.txt"},
		},
		{
			name:    "default path is current directory",
			args:    []string{"find"},
			dirs:    []string{"sub"},
			files:   []string{"a.txt", "sub/b.txt"},
			wantOut: []string{".", "a.txt", "sub", "sub/b.txt"},
		},
		{
			name:    "version flag reports injected version",
			version: "1.2.3",
			args:    []string{"find", "--version"},
			wantOut: []string{"find version 1.2.3"},
		},
		{
			name:       "unknown flag errors",
			args:       []string{"find", "--nope"},
			wantCode:   1,
			wantErrSub: "find:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			for _, dir := range tc.dirs {
				if err := fs.MkdirAll(dir, 0o755); err != nil {
					t.Fatalf("mkdir %s: %v", dir, err)
				}
			}
			for _, path := range tc.files {
				if err := afero.WriteFile(fs, path, []byte(""), 0o644); err != nil {
					t.Fatalf("write fixture %s: %v", path, err)
				}
			}

			var out, errOut bytes.Buffer
			code := run(tc.version, tc.args, nil, &out, &errOut, fs)

			if code != tc.wantCode {
				t.Fatalf("exit code = %d, want %d (stderr=%q)", code, tc.wantCode, errOut.String())
			}
			if tc.wantErrSub != "" {
				if !strings.Contains(errOut.String(), tc.wantErrSub) {
					t.Fatalf("stderr = %q, want substring %q", errOut.String(), tc.wantErrSub)
				}
				return
			}
			got := lines(out.String())
			if !equal(got, tc.wantOut) {
				t.Fatalf("stdout lines = %v, want %v", got, tc.wantOut)
			}
		})
	}
}

func lines(s string) []string {
	out := strings.Split(strings.TrimRight(s, "\n"), "\n")
	sort.Strings(out)
	return out
}

func equal(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func Test_main(t *testing.T) {
	origExit, origRun := osExit, runCLI
	t.Cleanup(func() { osExit, runCLI = origExit, origRun })

	gotCode := -1
	osExit = func(code int) { gotCode = code }
	runCLI = func(string, []string, io.Reader, io.Writer, io.Writer, afero.Fs) int { return 7 }

	main()

	if gotCode != 7 {
		t.Fatalf("main propagated exit code %d, want 7", gotCode)
	}
}
