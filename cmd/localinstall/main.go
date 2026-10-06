// localinstall applies artifacts prepared by watoolsctl install.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"watools/config"
	"watools/internal/localinstall"
)

func main() {
	source := flag.String("source", "", "source checkout")
	goExecutable := flag.String("go", "", "Go executable")
	appDir := flag.String("app-dir", "/Applications", "application directory")
	bundle := flag.String("bundle", "", "prepared application")
	packages := flag.String("packages", "", "prepared official packages")
	noLaunch := flag.Bool("no-launch", false, "do not open the installed application")
	flag.Parse()
	if err := execute(*source, *goExecutable, *appDir, *bundle, *packages, *noLaunch); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func execute(source, goExecutable, appDir, bundle, packages string, noLaunch bool) error {
	content, err := os.ReadFile(filepath.Join(source, "wails.json"))
	if err != nil {
		return err
	}
	config.ParseProject(content)
	var paths []string
	if packages != "" {
		paths, err = filepath.Glob(filepath.Join(packages, "*.wt"))
		if err != nil {
			return err
		}
		if len(paths) == 0 {
			return fmt.Errorf("no prepared plugin packages")
		}
	}
	// Once application is swapped, finish the commit or rollback before responding
	// to cancellation. The parent waits for this process group before cleaning up.
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	value := localinstall.Configuration{SourceRoot: source, GoExecutable: goExecutable, ToolPath: os.Getenv("PATH"), ApplicationPath: filepath.Join(appDir, "watools.app"), Version: config.ProjectVersion()}
	backup, err := localinstall.Apply(context.Background(), config.ProjectDataDir(), localinstall.ApplyRequest{Configuration: value, Bundle: bundle, Packages: paths})
	if err != nil {
		return err
	}
	fmt.Printf("Installed successfully. Application: %s\nData: %s\n", value.ApplicationPath, config.ProjectDataDir())
	if backup != "" {
		fmt.Printf("Previous application retained: %s\n", backup)
	}
	if bundle != "" {
		fmt.Println("Save open editors and restart WaTools to use the new application. Running windows were preserved.")
	}
	// Opening is deliberately explicit after updates; never activate all detached windows.
	if bundle != "" && !noLaunch && backup == "" {
		return openApplication(value.ApplicationPath)
	}
	return nil
}
