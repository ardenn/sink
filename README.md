# Sink

Sink is a simple, secure, and fast file upload service written in Go. It exposes a POST endpoint to accept files via `multipart/form-data` and saves them to a configurable local directory.

## Features

- **Simple API**: A single `POST /upload` endpoint.
- **Secure**:
  - Prevents directory traversal attacks.
  - Mitigates timing attacks when validating the authentication token.
  - Limits maximum upload size to prevent DoS attacks.
- **Configurable**: Driven by a simple YAML configuration file.
- **Containerised**: Ready to be deployed as an OCI (Docker or Podman) container.

## Configuration

Sink uses a `config.yaml` file for configuration. By default, it looks for `/app/config.yaml` when running in a container. You can override this location by setting the `CONFIG_PATH` environment variable pointing to a location from the container's perspective (if running in a container), or to a path accessible to the binary executable.

```yaml
port: 8080
uploadDir: "./uploads"
authToken: "secret-token-change-me"
maxFileSizeMb: 50
```

- `port`: The port the service will listen on (default: `8080`).
- `uploadDir`: The directory where uploaded files will be saved (default: `./uploads`).
- `authToken`: The token required in the `X-Auth-Token` header for authentication.
- `maxFileSizeMb`: The maximum allowed file size in megabytes (default: `10`).

## Running the Service

### 1. Building and Running the Binary Directly

Ensure you have Go 1.26 or later installed to build from source.

1.  **Clone the repository:**
    ```bash
    git clone git@github.com:ardenn/sink.git
    cd sink
    ```

2.  **Build the binary:**
    ```bash
    go build -o sink main.go
    ```

3.  **Configure:** Create or edit `config.yaml` to suit your needs.

4.  **Run:**
    ```bash
    chmod +x ./sink
    ./sink
    ```

**Important Note on Permissions:** The process running the binary must have read, write, and execute permissions for the directory specified in `uploadDir`. If the directory does not exist, the service will attempt to create it on startup (with `0750` permissions, meaning read, write, and execute for the owner, and read and execute for the group).

### 2. Running as a Container (Docker / Podman)

A `Dockerfile` is provided to run the service in a lightweight, secure container. The container runs as a non-root user.

1.  **Build the container image:**
    ```bash
    docker build -t sink .
    # or
    podman build -t sink .
    ```

2.  **Run the container:**
    ```bash
    docker run -p 8080:8080 -d --name sink sink
    # or
    podman run -p 8080:8080 -d --name sink sink
    ```

**Important Note on Container Volumes:** 
If you want to persist the uploaded files outside of the container, you should mount a volume. Because the container runs as a non-root user, you may encounter permission denied errors if the mounted host directory does not allow the container user to write to it.

A common approach is to create the directory on the host and ensure it's writable by the container user (e.g., using `chmod` or `chown`, or letting Docker create it if it has the right permissions).

```bash
# Example: Mounting a volume
mkdir -p ./appdata/uploads
chmod 777 ./appdata/uploads # (Adjust for your specific security needs)

docker run -p 8080:8080 -v $(pwd)/appdata/uploads:/appdata/uploads -d --name sink sink
```

### 3. Running with Docker Compose

You can also use Docker Compose to manage and run the service using the prebuilt image from the GitHub Container Registry. Create a `docker-compose.yml` file with the following configuration:

```yaml
services:
  sink:
    image: ghcr.io/ardenn/sink:latest
    ports:
      - "8080:8080"
    volumes:
      - ./appdata/uploads:/appdata/uploads
      - ./app:/app # Mount the app/ directory instead of config.yaml directly
    userns_mode: keep-id # Fixes permission issues in rootless Podman
    restart: unless-stopped
```

Ensure the `./appdata/uploads` (or whatever it's set to on the host) directory exists and has the appropriate permissions as mentioned above. The `userns_mode: keep-id` setting ensures the container user matches your host user, preventing permission issues with mounted volumes when running with rootless Podman. Then, start the service in the background:

```bash
docker compose up -d
```

## Usage

To upload a file, send a `POST` request to the `/upload` endpoint with the file included as `multipart/form-data` under the `file` field. Include your configured `authToken` in the `X-Auth-Token` header.

Using `curl`:

```bash
curl -X POST \
  -H "X-Auth-Token: secret-token-change-me" \
  -F "file=@/path/to/your/local/file.txt" \
  http://localhost:8080/upload
```

If successful, you will receive an `HTTP 200 OK` response.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
