# gimme-that-iso

`gimme-that-iso` is a command-line tool that securely downloads Linux ISOs and other large files, verifying their integrity using GPG signatures and checksums. It is designed to be run as a standalone binary, in a Docker container, or as a CronJob in a Kubernetes cluster.

## Features

- **Declarative Downloads**: Configure all your ISO downloads in a single `isos.json` file.
- **Concurrent Downloads**: Downloads multiple ISOs at the same time for improved performance.
- **Secure**: Verifies downloads using both GPG signatures and checksums (SHA512/SHA256).
- **Memory Efficient**: Downloads files by streaming them to disk, not loading them into memory.
- **Container Ready**: Includes a multi-stage `Dockerfile` for building a minimal, secure image.
- **Kubernetes Native**: Comes with a `cronjob.yaml` manifest for automated, scheduled execution in Kubernetes.

## Configuration

The application is configured using an `isos.json` file. You can add multiple files to be downloaded concurrently.

**`isos.json` example:**
```json
{
  "isos": [
    {
      "name": "Debian Netinstall",
      "url": "https://cdimage.debian.org/cdimage/release/current/amd64/iso-cd/debian-13.2.0-amd64-netinst.iso",
      "signature_url": "https://cdimage.debian.org/cdimage/release/current/amd64/iso-cd/SHA512SUMS.sign",
      "checksum_file_url": "https://cdimage.debian.org/cdimage/release/current/amd64/iso-cd/SHA512SUMS",
      "gpg_key_url": "https://keyserver.ubuntu.com/pks/lookup?op=get&search=0xDA87E80D6294BE9B"
    },
    {
      "name": "Alpine Standard",
      "url": "https://dl-cdn.alpinelinux.org/alpine/v3.20/releases/x86_64/alpine-standard-3.20.0-x86_64.iso",
      "signature_url": "https://dl-cdn.alpinelinux.org/alpine/v3.20/releases/x86_64/alpine-standard-3.20.0-x86_64.iso.asc",
      "checksum_file_url": "https://dl-cdn.alpinelinux.org/alpine/v3.20/releases/x86_64/alpine-standard-3.20.0-x86_64.iso.sha256",
      "gpg_key_url": "https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x293ACD0907D9495A"
    }
  ]
}
```

## Usage

### Local Development

1.  **Build:**
    ```sh
    go build -o gimme-that-iso ./cmd/gimme-that-iso
    ```

2.  **Run:**
    You can specify the download directory and the number of concurrent workers.
    ```sh
    ./gimme-that-iso --download-dir /path/to/your/downloads --workers 4
    ```

    **Example Output:**
    ```
     Alpine Standard 209.00 MiB / 209.00 MiB [====================] 100% 0s ] 93.64 MiB/s
    Debian Netinstall 784.00 MiB / 784.00 MiB [====================] 100% 0s ] 18.34 MiB/s
    ```

### Docker

1.  **Build the image:**
    ```sh
    docker build -t your-registry/gimme-that-iso:latest .
    ```

2.  **Run the container:**
    Mount a local directory for configuration and another for the downloads.
    ```sh
    docker run --rm \
      -v $(pwd)/isos.json:/app/isos.json \
      -v $(pwd)/downloads:/downloads \
      your-registry/gimme-that-iso:latest \
      --download-dir=/downloads --workers=4
    ```

### Kubernetes

The `cronjob.yaml` manifest provides a template for running `gimme-that-iso` as a scheduled job in Kubernetes.

1.  **Prerequisites:**
    - A running Kubernetes cluster.
    - A `PersistentVolume` available to satisfy the `PersistentVolumeClaim` for storage. This could be backed by NFS, an SMB share (using a suitable CSI driver), or any other storage class.

2.  **Update the Image and Args:**
    Before applying the manifest, you **must** update the `image` field in `cronjob.yaml`. You can also adjust the number of workers.

    ```yaml
    # in cronjob.yaml
    ...
    containers:
    - name: gimme-that-iso
      # IMPORTANT: Replace this with the actual image from your registry.
      image: your-registry/gimme-that-iso:latest
      imagePullPolicy: IfNotPresent
      args:
        - "--download-dir=/downloads"
        - "--workers=4"
    ...
    ```

3.  **Deploy:**
    Apply the manifest to your cluster.
    ```sh
    kubectl apply -f cronjob.yaml
    ```

    This will create:
    - A `ConfigMap` named `gimme-that-iso-config` with the `isos.json` data.
    - A `PersistentVolumeClaim` named `gimme-that-iso-downloads` requesting 10Gi of storage.
    - A `CronJob` named `gimme-that-iso-cronjob` scheduled to run daily at 2 AM.
