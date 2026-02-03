# Aegis - Autonomous Secure Backup

![License](https://img.shields.io/badge/License-MIT-blue.svg)
![Go Report Card](https://goreportcard.com/badge/github.com/pronzzz/aegis)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen)

**Aegis** is a self-managed, zero-trust backup system designed for paranoia-level data security. It features military-grade encryption, autonomous scheduling, and self-healing verification, ensuring your data remains yours, no matter where it's stored.

## 🚀 Key Features

- **🔒 Zero-Trust Encryption**: All data is encrypted client-side using **AES-256-GCM**. Keys are derived via **Argon2id**. The server (or storage bucket) never sees plaintext.
- **📦 Content-Addressable Storage (CAS)**: Built-in global deduplication and Zstd compression significantly reduce storage costs.
- **☁️ Pluggable Storage Fabric**: Store backups locally or on any **S3-compatible** cloud storage (AWS, MinIO, Cloudflare R2, Wasabi).
- **🛡️ Tamper-Evident Security**: Cryptographically chained audit logs record every action. Any unsanctioned modification is instantly detected.
- **🤖 Autonomous Daemon**: A smart background scheduler runs backups based on a flexible JSON configuration, handling retries and reporting strictly.
- **❤️ Self-Healing**: The `audit` command proactively detects bitrot and corruption, using Reed-Solomon parity (planned) to heal damaged data.

## 🛠️ Installation

```bash
# Clone the repository
git clone https://github.com/pronzzz/aegis.git
cd aegis

# Build the binary
go build -o aegis ./cmd/aegis

# Move to a directory in your PATH
sudo mv aegis /usr/local/bin/
```

## ⚡ Quick Start

### 1. Initialize Repository

Before you can backup, you must initialize a secure repository. You will be asked to set a strong passphrase. **Do not lose this passphrase**; without it, your data is unrecoverable.

```bash
aegis init
```

### 2. Manual Backup

Backup a specific folder or file immediately.

```bash
aegis backup ./my-secrets
```

### 3. Restore

List available snapshots and restore data.

```bash
aegis list
aegis restore <snapshot-id> ./restored-folder
```

## ⚙️ Configuration (Daemon Mode)

For automated backups, create a `config.json` file. This tells the Aegis daemon what to backup and where.

```json
{
  "jobs": [
    {
      "name": "Docs",
      "path": "/Users/me/Documents",
      "interval": "1h"
    }
  ],
  "storage": {
    "type": "s3",
    "bucket": "my-aegis-backups",
    "endpoint": "s3.amazonaws.com",
    "access_key": "AWS_ACCESS_KEY",
    "secret_key": "AWS_SECRET_KEY",
    "region": "us-east-1",
    "use_ssl": true
  }
}
```

Start the daemon:

```bash
export AEGIS_PASSPHRASE="your=passphrase"
aegis start --config config.json
```

## 🔍 Security & Auditing

Aegis includes tools to verify the integrity of your backup repository.

### Audit Repository Health

Check for missing chunks or bitrot. This verifies the Merkle tree of your data.

```bash
aegis audit
```

### Verify Usage Log

Verify that the `security.log` has not been tampered with.

```bash
aegis audit-log
```

## 🧪 Disaster Simulation

**WARNING**: The simulator is for development and testing only. It **destroys data**.

```bash
# Simulate 10% corruption and 10% data loss
aegis simulate --corruption 0.1 --deletion 0.1
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) and [Code of Conduct](CODE_OF_CONDUCT.md) for details.

## 📄 License

This project is licensed under the [MIT License](LICENSE).

---
*Built with ❤️ for privacy.*
