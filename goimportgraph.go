// goimportgraph is a small CLI listing (currently only) Git repositories behind
// `go list -mod=readonly -m all` by crawling the module paths for go-import.
package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/sa6mwa/goimportgraph/which"
	"golang.org/x/net/html"
)

var (
	goArgs = []string{"go", "list", "-mod=readonly", "-m", "all"}
)

type verbosePrinter struct {
	verbose bool
}

var v verbosePrinter

func (vp *verbosePrinter) print(a ...any) {
	if vp.verbose {
		fmt.Fprint(os.Stderr, a...)
	}
}
func (vp *verbosePrinter) printf(format string, a ...any) {
	if vp.verbose {
		fmt.Fprintf(os.Stderr, format, a...)
	}
}
func (vp *verbosePrinter) println(a ...any) {
	if vp.verbose {
		fmt.Fprintln(os.Stderr, a...)
	}
}
func (vp *verbosePrinter) printlnf(format string, a ...any) {
	if vp.verbose {
		fmt.Fprintf(os.Stderr, format+"\n", a...)
	}
}
func (vp *verbosePrinter) fatalf(format string, a ...any) {
	vp.printf(format, a...)
	os.Exit(1)
}
func (vp *verbosePrinter) fatallnf(format string, a ...any) {
	vp.printlnf(format, a...)
	os.Exit(1)
}
func (vp *verbosePrinter) fatalln(a ...any) {
	vp.println(a...)
	os.Exit(1)
}
func (vp *verbosePrinter) fatal(a ...any) {
	vp.println(a...)
	os.Exit(1)
}

func getRepoURL(goListMod string) (string, error) {
	modules := strings.Fields(goListMod)
	if len(modules) < 1 {
		return "", fmt.Errorf("not a module: %s", goListMod)
	}
	u, err := url.Parse(modules[0])
	if err != nil {
		return "", err
	}
	u.Scheme = "https"
	v := url.Values{}
	v.Set("go-get", "1")
	u.RawQuery = v.Encode()

	splitPath := strings.Split(u.Path, "/")
	if len(splitPath) == 0 {
		return "", fmt.Errorf("path error: %s", u.String())
	}
	switch {
	case strings.EqualFold(splitPath[0], "github.com"):
		if len(splitPath) > 3 {
			u.Path = strings.Join(splitPath[:3], "/")
		}
	}

	resp, err := http.Get(u.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%v: unable to %s %s", resp.StatusCode, http.MethodGet, u.String())
	}
	goImportContentStrings, err := getMetaGoImportContent(resp.Body)
	if err != nil {
		return "", err
	}
	if len(goImportContentStrings) >= 3 && goImportContentStrings[1] != "git" {
		return "", fmt.Errorf("%s: unsupported vcs %s", u.String(), goImportContentStrings[1])
	}
	if len(goImportContentStrings) >= 3 {
		return goImportContentStrings[2], nil
	}
	return "", fmt.Errorf("unable to extract ")
}

func getMetaGoImportContent(reader io.Reader) (strs []string, err error) {
	var ErrorUnableToFindGoImportMetaTag string = "unable to find go-import meta tag"
	tokenizer := html.NewTokenizer(reader)
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			return []string{}, errors.New(ErrorUnableToFindGoImportMetaTag)
		case html.StartTagToken:
			t := tokenizer.Token()
			if t.Data == "meta" {
				gotName := false
				for _, attr := range t.Attr {
					if attr.Key == "name" && attr.Val == "go-import" {
						gotName = true
					}
					if attr.Key == "content" {
						strs = strings.Fields(attr.Val)
					}
				}
				if gotName && len(strs) > 0 {
					return
				}
			}
		}
	}
}

func internalizeModuleName(prefix, goModuleString, suffix string) string {
	s := strings.Fields(goModuleString)
	if len(s) == 0 {
		return ""
	}
	replacer := strings.NewReplacer(
		".", "_",
		"/", "_",
		"!", "",
		":", "",
		"+", "",
	)
	return prefix + replacer.Replace(s[0]) + suffix
}

func main() {
	quiet := flag.Bool("q", false, "be more quiet")
	chDir := flag.String("C", "", fmt.Sprintf("change to `directory` before executing %s", strings.Join(goArgs, " ")))
	printPackageName := flag.Bool("n", false, "print package name after repo URL")
	internalize := flag.Bool("z", false, "internalize package name instead of the actual name (string replacer)")
	prefix := flag.String("p", "", "`prefix` to internalized module name")
	suffix := flag.String("s", "", "`suffix` to internalized module name")
	readable := flag.Bool("r", false, "more human readable output")
	flag.Parse()
	if *quiet {
		v.verbose = false
	} else {
		v.verbose = true
	}

	if len(*chDir) > 0 {
		err := os.Chdir(*chDir)
		if err != nil {
			v.fatalln(err)
		}
	}

	goPath := which.Which(goArgs[0])
	if goPath == "" {
		v.fatallnf("Error: unable to find `%s` in PATH=%s", goArgs[0], os.Getenv("PATH"))
	}

	cmd := exec.Command(goPath, goArgs[1:]...)
	var cmdOut bytes.Buffer
	var cmdErr bytes.Buffer
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdErr
	err := cmd.Run()
	if err != nil {
		v.printf("Error executing %s: %s", goPath, cmdErr.String())
		v.fatalln(err)
	}
	s := bufio.NewScanner(bytes.NewReader(cmdOut.Bytes()))
	for s.Scan() {
		repoURL, err := getRepoURL(s.Text())
		if err != nil {
			v.printlnf("Error: %v", err)
			continue
		}
		if *printPackageName {
			pkgName := strings.Fields(s.Text())[0]
			if *internalize {
				pkgName = internalizeModuleName(*prefix, s.Text(), *suffix)
				if len(pkgName) < 3 {
					pkgName = "NA/USE_URL"
				}
			}
			if *readable {
				fmt.Printf("%s => %s\n", repoURL, pkgName)
			} else {
				fmt.Printf("%s %s\n", repoURL, pkgName)
			}
		} else {
			fmt.Println(repoURL)
		}
	}
}
