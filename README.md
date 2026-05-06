# sshc

`sshc` is a modern SSH configuration helper and manager written in Go. It helps you keep your main `~/.ssh/config` clean by managing modular configuration files in a dedicated directory using the `Include` directive.

## Features

- **Safe Initialization**: Automatically backs up your existing SSH config and sets up the `Include` directive.
- **Modular Management**: Keep separate config files for different projects or environments in `~/.ssh/sshc.d/`.
- **Integrated Key Generation**: Automatically generate RSA, Ed25519, or ECDSA keys when adding new configurations.
- **Git-like CLI**: Intuitive commands like `add`, `rm`, `list`, `show`, and `edit`.

## Installation

### From Source

Ensure you have Go 1.26 or later installed.

```bash
# Clone the repository
git clone https://github.com/jmpeax/sshc.git
cd sshc

# Build and install
make build
sudo cp bin/sshc /usr/local/bin/
```

## Quick Start

### 1. Initialize

The first step is to prepare your environment. This command creates `~/.ssh/sshc.d/` and prepends `Include sshc.d/*` to your `~/.ssh/config`. A backup of your original config is created at `~/.ssh/config.backup`.

```bash
sshc init
```

### 2. Add a Configuration

Add a new managed config entry. By default, it uses the name as the host.

```bash
sshc add config my-server --host example.com --user admin
```

### 3. Add with Key Generation

You can generate a new SSH key specifically for this configuration:

```bash
# Create an Ed25519 key (default)
sshc add config github --host github.com --user git --create-key

# Create a 4096-bit RSA key
sshc add config legacy-server --create-key --key-type rsa --key-size 4096

# Create an Ed25519 key with a comment
sshc add config personal --create-key --key-comment "personal-key"
```

### 4. Manage Configurations

```bash
# List all managed configs
sshc list

# Show the content of a specific config
sshc show my-server

# Edit a config in your default editor
sshc edit config my-server

# Update specific fields of a config
sshc edit config my-server --user new-admin --host new-example.com

# Remove a managed config
sshc rm config my-server

# Remove a config and its associated SSH key
sshc rm config my-server --delete-key
```

## Examples

Here are some common scenarios where `sshc` makes managing your SSH configurations easier.

### 1. Multiple GitHub Accounts

Manage separate keys for your personal and work GitHub accounts without messing up your global config.

```bash
# Set up personal GitHub
sshc add config github-personal \
  --host github.com-personal \
  --hostname github.com \
  --user git \
  --create-key \
  --key-comment "personal@email.com"

# Set up work GitHub
sshc add config github-work \
  --host github.com-work \
  --hostname github.com \
  --user git \
  --create-key \
  --key-comment "work@company.com"
```

### 2. Multi-Environment Servers

Keep your Dev, Staging, and Production environments organized.

```bash
# Development server
sshc add config app-dev \
  --hostname dev.example.com \
  --user developer

# Staging server
sshc add config app-staging \
  --hostname staging.example.com \
  --user qa-team

# Production server
sshc add config app-prod \
  --hostname prod.example.com \
  --user admin \
  --forward-agent yes
```

### 3. Using a Jump Host (ProxyJump)

Configure access to a private server through a secure bastion/jump host.

```bash
# 1. Add the bastion host
sshc add config bastion \
  --hostname bastion.example.com \
  --user jump-user

# 2. Add the internal server using the bastion as a proxy
sshc add config internal-db \
  --hostname 10.0.1.50 \
  --user db-admin \
  --proxy-jump bastion
```

### 4. Legacy Systems with Specific Requirements

For older servers that require specific key types or ports.

```bash
# Generate a 4096-bit RSA key for a legacy system
sshc add config legacy-box \
  --hostname old-server.local \
  --port 2222 \
  --user sysadmin \
  --create-key \
  --key-type rsa \
  --key-size 4096
```

### 5. Managing Cloud Instances

Easily manage cloud instances with specific identity files.

```bash
# Add a cloud instance using an existing downloaded .pem key
sshc add config aws-web \
  --hostname ec2-54-xx-xx-xx.compute-1.amazonaws.com \
  --user ubuntu \
  --identity ~/downloads/aws-prod-key.pem
```

## Command Reference

### `sshc init`
Initializes `sshc` environment.
- Creates `~/.ssh/sshc.d/` directory.
- Backs up `~/.ssh/config` to `~/.ssh/config.backup`.
- Prepends `Include sshc.d/*` to `~/.ssh/config`.

### `sshc add config NAME [flags]`
Adds a new SSH configuration file.
- `--host string`: SSH Host alias (e.g., my-server). Defaults to NAME.
- `--hostname string`: The real hostname or IP address (e.g., example.com).
- `--user string`: The SSH user.
- `--port int`: The SSH port.
- `--identity string`: Path to an existing identity file.
- `--forward-agent string`: Forward SSH Agent (`yes` or `no`).
- `--proxy-jump string`: SSH ProxyJump host.
- `--create-key`: Generate a new SSH key pair.
- `--key-type string`: Key type: `ed25519` (default), `rsa`, or `ecdsa`.
- `--key-size int`: Key size in bits (for RSA: 2048, 4096; for ECDSA: 256, 384, 521).
- `--key-comment string`: SSH key comment (e.g., user@host).

### `sshc rm config NAME`
Removes a managed SSH configuration file.

### `sshc list`
Lists all configuration files managed in `~/.ssh/sshc.d/`.

### `sshc show NAME`
Prints the content of a managed configuration file.

### `sshc edit config NAME [flags]`
Edits a managed SSH configuration file.
- If no flags are provided, it opens the file in your default editor (`$EDITOR`).
- `--host string`: Update the SSH Host alias.
- `--hostname string`: Update the real hostname/IP.
- `--user string`: Update the SSH user.
- `--port int`: Update the SSH port.
- `--identity string`: Update the path to identity file.
- `--forward-agent string`: Update Forward SSH Agent (`yes` or `no`).
- `--proxy-jump string`: Update SSH ProxyJump.

## Development

The project includes a `Makefile` for common tasks:

- `make build`: Compiles the binary to `bin/sshc`.
- `make test`: Runs the test suite.
- `make lint`: Runs the linter.
- `make clean`: Removes build artifacts.

## License

MIT
