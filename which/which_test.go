package which

import (
	"os"
	"path"
	"strings"
	"testing"
)

func TestWhich(t *testing.T) {
	paths := []string{"/bin", "/usr/bin", "/usr/local/bin"}
	bins := []string{"sh", "ls", "dash", "bash", "basename"}
	okbins := []string{}

	for _, binary := range bins {
		for _, p := range paths {
			fi, err := os.Stat(path.Join(p, binary))
			if err != nil || (err == nil && (fi.IsDir() || fi.Mode()&0o111 == 0)) {
				continue
			}
			okbins = append(okbins, binary)
			break
		}
	}
	if len(okbins) == 0 {
		t.Error("unable to test Which as no binaries found on file system")
		return
	}

	if err := os.Setenv("PATH", strings.Join(paths, string(os.PathListSeparator))); err != nil {
		t.Fatal(err)
	}

	for _, binary := range okbins {
		fullPath := Which(binary)
		if fullPath == "" {
			t.Errorf("expected a full path for %s, got empty string", binary)
			continue
		}
		if ret := func() bool {
			for _, p := range paths {
				if strings.HasPrefix(fullPath, p) {
					return true
				}
			}
			return false
		}(); !ret {
			t.Errorf("expected %s in any of %s; got %s", binary, strings.Join(paths, ", "), fullPath)
			continue
		}
	}
	if Which(".should-not-be-found-anywhere-f02398j4f9082408fj208j4087c0870287204ijhcljh20837408jf") != "" {
		t.Error("expected not to be found, but did")
	}
}
