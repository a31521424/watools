# WaTools

[![MIT License](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/a31521424/watools/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/a31521424/watools)](https://goreportcard.com/report/github.com/a31521424/watools)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-lightgrey.svg)](https://wails.io)

An open-source, lightweight, and extensible productivity toolbox inspired by uTools and Alfred.

WaTools aims to provide a fast, modern, and cross-platform alternative for common development and daily tasks, accessible via a simple global hotkey.

---

## 📸 Screenshot

*A picture is worth a thousand words. Please add a screenshot of the application here.*

![WaTools Screenshot](https://raw.githubusercontent.com/a31521424/watools/main/screenshot.png)

---

## ✨ Features

-   **Global Access**: Instantly open the app from anywhere with a global hotkey (`Alt+Space`).
-   **App Launcher**: Quickly find and launch applications on your system.
-   **Extensible Commands**: Built-in support for a variety of commands.
-   **Simple Calculator**: Perform basic calculations directly in the search bar.
-   **Modern UI**: Clean and intuitive user interface built with React and Tailwind CSS.
-   **Cross-Platform**: Built with Wails, aiming for full support on macOS, Windows, and Linux.

---

## 🛠️ Technology Stack

-   **Backend**: [Go](https://golang.org/)
-   **Framework**: [Wails](https://wails.io/) (v2)
-   **Frontend**: [React](https://reactjs.org/), [TypeScript](https://www.typescriptlang.org/)
-   **Styling**: [Tailwind CSS](https://tailwindcss.com/)
-   **UI Components**: [shadcn/ui](https://ui.shadcn.com/)

---

## 🚀 Getting Started (For Developers)

To get a local copy up and running for development, follow these simple steps.

### Prerequisites

-   Go (v1.21+)
-   Node.js (v18+)
-   Wails CLI: Follow the [official Wails installation guide](https://wails.io/docs/gettingstarted/installation).

### Installation & Running

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/a31521424/watools.git
    ```
2.  **Navigate to the project directory:**
    ```sh
    cd watools
    ```
3.  **Install frontend dependencies:**
    ```sh
    cd frontend && npm install && cd ..
    ```
4.  **Run in development mode:**
    This command starts the application with live-reloading for both the Go backend and the React frontend.
    ```sh
    wails dev
    ```
5.  **Build the application:**
    To build a production-ready binary for your platform, run:
    ```sh
    wails build
    ```
    The executable will be available in the `build/bin/` directory.

---

## 🤝 Contributing

Contributions, issues, and feature requests are welcome! Feel free to check the [issues page](https://github.com/a31521424/watools/issues).

---

## 📄 License

This project is licensed under the MIT License - see the `LICENSE` file for details.
