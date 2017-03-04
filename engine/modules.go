package engine

import (
	"log"
	"os"
	"os/user"
	"path/filepath"
	"plugin"
	"strings"
)

// ModuleFileName defines the pattern of module file names
var ModuleFileName = "talk-*.so"

// ModulesUserSubDir defines the sub-directory per user for modules
var ModulesUserSubDir = ".talk/modules"

// LoadModules load modules from specified directory,
// if empty use default directories
func LoadModules(dirs []string) {
	if len(dirs) == 0 {
		dirs = DefaultModulesDirs()
	}
	for _, dir := range dirs {
		matches, err := filepath.Glob(filepath.Join(dir, ModuleFileName))
		if err != nil {
			log.Printf("Warning: %s: %v\n", dir, err)
			continue
		}
		for _, m := range matches {
			if st, err := os.Stat(m); err != nil || st.IsDir() {
				continue
			}
			log.Printf("Loading %s\n", filepath.Base(m))
			_, e := plugin.Open(m)
			if e != nil {
				log.Printf("Error(%s): %v\n", m, e)
			}
		}
	}
}

func execFile() string {
	execFn, err := os.Executable()
	if err == nil {
		if realFn, e := filepath.EvalSymlinks(execFn); e == nil {
			execFn = realFn
		}
	} else {
		execFn, _ = os.Getwd()
	}
	return execFn
}

// DefaultModulesDirs returns the default modules load directory
func DefaultModulesDirs() (dirs []string) {
	execDir, _ := filepath.Split(execFile())
	execDir = filepath.Clean(execDir)
	if strings.HasSuffix(execDir, "/bin") {
		execDir = execDir[0:len(execDir)-3] + "lib"
	}
	dirs = append(dirs, execDir)
	if u, err := user.Current(); err == nil {
		dirs = append(dirs, filepath.Join(u.HomeDir, ModulesUserSubDir))
	}

	return dirs
}
