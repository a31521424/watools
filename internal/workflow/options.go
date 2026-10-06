package workflow

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type Options struct {
	Task, Port, Scope, AppDir string
	Debug, NoLaunch           bool
	PluginAction              string
	Plugins                   []string
}

func Parse(args []string) (Options, error) {
	o := Options{Task: "help", Port: "9245", Scope: "all", AppDir: "/Applications"}
	if len(args) == 0 {
		return o, nil
	}
	o.Task = args[0]
	args = args[1:]
	if o.Task == "--help" || o.Task == "-h" {
		o.Task = "help"
	}
	switch o.Task {
	case "help", "doctor", "setup", "bindings", "check", "check-race", "dev-build":
		if len(args) > 0 {
			return o, fmt.Errorf("%s accepts no arguments", o.Task)
		}
	case "dev", "browser":
		if len(args) > 0 {
			if len(args) != 2 || (args[0] != "--port" && args[0] != "-port") {
				return o, fmt.Errorf("use --port <1..65535, except 9246>")
			}
			port, err := strconv.Atoi(args[1])
			if err != nil || port < 1 || port > 65535 || port == 9246 {
				return o, fmt.Errorf("invalid Vite port")
			}
			o.Port = args[1]
		}
	case "build":
		if len(args) > 1 || len(args) == 1 && args[0] != "--debug" {
			return o, fmt.Errorf("build accepts only --debug")
		}
		o.Debug = len(args) == 1
	case "plugins":
		if len(args) == 0 {
			return o, fmt.Errorf("use plugins list|package|install [package-id...]")
		}
		o.PluginAction = args[0]
		o.Plugins = args[1:]
		if o.PluginAction != "list" && o.PluginAction != "package" && o.PluginAction != "install" || o.PluginAction == "list" && len(o.Plugins) > 0 {
			return o, fmt.Errorf("invalid plugin command")
		}
	case "install":
		for i := 0; i < len(args); i++ {
			flag := args[i]
			if flag == "--no-launch" {
				o.NoLaunch = true
				continue
			}
			if (flag != "--scope" && flag != "--app-dir" && flag != "--plugin") || i+1 == len(args) || strings.HasPrefix(args[i+1], "--") {
				return o, fmt.Errorf("invalid install option: %s", flag)
			}
			i++
			switch flag {
			case "--scope":
				o.Scope = args[i]
			case "--app-dir":
				var err error
				o.AppDir, err = filepath.Abs(args[i])
				if err != nil {
					return o, err
				}
			case "--plugin":
				o.Plugins = append(o.Plugins, args[i])
			}
		}
		if o.Scope != "all" && o.Scope != "application" && o.Scope != "plugins" {
			return o, fmt.Errorf("invalid installation scope")
		}
		if o.Scope == "application" && len(o.Plugins) > 0 {
			return o, fmt.Errorf("application installation cannot select plugins")
		}
	default:
		return o, fmt.Errorf("unknown command: %s", o.Task)
	}
	for _, id := range o.Plugins {
		if !strings.HasPrefix(id, "watools.plugin.") || strings.Trim(id, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-") != "" {
			return o, fmt.Errorf("invalid plugin package ID: %s", id)
		}
	}
	return o, nil
}
