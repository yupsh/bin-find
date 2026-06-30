package main

import (
	"context"
	"fmt"
	"io"

	command "github.com/gloo-foo/cmd-find"
	gloo "github.com/gloo-foo/framework"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

const (
	flagType     = "type"
	flagMaxDepth = "maxdepth"
)

// usageText is the command's multi-line usage synopsis, shown in --help.
// cli/v3 indents the whole block by 3 spaces, so these lines are flush-left to
// stay aligned in the rendered output.
const usageText = `find [PATH] [OPTIONS]

search for files in the directory hierarchy rooted at PATH (default: .).`

// init replaces urfave/cli's default --version/-v flag with a --version-only
// flag, freeing the single-letter -v for command flags while still exposing the
// injected build version.
func init() {
	cli.VersionFlag = &cli.BoolFlag{Name: "version", Usage: "print version information and exit"}
}

// run builds and executes the find CLI against the injected version, I/O, and
// filesystem, returning the process exit code. find is a source-position
// command: it does not read stdin, but walks the injected filesystem.
func run(version string, args []string, _ io.Reader, stdout, stderr io.Writer, fs afero.Fs) int {
	cmd := newApp(version, stdout, fs)
	cmd.Writer = stdout
	cmd.ErrWriter = stderr
	if err := cmd.Run(context.Background(), args); err != nil {
		_, _ = fmt.Fprintf(stderr, "find: %v\n", err)
		return 1
	}
	return 0
}

func newApp(version string, stdout io.Writer, fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:            "find",
		Version:         version,
		Usage:           "search for files in a directory hierarchy",
		UsageText:       usageText,
		HideHelpCommand: true,
		// Keep exit handling in run() rather than letting urfave/cli call
		// os.Exit, so the exit code stays testable.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: flagType, Usage: "file is of type TYPE (f=file, d=directory)"},
			&cli.IntFlag{Name: flagMaxDepth, Usage: "descend at most LEVELS (a non-negative integer) levels"},
		},
		Action: action(stdout, fs),
	}
}

func action(stdout io.Writer, fs afero.Fs) cli.ActionFunc {
	return func(_ context.Context, c *cli.Command) error {
		_, err := gloo.Run(source(c, fs), gloo.ByteWriteTo(stdout))
		return err
	}
}

func path(c *cli.Command) string {
	if c.NArg() == 0 {
		return "."
	}
	return c.Args().First()
}

// source builds the find Source for the active flags. command.Find takes a
// variadic of an unexported interface, so the option set cannot be assembled
// dynamically; each combination of the optional flags is enumerated with
// explicitly typed arguments. FindFs is always applied.
func source(c *cli.Command, fs afero.Fs) gloo.Source[[]byte] {
	p := path(c)
	switch {
	case c.IsSet(flagType) && c.IsSet(flagMaxDepth):
		return command.Find(
			p,
			command.FindFs(fs),
			command.FindType(c.String(flagType)),
			command.FindMaxDepth(c.Int(flagMaxDepth)),
		)
	case c.IsSet(flagType):
		return command.Find(p, command.FindFs(fs), command.FindType(c.String(flagType)))
	case c.IsSet(flagMaxDepth):
		return command.Find(p, command.FindFs(fs), command.FindMaxDepth(c.Int(flagMaxDepth)))
	default:
		return command.Find(p, command.FindFs(fs))
	}
}
