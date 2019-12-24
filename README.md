# WS2Connect

[![Documentation](https://img.shields.io/badge/documentation-latest-green)](https://reddec.github.io/ws2connect/)
[![license](https://img.shields.io/github/license/reddec/ws2connect.svg)](https://github.com/reddec/ws2connect)
[![](https://godoc.org/github.com/reddec/ws2connect?status.svg)](http://godoc.org/github.com/reddec/ws2connect)
[![donate](https://img.shields.io/badge/help_by️-donate❤-ff69b4)](http://reddec.net/about/#donate)


Expose any TCP service over websocket. 

* Single binary
* Pre-built for all major OS (see installation)
* Few resource consumption
* Blazing fast
* Supports multiple endpoints with multiple mappings
* Supports TLS (HTTPS) serving


## Examples


* Expose some FIX API (host: `example.com:9823`) over websocket on `http://127.0.0.1:8080/ws` path

```bash
ws2connect /ws:example.com:9823
```

* Expose several services

```bash
ws2connect /ws:example.com:9823 /another-ws:host:9912
```

* Change binding to `8888` port

```bash
ws2connect -b 0.0.0.0:8888 /ws:example.com:9823
```

* Server over HTTPS (with `server.crt` and `server.key` files)

```bash
ws2connect --tls /ws:example.com:9823
```

## Usage

    Usage:
      ws2connect [OPTIONS] [endpoints]
    
    Application Options:
      -b, --binding=           HTTP binding address (default: :8080) [$BINDING]
      -t, --timeout=           Backend connection timeout (default: 15s) [$TIMEOUT]
          --graceful-shutdown= Delay before server shutdown (default: 15s) [$GRACEFUL_SHUTDOWN]
          --tls                Enable HTTPS serving with TLS [$TLS]
          --cert-file=         Path to certificate for TLS (default: server.crt) [$CERT_FILE]
          --key-file=          Path to private key for TLS (default: server.key) [$KEY_FILE]
      -q, --quiet              Disable logging [$QUIET]
    
    Help Options:
      -h, --help               Show this help message
    
    Arguments:
      endpoints:               Endpoint mapping (/path:address)


## Installation

### Binary

* From [releases](releases) page
* From bintray repository for most debian-based distribution (trusty, xenial, bionic, buster, wheezy):
```bash
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys 379CE192D401AB61
echo "deb https://dl.bintray.com/reddec/ws2connect-debian {distribution} main" | sudo tee -a /etc/apt/sources.list
sudo apt install ws2connect
```

### From source

* Expected Go version at least 1.13 and upper
* `go get -v github.com/reddec/ws2connect/cmd/...`