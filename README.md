# watools

An open-source, lightweight productivity toolbox inspired by uTools.

## Core Technology

This project is built using the Wails framework (v2), combining a Go backend with a web-based frontend.

## Platform Status

This project aims for cross-platform compatibility. However, initial development and stabilization efforts are prioritized for macOS.

## How to Build and Run from Source

To compile and run this project on your local machine, you need to have Go and the Wails CLI installed. Follow the official Wails installation guide if you haven't already.

1. Clone the repository:
   git clone https://github.com/YOUR_USERNAME/watools.git

2. Navigate into the project directory:
   cd watools

3. To build the production application:
   wails build

   This will create a distributable binary for your platform in the `build/bin/` directory. You can run the application from there.

4. To run the application in development (live-reload mode):
   wails dev

   This command will build and run the application, automatically watching for file changes in both the Go backend and the frontend. The application will hot-reload as you make code changes. Press Ctrl+C in the terminal to stop the development server.

## Current Features

- Quick Launch: Global hotkey (Alt+Space) to open the main window.
- Simple Calculator: Evaluate basic mathematical expressions.
- (Add your other completed features here)

## Contributing

Contributions are welcome. Please open an issue to discuss any proposed changes or new features.