package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"
)

var scanner = bufio.NewScanner(os.Stdin)
var colorStdout bool
var colorStderr bool
var dir string
var moduleUrl string
var exeName string
var remoteName string
var remoteUrl string
var doGit, doMainGo, doGitRemote, doGitignore, doMakefile *bool
var onWarn int // What to do on a warning, 0: prompt, 1: continue, 2: stop

func scanLine(prompt string) string {
	fmt.Print(prompt)
	scanner.Scan()
	return strings.Trim(scanner.Text(), " \t")
}

func setOptions() {
	o, _ := os.Stdout.Stat()
	if (o.Mode() & os.ModeCharDevice) == os.ModeCharDevice {
		colorStdout = true
	} else {
		colorStdout = false
	}

	e, _ := os.Stderr.Stat()
	if (e.Mode() & os.ModeCharDevice) == os.ModeCharDevice {
		colorStderr = true
	} else {
		colorStderr = false
	}

	var doMainGoOpt, doGitOpt, doGitignoreOpt, doMakefileOpt bool
	var noDoMainGoOpt, noDoGitOpt, noDoGitignoreOpt, noDoMakefileOpt bool
	var onWarnOpt string

	flag.StringVar(&moduleUrl, "module-url", "", "url of the go module")
	flag.StringVar(&exeName, "executable-name", "", "name of the output executable")
	flag.StringVar(&remoteUrl, "remote-url", "", "url of the git remote.\nThis option sets do-git to true")
	flag.StringVar(&remoteName, "remote-name", "", "name of the git remote.\nThis option sets do-git to true")
	flag.StringVar(&onWarnOpt, "on-warning", "Prompt",
		`What to do on a warning, one of:
    	Prompt: prompt for wether to continue or not
    	Continue: ignore warning and continue
    	Stop: stop the current opperation`)

	flag.BoolVar(&doMainGoOpt, "do-main-go", false, "sets that a main.go file should be created")
	flag.BoolVar(&doGitOpt, "do-git", false, "sets that a git repository should be created")
	flag.BoolVar(&doGitignoreOpt, "do-gitignore", false, "sets that the gitignore should be created.\nThis option sets do-git to true")
	flag.BoolVar(&doMakefileOpt, "do-makefile", false, "sets that a Makefile should be created")
	flag.BoolVar(&noDoMainGoOpt, "no-do-main-go", false, "sets that a main.go file should not be created")
	flag.BoolVar(&noDoGitOpt, "no-do-git", false, "sets that a git repository should not be created")
	flag.BoolVar(&noDoGitignoreOpt, "no-do-gitignore", false, "sets that the gitignore should not be created.")
	flag.BoolVar(&noDoMakefileOpt, "no-do-makefile", false, "sets that a Makefile should not be created")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [project_dir]\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nNo options are required, if anything isn't known input is asked on stdin")
		fmt.Fprintln(os.Stderr, "The program will error and exit if conflicting arguments are given.")
		fmt.Fprintln(os.Stderr, "At any point, unless specified otherwise, you can give an empty input to abort the program.")
	}

	flag.Parse()
	exeName, _ = strings.CutSuffix(exeName, ".exe")

	switch strings.ToLower(onWarnOpt) {
	case "prompt":
		onWarn = 0
	case "continue":
		onWarn = 1
	case "stop":
		onWarn = 2
	default:
		ePrintlnC("{red}Error: unkown action on warning `" + onWarnOpt + "`{reset}")
		flag.Usage()
		os.Exit(1)
	}

	var effectiveDoGit *bool = nil
	True, False := true, false

	if remoteUrl != "" || remoteName != "" {
		effectiveDoGit = &True
		doGitRemote = &True
	}

	if doGitOpt {
		effectiveDoGit = &True
	}

	if noDoGitOpt {
		if effectiveDoGit == &True {
			ePrintlnC("{red}Error: conflicting commandline arguments given{reset}")
			flag.Usage()
			os.Exit(1)
		}

		effectiveDoGit = &False
	}

	if doGitignoreOpt {
		if effectiveDoGit == &False {
			ePrintlnC("{red}Error: conflicting commandline arguments given{reset}")
			flag.Usage()
			os.Exit(1)
		}

		effectiveDoGit = &True
		doGitignore = &True
	}

	if noDoGitignoreOpt {
		if effectiveDoGit == &True {
			ePrintlnC("{red}Error: conflicting commandline arguments given{reset}")
			flag.Usage()
			os.Exit(1)
		}

		effectiveDoGit = &False
		doGitignore = &False
	}

	doGit = effectiveDoGit

	if doMakefileOpt {
		doMakefile = &True
	}

	if noDoMakefileOpt {
		if doMakefile == &True {
			ePrintlnC("{red}Error: conflicting commandline arguments given{reset}")
			flag.Usage()
			os.Exit(1)
		}

		doMakefile = &False
	}

	if doMainGoOpt {
		doMainGo = &True
	}

	if noDoMainGoOpt {
		if doMainGo == &True {
			ePrintlnC("{red}Error: conflicting commandline arguments given{reset}")
			flag.Usage()
			os.Exit(1)
		}

		doMainGo = &False
	}

	if len(flag.Args()) == 0 {
		return
	}

	dir = flag.Arg(0)
}

func printCmd(cmd *exec.Cmd) {
	printC("{blue}${reset} ")

	for _, arg := range cmd.Args {
		fmt.Print(arg, " ")
	}

	fmt.Println()
}

func main() {
	setOptions()

	makeProjectDir()
	makeModule()
	makeMainGo()
	makeMakefile()
	makeGitRepo()
}

func makeMainGo() {
	if doMainGo == nil {
		answer := scanLine("Do you want to make main.go? (Y/N) ")
		if answer == "" {
			os.Exit(0)
		}

		if answer[0] == 'N' || answer[0] == 'n' {
			return
		}
	} else if !*doMainGo {
		return
	}

	printlnC("{blue}Info:{reset} creating main.go")

	if exists("main.go") {
		contents, err := os.ReadFile("main.go")
		if err != nil {
			ePrintlnC("{red}Error checking contents of main.go:", err.Error(), "{reset}")
			return
		}

		if len(contents) > 0 {
			if onWarn == 0 {
				printlnC("{yellow}Warning: main.go already exists and is not empty{reset}")
				answer := scanLine("Do you want to override it? (Y/N) ")
				if answer == "" {
					os.Exit(0)
				}

				if answer[0] != 'Y' && answer[0] != 'y' {
					return
				}
			} else if onWarn == 2 {
				return
			}
		}
	}

	file, err := os.Create("main.go")
	if err != nil {
		ePrintlnC("{red}Error creating main.go:", err.Error(), "{reset}")
		return
	}

	writer := bufio.NewWriter(file)

	writer.WriteString("package main\n\n")
	writer.WriteString("import \"fmt\"\n\n")
	writer.WriteString("func main() {\n")
	writer.WriteString("	fmt.Println(\"generated by github.com/Peirceman/go_project_creator\")\n")
	writer.WriteString("}\n")
	writer.Flush()

	file.Close()
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

func makeMakefile() {
	if doMakefile == nil {
		answer := scanLine("Do you want to make a Makefile? (Y/N) ")
		if answer == "" {
			os.Exit(0)
		}
		if answer[0] == 'N' || answer[0] == 'n' {
			return
		}
	} else if !*doMakefile {
		return
	}

	printlnC("{blue}Info:{reset} creating Makefile")

	if exists("Makefile") {
		contents, err := os.ReadFile("Makefile")
		if err != nil {
			ePrintlnC("{red}Error checking contents of Makefile:", err.Error(), "{reset}")
			return
		}

		if len(contents) > 0 {
			if onWarn == 0 {
				printlnC("{yellow}Warning: Makefile already exists and is not empty{reset}")
				answer := scanLine("Do you want to override it? (Y/N) ")
				if answer == "" {
					os.Exit(0)
				}
				if answer[0] != 'Y' && answer[0] != 'y' {
					return
				}
			} else if onWarn == 2 {
				return
			}
		}
	}

	file, err := os.Create("Makefile")
	if err != nil {
		ePrintlnC("{red}Error creating Makefile:", err.Error(), "{reset}")
		return
	}

	if exeName == "" {
		exeName, _ = strings.CutSuffix(scanLine("Executable name: "), ".exe")
	}

	if exeName == "" {
		os.Exit(0)
	}

	writer := bufio.NewWriter(file)

	writer.WriteString("# generated by go_project_creator: https://github.com/Peirceman/go_project_creator\n\n")
	writer.WriteString("GO_FILES := $(shell find . -name '*.go' ! -name '*_test.go' -type f)\n")
	writer.WriteString("OUTPUT_EXE := ")
	writer.WriteString(exeName)
	writer.WriteString("\n\n")
	writer.WriteString("ifdef OS\n")
	writer.WriteString("	OUTPUT_EXE := $(OUTPUT_EXE).exe\n")
	writer.WriteString("endif\n")
	writer.WriteString("\n")
	writer.WriteString("$(OUTPUT_EXE): $(GO_FILES)\n")
	writer.WriteString("	go build -o $(OUTPUT_EXE)\n")
	writer.Flush()

	file.Close()
}

func runCmd(name string, args ...string) error {

	cmd := exec.Command(name, args...)
	printCmd(cmd)

	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	err := cmd.Run()

	return err
}

func makeGitRepo() {
	if doGit == nil {
		answer := scanLine("Do you want to make a git repository? (Y/N) ")
		if answer == "" {
			os.Exit(0)
		}
		if answer[0] == 'N' || answer[0] == 'n' {
			return
		}
	} else if !*doGit {
		return
	}

	printlnC("{blue}Info:{reset} creating git repository")
	err := runCmd("git", "init")

	if err != nil {
		ePrintlnC("{red}Error creating git repository:", err.Error(), "{reset}")
		return
	}

	makeGitIgnore()
	addGitRemote()
}

func addGitRemote() {
	if doGitRemote == nil {
		answer := scanLine("Do you want to add a remote? (Y/N) ")
		if answer == "" {
			os.Exit(0)
		}
		if answer[0] != 'Y' && answer[0] != 'y' {
			return
		}
	} else if !*doGitRemote {
		return
	}

	if remoteName == "" {
		remoteName = scanLine("Enter remote name: (empty for origin) ")
		if remoteName == "" {
			remoteName = "origin"
		}
	}

	if remoteUrl == "" {
		remoteUrl = scanLine("Enter remote url: (empty for " + moduleUrl + ") ")
		if remoteUrl == "" {
			remoteUrl = moduleUrl
		}
	}

	err := runCmd("git", "remote", "add", remoteName, remoteUrl)
	if err != nil {
		ePrintlnC("{red}Error adding remote:", err.Error(), "{reset}")
	}
}

func makeGitIgnore() {
	if doGitignore == nil {
		answer := scanLine("Do you want to add a gitignore? (Y/N) ")
		if answer == "" {
			os.Exit(0)
		}
		if answer[0] != 'Y' && answer[0] != 'y' {
			return
		}
	} else if !*doGitignore {
		return
	}

	printlnC("{blue}Info:{reset} creating gitignore")

	if exists(".gitignore") {
		contents, err := os.ReadFile(".gitignore")
		if err != nil {
			printlnC("{red}Error checking contents of gitignore:", err.Error(), "{reset}")
			return
		}

		if len(contents) > 0 {
			if onWarn == 0 {
				printlnC("{yellow}Warning: gitignore already exists and is not empty{reset}")
				answer := scanLine("Do you want to override it? (Y/N) ")
				if answer == "" {
					os.Exit(0)
				}
				if answer[0] != 'Y' && answer[0] != 'y' {
					return
				}
			} else if onWarn == 2 {
				return
			}
		}
	}

	file, err := os.Create(".gitignore")
	if err != nil {
		ePrintlnC("{red}Error creating gitignore:", err.Error(), "{reset}")
		return
	}

	if exeName == "" {
		exeName, _ = strings.CutSuffix(scanLine("Executable name: "), ".exe")
	}

	if exeName == "" {
		os.Exit(0)
	}

	writer := bufio.NewWriter(file)
	writer.WriteString("# generated by go_project_creator: https://github.com/Peirceman/go_project_creator\n\n/")
	writer.WriteString(exeName)
	writer.WriteString("\n/")
	writer.WriteString(exeName)
	writer.WriteString(".exe\n")
	writer.Flush()

	file.Close()
}

func makeModule() {
	printlnC("{blue}Info:{reset} creating module")
	if moduleUrl == "" {
		moduleUrl = scanLine("Enter module url: ")
	}

	if moduleUrl == "" {
		os.Exit(0)
	}

	err := runCmd("go", "mod", "init", moduleUrl)

	if err != nil {
		ePrintlnC("{red}Error creating go module:", err.Error(), "{reset}")
		os.Exit(1)
	}
}

func makeProjectDir() {
	printlnC("{blue}Info:{reset} creating project directory")

	if dir == "" {
		dir = scanLine("Enter project directory name: ")
	}

	if dir == "" {
		os.Exit(0)
	}

	if exists(dir) {
		contents, err2 := os.ReadDir(dir)
		if len(contents) > 0 {
			if onWarn == 0 {
				printlnC("{yellow}Warning: directory", dir, "already exists and is not empty{reset}")
				answer := scanLine("Do you want to continue? (Y/N) ")

				if answer == "" || (answer[0] != 'Y' && answer[0] != 'y') {
					os.Exit(0)
				}
			} else if onWarn == 2 {
				os.Exit(0)
			}
		} else if err2 != nil {
			ePrintlnC("{red}Error checking contents of directory:", err2.Error(), "{reset}")
			os.Exit(1)
		}
	}

	err := os.Mkdir(dir, os.ModeDir)

	if err != nil {
		ePrintlnC("{red}Error creating directory", dir+":", err.(*os.PathError).Err.Error(), "{reset}")
		os.Exit(1)
	}

	os.Chdir(dir)
}
