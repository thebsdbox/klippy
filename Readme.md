
# Klippy

## A tool for helping deploy container-based applications

```
 __                 
/  \        _______________ 
|  |       /               \
@  @       | It looks      |
|| ||      | like you      |
|| ||   <--| are deploying |
|\_/|      | a container.  |
\___/      \_______________/
```



### Images

Image support is the only function available at the moment, and allows `klippy` to interact with any registry that supports the `Version 2` API (details here https://docs.docker.com/registry/spec/api/#detail).

#### List tags of an image

The command `klippy image tags --name <image>` query a registry and retrieve all tags for a particular image. If no Registry hostname is specified then `klippy` will default the hub.docker.com mimicking the same behaviour as the docker cli.

#### List commands used to build an image

The command `klippy image commands --name <image>` will again query a registry and retrieve all of the commands that were used to build the entire image. 

**NOTE** The lines in red are `NOP` lines, as in no actual commands are run within the layer.

**Example:**

```
klippy image commands --name library/golang
Layer    Command
0   WORKDIR /go
1   /bin/sh -c mkdir -p "$GOPATH/src" "$GOPATH/bin" \
       && chmod -R 777 "$GOPATH"
2   ENV PATH=/go/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
3   ENV GOPATH=/go
4   /bin/sh -c set -eux; dpkgArch="$(dpkg --print-architecture)"; case "${dpkgArch##*-}" in amd64) goRelArch='linux-amd64'; goRelSha256='2871270d8ff0c8c69f161aaae42f9f28739855ff5c5204752a8d92a1c9f63993' ;; armhf) goRelArch='linux-armv6l'; goRelSha256='bc601e428f458da6028671d66581b026092742baf6d3124748bb044c82497d42' ;; arm64) goRelArch='linux-arm64'; goRelSha256='25e1a281b937022c70571ac5a538c9402dd74bceb71c2526377a7e5747df5522' ;; i386) goRelArch='linux-386'; goRelSha256='52935db83719739d84a389a8f3b14544874fba803a316250b8d596313283aadf' ;; ppc64el) goRelArch='linux-ppc64le'; goRelSha256='f929d434d6db09fc4c6b67b03951596e576af5d02ff009633ca3c5be1c832bdd' ;; s390x) goRelArch='linux-s390x'; goRelSha256='93afc048ad72fa2a0e5ec56bcdcd8a34213eb262aee6f39a7e4dfeeb7e564c9d' ;; *) goRelArch='src'; goRelSha256='558f8c169ae215e25b81421596e8de7572bd3ba824b79add22fba6e284db1117'; echo >&2; echo >&2 "warning: current architecture ($dpkgArch) does not have a corresponding Go binary release; will be building from source"; echo >&2 ;; esac; url="https://golang.org/dl/go${GOLANG_VERSION}.${goRelArch}.tar.gz"; wget -O go.tgz "$url"; echo "${goRelSha256} *go.tgz" | sha256sum -c -; tar -C /usr/local -xzf go.tgz; rm go.tgz; if [ "$goRelArch" = 'src' ]; then echo >&2; echo >&2 'error: UNIMPLEMENTED'; echo >&2 'TODO install golang-any from jessie-backports for GOROOT_BOOTSTRAP (and uninstall after build)'; echo >&2; exit 1; fi; export PATH="/usr/local/go/bin:$PATH"; go version
5   ENV GOLANG_VERSION=1.11.1
6   /bin/sh -c apt-get update \
       && apt-get install -y --no-install-recommends g++ gcc libc6-dev make pkg-config \
       && rm -rf /var/lib/apt/lists/*
7   /bin/sh -c apt-get update \
       && apt-get install -y --no-install-recommends bzr git mercurial openssh-client subversion procps \
       && rm -rf /var/lib/apt/lists/*
8   /bin/sh -c set -ex; if ! command -v gpg > /dev/null; then apt-get update; apt-get install -y --no-install-recommends gnupg dirmngr ; rm -rf /var/lib/apt/lists/*; fi
9   /bin/sh -c apt-get update \
       && apt-get install -y --no-install-recommends ca-certificates curl netbase wget \
       && rm -rf /var/lib/apt/lists/*
10   CMD ["bash"]
11   ADD file:58d5c21fcabcf1eec94e8676a3b1e51c5fdc2db5c7b866a761f907fa30ede4d8 in /
```

#### Advanced Logging

This can be modified by turning logging upto 5 `--logLevel 5`