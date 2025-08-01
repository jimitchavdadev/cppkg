# CPPKG: A Simple C++ Package Manager

CPPKG is a dependency manager for C++ projects, written in Go. It uses Git repositories as the source for packages and provides a modern, automated workflow for handling complex dependency graphs.

## 🎯 Objectives

The goal of this project is to provide a C++ package management experience similar to tools like NPM or Cargo. It achieves this through the following core features:

  * **Declarative Dependencies**: Manage your project's dependencies in a simple `cppkg.json` file.
  * **Git-Based**: Fetch packages directly from any Git repository.
  * **Semantic Versioning**: Specify dependency versions using SemVer ranges (`^1.2.0`, `~1.0.0`) for flexible and safe updates.
  * **Transitive Dependency Resolution**: Automatically discover and install dependencies of your dependencies.
  * **Reproducible Builds**: Generate a `cppkg.lock` file to pin the exact commit of every package, ensuring consistent builds across all environments.
  * **Build System Integration**: Automatically generate a `cppkg.cmake` file to seamlessly integrate downloaded dependencies with your CMake project.
  * **Lifecycle Commands**: A full suite of commands including `install`, `upgrade`, `uninstall`, and `init`.
  * **Extensibility**: Run custom commands at different stages of the lifecycle using `hooks`.

-----

## 📂 Project Structure

The project is organized into a `pkg` directory for modularity, with `main.go` serving as the CLI entry point.

```
├── go.mod
├── go.sum
├── LICENSE
├── main.go
├── pkg/
│   ├── config/
│   │   └── config.go
│   ├── git/
│   │   └── git.go
│   ├── resolver/
│   │   ├── conflicts/
│   │   │   └── resolve.go
│   │   ├── dependency/
│   │   │   └── discover.go
│   │   └── install.go
│   ├── types/
│   │   ├── types_extra.go
│   │   └── types.go
│   └── utils/
│       └── utils.go
└── README.md
```


-----

## ⚙️ How It Works

CPPKG automates the process of fetching and managing C++ libraries.

### Configuration Files

  * **`cppkg.json`**: The manifest file where you declare your project's direct dependencies and custom scripts.
  * **`cppkg.lock`**: An auto-generated file that locks the dependency tree to specific Git commits for reproducibility. **Do not edit this file manually.**
  * **`cppkg.cmake`**: An auto-generated file that tells CMake where to find the headers for all installed dependencies.

### Commands

The CLI provides several commands to manage your project:

  * **`cppkg init`**
    Initializes a new project by creating a `cppkg.json` file in the current directory.

  * **`cppkg install [url#version]`**

      - If run without arguments, it installs all dependencies listed in `cppkg.json` according to the `cppkg.lock` file if it exists, ensuring a reproducible build. If no lock file is present, it resolves all dependencies and creates one.
      - If run with a package string (e.g., `https://github.com/fmtlib/fmt.git#^10.0.0`), it adds the package to `cppkg.json` and then installs it.

  * **`cppkg upgrade`**
    Ignores the `cppkg.lock` file and attempts to find the newest possible versions of all packages that still satisfy the version constraints in `cppkg.json`. It then updates the lock file.

  * **`cppkg uninstall <name>`**
    Removes a package from `cppkg.json` and re-calculates the dependency tree, removing all now-unnecessary packages from your project.

  * **`hooks`**
    You can define a `scripts` block in your `cppkg.json` to run shell commands. Currently, `postinstall` is supported.

    ```json
    "scripts": {
      "postinstall": "cmake . && make"
    }
    ```

### Example Workflow

1.  **Initialize your project.**

    ```sh
    mkdir my-awesome-app && cd my-awesome-app
    cppkg init
    ```

2.  **Add a dependency.**

    ```sh
    # This will fetch the latest version of {fmt} in the 10.x.x range.
    cppkg install https://github.com/fmtlib/fmt.git#^10.0.0
    ```

    This updates `cppkg.json` and creates `cppkg.lock`, `cppkg.cmake`, and the `cpp_modules` directory.

3.  **Write your code.**

      * **`main.cpp`**
        ```cpp
        #include <fmt/core.h>

        int main() {
            fmt::print("CPPKG is working!\n");
        }
        ```
      * **`CMakeLists.txt`**
        ```cmake
        cmake_minimum_required(VERSION 3.10)
        project(MyAwesomeApp)

        # Let cppkg handle finding the dependencies.
        include(cppkg.cmake)

        add_executable(my_app main.cpp)
        ```

4.  **Build and run.**

    ```sh
    cmake -S . -B build
    cmake --build build
    ./build/my_app
    ```
