package main

import (
	"flag"
	"os"

	"github.com/go-go-golems/geppetto/pkg/analysis/turnsrefactor"
)

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	var write bool
	var verify bool
	var allowNoop bool
	var patterns turnsrefactor.StringSliceFlag

	fs.BoolVar(&write, "w", false, "write files in-place")
	fs.BoolVar(&verify, "verify", true, "fail if any targeted calls remain after rewrite")
	fs.BoolVar(&allowNoop, "allow-noop", false, "do not fail when no rewrites are applied")
	fs.Var(&patterns, "packages", "package patterns to load (repeatable). default: ./...")

	_ = fs.Parse(os.Args[1:])
	pkgs := patterns.Values
	if len(pkgs) == 0 {
		pkgs = []string{"./..."}
	}

	cfg := turnsrefactor.Config{
		PackagePatterns: pkgs,
		Write:           write,
		Verify:          verify,
		AllowNoop:       allowNoop,
		Stdout:          os.Stdout,
		Stderr:          os.Stderr,
	}

	if err := turnsrefactor.Run(cfg); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
