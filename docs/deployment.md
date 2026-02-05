# Deployment

## Bare Metal

### 1. Build the binaries

```bash
git clone https://github.com/bitswalk/ldf.git
cd ldf
task build
```

This produces `build/bin/ldfd` and `build/bin/ldfctl`.

### 2. Install

```bash
sudo cp build/bin/ldfd /usr/local/bin/
sudo cp build/bin/ldfctl /usr/local/bin/
```

### 3. Create a service user

```bash
sudo useradd -r -s /usr/sbin/nologin -d /var/lib/ldfd ldfd
sudo mkdir -p /var/lib/ldfd
sudo chown ldfd:ldfd /var/lib/ldfd
```

### 4. Create configuration

```bash
sudo mkdir -p /etc/ldfd
sudo cp docs/samples/ldfd.yml /etc/ldfd/ldfd.yml
sudo chown ldfd:ldfd /etc/ldfd/ldfd.yml
```

Edit `/etc/ldfd/ldfd.yml` to set your desired configuration. See [Configuration](configuration.md) for all options.

### 5. Run

```bash
sudo -u ldfd ldfd --config /etc/ldfd/ldfd.yml
```

## Systemd

A sample systemd unit file is provided at `docs/samples/ldfd.service`.

### 1. Install the service

After building and installing the binary (see Bare Metal steps 1-4 above):

```bash
sudo cp docs/samples/ldfd.service /etc/systemd/system/ldfd.service
sudo systemctl daemon-reload
```

### 2. Configure secrets (optional)

For S3 storage credentials, create an environment file:

```bash
sudo touch /etc/ldfd/ldfd.env
sudo chmod 600 /etc/ldfd/ldfd.env
sudo chown ldfd:ldfd /etc/ldfd/ldfd.env
```

Add credentials to `/etc/ldfd/ldfd.env`:

```
LDFD_STORAGE_S3_ACCESS_KEY=your-access-key
LDFD_STORAGE_S3_SECRET_KEY=your-secret-key
```

Then uncomment the `EnvironmentFile` line in the service file:

```ini
EnvironmentFile=-/etc/ldfd/ldfd.env
```

### 3. Start the service

```bash
sudo systemctl enable ldfd
sudo systemctl start ldfd
sudo systemctl status ldfd
```

### 4. View logs

```bash
sudo journalctl -u ldfd -f
```

### Service file details

The provided unit file includes security hardening:

- Runs as a dedicated `ldfd` user (non-root)
- `ProtectSystem=strict` -- Read-only filesystem except allowed paths
- `ProtectHome=yes` -- No access to home directories
- `PrivateTmp=yes` -- Isolated `/tmp`
- `NoNewPrivileges=yes` -- Cannot gain additional privileges
- `ReadWritePaths=/var/lib/ldfd` -- Only writable path
- `LimitNOFILE=65536` -- File descriptor limit
- Automatic restart on failure with 5-second delay

## Docker

A multi-stage Dockerfile is provided at `tools/docker/Dockerfile`.

### Build the image

```bash
docker build -f tools/docker/Dockerfile -t ldf:latest .
```

### Run the container

```bash
docker run -d \
  --name ldfd \
  -p 8443:8443 \
  -v ldfd-data:/var/lib/ldfd \
  ldf:latest
```

### With custom configuration

Mount a config file:

```bash
docker run -d \
  --name ldfd \
  -p 8443:8443 \
  -v ldfd-data:/var/lib/ldfd \
  -v /path/to/ldfd.yml:/opt/ldf/config/ldfd.yml:ro \
  ldf:latest
```

### With S3 storage

Pass credentials via environment variables:

```bash
docker run -d \
  --name ldfd \
  -p 8443:8443 \
  -v ldfd-data:/var/lib/ldfd \
  -e LDFD_STORAGE_S3_ENDPOINT=s3.example.com \
  -e LDFD_STORAGE_S3_PROVIDER=garage \
  -e LDFD_STORAGE_S3_BUCKET=ldf-distributions \
  -e LDFD_STORAGE_S3_ACCESS_KEY=your-key \
  -e LDFD_STORAGE_S3_SECRET_KEY=your-secret \
  ldf:latest
```

### Container details

The Docker image:

- Uses Alpine Linux as the runtime base
- Runs as a non-root `ldf` user (uid 1000)
- Includes build tools for kernel compilation (gcc, make, linux-headers)
- Copies `ldfd`, `ldfctl` binaries and WebUI assets
- Uses the sample config at `/opt/ldf/config/ldfd.yml`
- Exposes port 8443

## Storage Setup

### Local storage

Local storage is the default. Artifacts are stored in a directory on the filesystem:

```yaml
storage:
  type: local
  local:
    path: /var/lib/ldfd/artifacts
```

Ensure the ldfd user has write access to this directory.

### S3-compatible storage

ldfd supports four S3 provider types, each with different URL construction:

#### GarageHQ

```yaml
storage:
  type: s3
  s3:
    provider: garage
    endpoint: s3.example.com
    region: garage
    bucket: ldf-distributions
```

Garage uses `api.{endpoint}` for the API and `{bucket}.{endpoint}` for web access.

#### MinIO

```yaml
storage:
  type: s3
  s3:
    provider: minio
    endpoint: minio.example.com:9000
    region: us-east-1
    bucket: ldf-distributions
```

MinIO uses the endpoint directly with path-style addressing.

#### AWS S3

```yaml
storage:
  type: s3
  s3:
    provider: aws
    region: us-east-1
    bucket: ldf-distributions
```

AWS uses `s3.{region}.amazonaws.com` automatically.

#### Generic S3-compatible

```yaml
storage:
  type: s3
  s3:
    provider: other
    endpoint: s3.example.com
    region: us-east-1
    bucket: ldf-distributions
```

Generic provider uses path-style addressing with the endpoint directly.

For all S3 providers, pass credentials via environment variables rather than config files:

```bash
export LDFD_STORAGE_S3_ACCESS_KEY="your-access-key"
export LDFD_STORAGE_S3_SECRET_KEY="your-secret-key"
```
