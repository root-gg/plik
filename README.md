[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)
[![Build](https://github.com/root-gg/plik/actions/workflows/master.yaml/badge.svg)](https://github.com/root-gg/plik/actions/workflows/master.yaml)
[![Go Report](https://img.shields.io/badge/Go_report-A+-brightgreen.svg)](http://goreportcard.com/report/root-gg/plik)
[![Docker Pulls](https://img.shields.io/docker/pulls/rootgg/plik.svg)](https://hub.docker.com/r/rootgg/plik)
[![GoDoc](https://godoc.org/github.com/root-gg/plik?status.svg)](https://godoc.org/github.com/root-gg/plik)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](http://opensource.org/licenses/MIT)

Want to access the documentation? **https://root-gg.github.io/plik**

Want to see a live demo ? **https://plik.root.gg**

Want to chat with us ? Telegram channel : **https://t.me/plik_rootgg**

# Plik

Plik is a scalable & friendly temporary file upload system — like WeTransfer, self-hosted.

### Features

- 🖥️ Modern Vue 3 web interface
- 🧑‍💻 Powerful [Command line client](https://root-gg.github.io/plik/features/cli-client.html)
- ☁️ Multiple storage backends (local, S3, OpenStack Swift, Google Cloud Storage)
- 🗄️ Multiple metadata backends (SQLite, PostgreSQL, MySQL)
- 🔑 Multiple authentication providers (Local, Google, OVH, OIDC)
- ⏱️ Configurable TTL with auto-cleanup
- 💣 OneShot downloads (file deleted after first download)
- ⚡ Stream mode (uploader → downloader, nothing stored)
- 🔐 Password-protected uploads (BasicAuth)
- 🔒 End-to-end encryption with [Age](https://age-encryption.org/) (CLI ↔ Web interoperable)
- 📦 Archive directly from CLI/Web
- 📊 Prometheus metrics
- 🤖 [MCP server](https://root-gg.github.io/plik/features/mcp-server.html) for AI assistant integration

### Third-party clients

   - [ShareX](https://getsharex.com/) Uploader : Directly integrated into ShareX
   - [plikSharp](https://github.com/iss0/plikSharp) : A .NET API client for Plik
   - [Filelink for Plik](https://gitlab.com/joendres/filelink-plik) : Thunderbird Addon to upload attachments to Plik

### Quick Start

```bash
# Docker
docker run -p 8080:8080 rootgg/plik

# From release
wget https://github.com/root-gg/plik/releases/download/1.4.0/plik-server-1.4.0-linux-amd64.tar.gz
tar xzvf plik-server-1.4.0-linux-amd64.tar.gz
cd plik-server-1.4.0-linux-amd64/server && ./plikd

# Debian / Ubuntu
curl -fsSL https://root-gg.github.io/plik/apt/gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/plik.gpg
echo "deb [signed-by=/etc/apt/keyrings/plik.gpg] https://root-gg.github.io/plik/apt stable main" | sudo tee /etc/apt/sources.list.d/plik.list
sudo apt update && sudo apt install plik-server
sudo systemctl start plikd

# From source
git clone https://github.com/root-gg/plik.git
cd plik/webapp && npm install && cd ..
make
cd server && ./plikd

# Kubernetes (Helm)
helm repo add plik https://root-gg.github.io/plik
helm install plik plik/plik
```

Open web interface at [http://127.0.0.1:8080](http://127.0.0.1:8080)

### Command Line Client

```bash
$ plik myfile.txt
Upload successfully created at Sat, 21 Feb 2026 09:02:54 CET :
    http://127.0.0.1:8080/#/?id=vDPmPEUqc5oCt31T

myfile.txt :  2.56 KiB / 2.56 KiB [=========================================] 100.00% 719.15 KiB/s 0s

Commands :
curl -s "http://127.0.0.1:8080/file/vDPmPEUqc5oCt31T/UZzSdZ7zPgfRiTem/myfile.txt" > 'myfile.txt'

# or with just curl
$ curl --form 'file=@/path/to/myfile.txt' http://127.0.0.1:8080
https://plik.root.gg/file/eeBKaTQhg5xv0zTL/WWVhZc0PFtvoZgCu/myfile.txt
```

See: [CLI Client Documentation](https://root-gg.github.io/plik/features/cli-client.html) for installation

### How to Contribute

Contributions are welcome! See the [contributing guide](https://root-gg.github.io/plik/contributing) for development setup and build instructions.

### License

[MIT](LICENSE)
