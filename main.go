// File: cpp-package-manager/main.go
package main

import (
	"cpp-package-manager/pkg/config"
	"cpp-package-manager/pkg/resolver"
	"cpp-package-manager/pkg/types"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "init":
		handleInit()
	case "install":
		handleInstall(args)
	case "upgrade":
		handleUpgrade()
	case "uninstall":
		handleUninstall(args)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
	}
}

func handleInit() {
	if _, err := os.Stat(config.ConfigFile); !os.IsNotExist(err) {
		fmt.Println("cppkg.json already exists.")
		return
	}
	cfg := types.PackageConfig{
		Name:         "my-cpp-project",
		Version:      "0.1.0",
		Dependencies: make(map[string]string),
	}
	if err := config.SaveConfig(&cfg); err != nil {
		fmt.Printf("Error creating cppkg.json: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Initialized empty C++ project (created cppkg.json).")
}

func handleInstall(args []string) {
	if len(args) > 0 {
		if err := resolver.AddNewPackage(args[0]); err != nil {
			fmt.Printf("Error adding package %s: %v\n", args[0], err)
			os.Exit(1)
		}
	}
	if err := resolver.InstallDependencies(false); err != nil {
		fmt.Printf("Error installing dependencies: %v\n", err)
		os.Exit(1)
	}
}

func handleUpgrade() {
	fmt.Println("Upgrading all packages to the latest versions satisfying cppkg.json...")
	if err := resolver.InstallDependencies(true); err != nil {
		fmt.Printf("Error upgrading dependencies: %v\n", err)
		os.Exit(1)
	}
}

func handleUninstall(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: uninstall command requires a package name.")
		printUsage()
		os.Exit(1)
	}
	packageName := args[0]
	if err := resolver.UninstallPackage(packageName); err != nil {
		fmt.Printf("Error uninstalling package %s: %v\n", packageName, err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: cppkg <command> [arguments]")
	fmt.Println("\nCommands:")
	fmt.Println("  init          Initialize a new project (creates cppkg.json)")
	fmt.Println("  install       Install all dependencies from cppkg.json")
	fmt.Println("  install <url#version> Install a single new package and add to cppkg.json")
	fmt.Println("  upgrade       Upgrade all packages to their latest allowed versions")
	fmt.Println("  uninstall <name>  Remove a dependency from the project")
}
