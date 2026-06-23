<p align="center">
  <img src="logo.png" alt="Hacdot Convert Logo" width="80" height="80">
</p>

<h1 align="center">Hacdot Convert</h1>

<p align="center">
  A fast, minimal desktop video converter built with <a href="https://wails.io">Wails v2</a> (Go + HTML/CSS/JS).<br>
  Drag-and-drop video files and convert them using hardware-accelerated FFmpeg encoders — with zero bloat.
</p>

<p align="center">
  <img src="screen shot/Screenshot 2026-06-22 180030.png" alt="Hacdot Convert Screenshot" width="800">
</p>

---

## Features

- Drag-and-drop video file input
- Hardware-accelerated encoding (NVENC, QSV, AMF) with software fallback
- Bundled FFmpeg/ffprobe — no external install required

## Tech Stack

| Layer    | Technology                      |
|----------|---------------------------------|
| Runtime  | [Wails v2](https://wails.io)    |
| Backend  | Go 1.22                         |
| Frontend | Vanilla HTML / CSS / JavaScript |
| Bundler  | Wails embedded assets           |
| Video    | FFmpeg / ffprobe 6.1 (bundled)  |


## Project Structure

```
hacdot-convert/
├── main.go              # Entry point — bootstraps the Wails app
├── app.go               # Core application logic & Go-JS bindings
├── app_embedded.go      # Embedded asset helpers
├── app_lite.go          # Lightweight app variant
├── go.mod / go.sum      # Go module files
├── wails.json           # Wails project configuration
├── frontend/
│   ├── index.html       # Single-page app shell
│   ├── main.js          # UI logic & Wails runtime calls
│   ├── style.css        # Monochrome design system styles
│   └── wailsjs/         # Auto-generated Wails JS bindings
├── binaries/            # Bundled FFmpeg/ffprobe executables
├── build/               # Wails build output & app icons
└── design.md            # Design system specification
```

## Hardware Acceleration

The app probes for available GPU encoders at startup and selects the best available option:

| Priority | Encoder        | Vendor        |
|----------|---------------|---------------|
| 1        | `h264_nvenc`  | NVIDIA        |
| 2        | `h264_qsv`    | Intel (QSV)   |
| 3        | `h264_amf`    | AMD           |
| 4        | `libx264`     | CPU (fallback)|

## License

LGPL
