# Project State Summary: gimme-that-iso

This document summarizes the current state of the `gimme-that-iso` project as of the end of our session.

## Project Goal

The objective was to create a robust, command-line tool for securely downloading large files (like Linux ISOs) based on a JSON configuration. The tool is designed to be run locally, in Docker, or as a scheduled CronJob in a Kubernetes cluster.

## Current Status: Complete & Functional

All requested features have been implemented, tested, and are working correctly. The application is considered complete and is fully operational.

## Implemented Features

*   **Concurrent Downloads**: The application uses a worker pool of goroutines to download multiple files simultaneously for high performance.
*   **Secure Verification**:
    *   **GPG Signatures**: Automatically verifies the integrity of downloaded files using GPG signatures. It is smart enough to handle two different verification flows: signatures that cover a checksum file (like Debian) and signatures that cover the ISO file directly (like Alpine).
    *   **Checksum Validation**: Verifies SHA512 or SHA256 checksums after GPG verification passes.
*   **Pipeline-Friendly Logging**: All output is designed for clarity in automated environments.
    *   Every log line is prefixed with a standard timestamp (e.g., `2025/12/14 00:08:05`).
    *   Each log line is tagged with a `[Worker ID]` and the full `[ISO Name]` for easy tracking of concurrent jobs.
*   **Detailed Progress**: Download progress is logged periodically with percentage complete, file sizes, average speed in **MB/s**, and an estimated time of arrival (ETA).
*   **Configuration**: All downloads are configured via an `isos.json` file.
*   **Deployment Ready**:
    *   A multi-stage `Dockerfile` is provided to build a minimal and secure application image.
    *   A `cronjob.yaml` Kubernetes manifest is provided for easy deployment as a scheduled task.
*   **Startup Banner**: The application displays your chosen ASCII art banner on startup, read directly from an embedded `banner.txt` file.

## Key Files

*   `cmd/gimme-that-iso/main.go`: The main application entrypoint.
*   `internal/config/`: Package for loading `isos.json`.
*   `internal/downloader/`: Package for handling file downloads and progress logging.
*   `internal/verifier/`: Package for all GPG and checksum logic.
*   `internal/ui/`: Package for displaying the startup banner.
*   `Dockerfile`: Builds the final application container.
*   `cronjob.yaml`: Kubernetes deployment manifest.
*   `README.md`: Full documentation.
