// Which imitates the which(1) command.
package which

import (
	"os"
	"path/filepath"
	"strings"
)

// Which imitates the which(1) command and attempts to find an executable in one
// of the paths in the PATH environment variable. It is not Windows-compatible.
// Input is the executables base name. The function returns the full path of the
// executable if found, or empty string if not.
func Which(prog string) string {
	for _, p := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
		progPath := filepath.Join(p, prog)
		fi, err := os.Stat(progPath)
		if err != nil {
			continue
		}
		switch {
		case fi.IsDir():
			continue
		case fi.Mode()&0o111 != 0:
			return progPath
		}
	}
	return ""
}
