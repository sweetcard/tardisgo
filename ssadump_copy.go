// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Modifications:
// Copyright 2014 Elliott Stoneham and The TARDIS Go Authors
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// TARDIS Go modified version of ssadump: a tool for displaying and interpreting the SSA form of Go programs.
// TODO add own command line interface etc
package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"code.google.com/p/go.tools/go/types"
	"code.google.com/p/go.tools/importer"
	"code.google.com/p/go.tools/ssa"
	//"code.google.com/p/go.tools/ssa/interp" //TARDIS Go temporary removal due to Windows 7 related bug

	_ "github.com/tardisgo/tardisgo/haxe" // TARDIS Go addition
	"github.com/tardisgo/tardisgo/pogo"   // TARDIS Go addition
	"go/parser"                           // TARDIS Go addition
)

var buildFlag = flag.String("build", "", `Options controlling the SSA builder.
The value is a sequence of zero or more of these letters:
C	perform sanity [C]hecking of the SSA form.
D	include [D]ebug info for every function.
P	log [P]ackage inventory.
F	log [F]unction SSA code.
S	log [S]ource locations as SSA builder progresses.
G	use binary object files from gc to provide imports (no code).
L	build distinct packages seria[L]ly instead of in parallel.
N	build [N]aive SSA form: don't replace local loads/stores with registers.
`)

var runFlag = flag.Bool("run", false, "Invokes the SSA interpreter on the program.")

var interpFlag = flag.String("interp", "", `Options controlling the SSA test interpreter.
The value is a sequence of zero or more more of these letters:
R	disable [R]ecover() from panic; show interpreter crash instead.
T	[T]race execution of the program.  Best for single-threaded programs!
`)

// TARDIS Go modification TODO review words here
const usage = `SSA builder and TARDIS Go transpiler (version 0.0.1-unreleased : interpreter removed due to Win7 related bug).
Usage: tardisgo [<flag> ...] <args> ...
A shameless copy of the ssadump utility, but also writes a 'Go.hx' haxe file into the 'tardis' sub-directory of the current location (which you must create by hand).
Example:
% tardisgo hello.go
`
const ignore = `
Use -help flag to display options.

Examples:
% ssadump -build=FPG hello.go         # quickly dump SSA form of a single package
% ssadump -run -interp=T hello.go     # interpret a program, with tracing
% ssadump -run unicode -- -test.v     # interpret the unicode package's tests, verbosely
` + importer.InitialPackagesUsage +
	`
When -run is specified, ssadump will find the first package that
defines a main function and run it in the interpreter.
If none is found, the tests of each package will be run instead.
`

// end TARDIS Go modification

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func init() {
	// If $GOMAXPROCS isn't set, use the full capacity of the machine.
	// For small machines, use at least 4 threads.
	if os.Getenv("GOMAXPROCS") == "" {
		n := runtime.NumCPU()
		if n < 4 {
			n = 4
		}
		runtime.GOMAXPROCS(n)
	}
}

func main() {
	flag.Parse()
	args := flag.Args()

	impctx := importer.Config{
		Build:         &build.Default,
		SourceImports: true,
	}
	// TODO(adonovan): make go/types choose its default Sizes from
	// build.Default or a specified *build.Context.
	var wordSize int64 = 8
	switch impctx.Build.GOARCH {
	case "386", "arm":
		wordSize = 4
	}
	impctx.TypeChecker.Sizes = &types.StdSizes{
		MaxAlign: 8,
		WordSize: wordSize,
	}

	var debugMode bool
	var mode ssa.BuilderMode
	for _, c := range *buildFlag {
		switch c {
		case 'D':
			debugMode = true
		case 'P':
			mode |= ssa.LogPackages | ssa.BuildSerially
		case 'F':
			mode |= ssa.LogFunctions | ssa.BuildSerially
		case 'S':
			mode |= ssa.LogSource | ssa.BuildSerially
		case 'C':
			mode |= ssa.SanityCheckFunctions
		case 'N':
			mode |= ssa.NaiveForm
		case 'G':
			impctx.SourceImports = false
		case 'L':
			mode |= ssa.BuildSerially
		default:
			log.Fatalf("Unknown -build option: '%c'.", c)
		}
	}

	var interpMode interp.Mode
	for _, c := range *interpFlag {
		switch c {
		case 'T':
			interpMode |= interp.EnableTracing
		case 'R':
			interpMode |= interp.DisableRecover
		default:
			log.Fatalf("Unknown -interp option: '%c'.", c)
		}
	}

	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	// Profiling support.
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Load, parse and type-check the program.
	imp := importer.New(&impctx)
	infos, args, err := imp.LoadInitialPackages(args)
	if err != nil {
		log.Fatal(err)
	}

	// The interpreter needs the runtime package.
	if *runFlag {
		if _, err := imp.ImportPackage("runtime"); err != nil {
			log.Fatalf("ImportPackage(runtime) failed: %s", err)
		}
	}

	// Create and build SSA-form program representation.
	prog := ssa.NewProgram(imp.Fset, mode)

	// TARDIS GO additions to add the language specific go runtime code
	goruntime, err := parser.ParseFile(imp.Fset, "langgoruntime.go", pogo.LanguageList[pogo.TargetLang].Goruntime, 0) // Parse the input file.
	if err != nil {
		fmt.Print(err) // parse error
		return
	}
	goruntimePI := imp.CreatePackage("goruntime", goruntime) // Create single-file goruntime package and import its dependencies.
	if goruntimePI.Err != nil {
		log.Fatal(goruntimePI.Err)
	}
	prog.CreatePackage(goruntimePI)
	// end TARDIS GO additions

	if err := prog.CreatePackages(imp); err != nil {
		log.Fatal(err)
	}

	if debugMode {
		for _, pkg := range prog.AllPackages() {
			pkg.SetDebugMode(true)
		}
	}
	prog.BuildAll()

	// Run the interpreter.
	if *runFlag {
		// If some package defines main, run that.
		// Otherwise run all package's tests.
		var main *ssa.Package
		var pkgs []*ssa.Package
		for _, info := range infos {
			pkg := prog.Package(info.Pkg)
			if pkg.Func("main") != nil {
				main = pkg
				break
			}
			pkgs = append(pkgs, pkg)
		}
		if main == nil && pkgs != nil {
			main = prog.CreateTestMainPackage(pkgs...)
		}
		if main == nil {
			log.Fatal("No main package and no tests")
		}

		if runtime.GOARCH != impctx.Build.GOARCH {
			log.Fatalf("Cross-interpretation is not yet supported (target has GOARCH %s, interpreter has %s).",
				impctx.Build.GOARCH, runtime.GOARCH)
		}

		// interp.Interpret(main, interpMode, impctx.TypeChecker.Sizes, main.Object.Path(), args) //TARDIS Go temporary removal due to Windows 7 related bug
	}

	// TARDIS Go additions: copy run interpreter code above, but call pogo class
	if true {
		// If some package defines main, run that.
		// Otherwise run all package's tests.
		var main *ssa.Package
		var pkgs []*ssa.Package
		for _, info := range infos {
			pkg := prog.Package(info.Pkg)
			if pkg.Func("main") != nil {
				main = pkg
				break
			}
			pkgs = append(pkgs, pkg)
		}
		if main == nil && pkgs != nil {
			main = prog.CreateTestMainPackage(pkgs...)
		}
		if main == nil {
			log.Fatal("No main package and no tests")
		}

		if runtime.GOARCH != impctx.Build.GOARCH {
			log.Fatalf("Cross-interpretation is not yet supported (target has GOARCH %s, interpreter has %s).",
				impctx.Build.GOARCH, runtime.GOARCH)
		}

		//interp.Interpret(main, interpMode, impctx.TypeChecker.Sizes, main.Object.Path(), args)
		pogo.EntryPoint(main) // TARDIS Go entry point, no return, does os.Exit at end
	}

}