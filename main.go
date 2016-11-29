package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/mattn/gover"
)

func usage() {
	fmt.Printf(`Usage of %s:
 Tasks:
   gom build   [options]   : Build with _vendor packages
   gom install [options]   : Install bundled packages into _vendor directory, by default.
                              GOM_VENDOR_NAME=. gom install [options], for regular src folder.
   gom test    [options]   : Run tests with bundles
   gom run     [options]   : Run go file with bundles
   gom doc     [options]   : Run godoc for bundles
   gom exec    [arguments] : Execute command with bundle environment
   gom tool    [options]   : Run go tool with bundles
   gom env     [arguments] : Run go env
   gom fmt     [arguments] : Run go fmt
   gom list    [arguments] : Run go list
   gom vet     [arguments] : Run go vet
   gom update              : Update all dependencies (Experiment)
   gom gen travis-yml      : Generate .travis.yml which uses "gom test"
   gom gen gomfile         : Scan packages from current directory as root
                              recursively, and generate Gomfile
   gom lock                : Generate Gomfile.lock
`, os.Args[0])
	os.Exit(1)
}

var productionEnv = flag.Bool("production", false, "production environment")
var developmentEnv = flag.Bool("development", false, "development environment")
var testEnv = flag.Bool("test", false, "test environment")
var projectMode = flag.Bool("project-mode", false, "do not move from vendor/src to vendor/")
var customGroups = flag.String("groups", "", "comma-separated list of Gomfile groups")
var customGroupList []string
var vendorFolder string
var isVendoringSupported bool

func init() {
	isVendoringSupported = checkVendoringSupport()
	if isVendoringSupported {
		vendorFolder = "vendor"
	} else {
		if len(os.Getenv("GOM_VENDOR_NAME")) > 0 {
			vendorFolder = os.Getenv("GOM_VENDOR_NAME")
		} else {
			vendorFolder = "_vendor"
		}
	}
}

func goversion() string {
	defer recover()
	return gover.Version()
}

// checkVendoringSupport return whether go have native vendor support.
// If return false, gom behave vendor directory as GOPATH.
// If return true, gom doesn't move anything.
func checkVendoringSupport() bool {
	go150, _ := version.NewVersion("1.5.0")
	go160, _ := version.NewVersion("1.6.0")
	go173, _ := version.NewVersion("1.7.3")
	ver := goversion()

	// TODO: maybe gccgo?
	if ver == "" {
		return true
	}

	goVer, err := version.NewVersion(strings.TrimPrefix(ver, "go"))
	if err != nil {
		panic(fmt.Sprintf("gover.Version() returned invalid semantic version: %s", ver))
	}

	// See: https://golang.org/doc/go1.6#go_command
	if goVer.LessThan(go150) {
		return false
	} else if (goVer.Equal(go150) || goVer.GreaterThan(go150)) && goVer.LessThan(go160) {
		return os.Getenv("GO15VENDOREXPERIMENT") == "1"
	} else if (goVer.Equal(go160) || goVer.GreaterThan(go160)) && goVer.LessThan(go173) {
		return os.Getenv("GO15VENDOREXPERIMENT") != "0"
	} else {
		return true
	}
}

func vendorSrc(vendor string) string {
	if isVendoringSupported {
		return vendor
	} else {
		return filepath.Join(vendor, "src")
	}
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() == 0 {
		usage()
	}
	handleSignal()

	if !*productionEnv && !*developmentEnv && !*testEnv {
		*developmentEnv = true
	}

	customGroupList = strings.Split(*customGroups, ",")

	var err error
	subArgs := flag.Args()[1:]
	switch flag.Arg(0) {
	case "install", "i":
		err = install(subArgs)
	case "build", "b":
		err = run(append([]string{"go", "build"}, subArgs...), None)
	case "test", "t":
		err = run(append([]string{"go", "test"}, subArgs...), None)
	case "run", "r":
		err = run(append([]string{"go", "run"}, subArgs...), None)
	case "doc", "d":
		err = run(append([]string{"godoc"}, subArgs...), None)
	case "exec", "e":
		err = run(subArgs, None)
	case "env", "tool", "fmt", "list", "vet":
		err = run(append([]string{"go", flag.Arg(0)}, subArgs...), None)
	case "update", "u":
		err = update()
	case "gen", "g":
		switch flag.Arg(1) {
		case "travis-yml":
			err = genTravisYml()
		case "gomfile":
			err = genGomfile()
		default:
			usage()
		}
	case "lock", "l":
		err = genGomfileLock()
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gom: ", err)
		os.Exit(1)
	}
}
