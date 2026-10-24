# bibverse

Prints the verse sung before the Gospel at today's Catholic Mass, read from the [USCCB daily readings](https://bible.usccb.org/daily-bible-reading), as one line on standard output. It is one self-contained binary for Linux, macOS and Windows, with TLS roots compiled in for a host without a CA bundle.

## Install

Each tagged release attaches binaries for Linux, macOS and Windows on `amd64` and `arm64`, a `SHA256SUMS` file and its signature. The Linux ones are statically linked, and the macOS and Windows ones link only the libraries of the operating system itself.

### Linux

```sh
arch=amd64
base=https://github.com/h3nc4/bibverse/releases/latest/download
wget "${base}/bibverse-linux-${arch}" "${base}/SHA256SUMS" "${base}/SHA256SUMS.asc"
wget -qO- https://h3nc4.com/dreamweaver.asc | gpg --import
gpg --verify SHA256SUMS.asc SHA256SUMS
sha256sum --ignore-missing -c SHA256SUMS
install -D -m 755 "bibverse-linux-${arch}" ~/.local/bin/bibverse
```

A good signature names key `3BC15CA502A834A6927527DA6B8614EABCBC4AA4`.

### macOS

`arm64` is Apple silicon and `amd64` is an Intel Mac.

```sh
arch=arm64
base=https://github.com/h3nc4/bibverse/releases/latest/download
curl -L -O "${base}/bibverse-darwin-${arch}" -O "${base}/SHA256SUMS"
grep "bibverse-darwin-${arch}$" SHA256SUMS | shasum -a 256 -c
sudo install -m 755 "bibverse-darwin-${arch}" /usr/local/bin/bibverse
```

The binary lacks an Apple Developer ID signature. A copy downloaded through a browser gets the quarantine attribute, and Gatekeeper then refuses to open it until `xattr -d com.apple.quarantine bibverse-darwin-${arch}` removes it. The `curl` download above skips that attribute.

### Windows

Download `bibverse-windows-amd64.exe` from the [latest release](https://github.com/h3nc4/bibverse/releases/latest), or `bibverse-windows-arm64.exe` on an Arm device, and run it from a terminal. PowerShell prints its checksum for comparison with `SHA256SUMS`:

```powershell
Get-FileHash .\bibverse-windows-amd64.exe
```

SmartScreen warns about an unsigned program on its first run, and "More info" then "Run anyway" starts it.

### Usage

Run it with `bibverse`. The output is plain text, so it pipes into anything, such as `bibverse | cowsay`.

## Behaviour

bibverse reads the USCCB readings feed, `https://bible.usccb.org/readings.rss`, and takes the newest day in it. The verse is the acclamation just before the Gospel, with the `R.` refrains removed and its reference appended:

```
I take no pleasure in the death of the wicked man, says the Lord, but rather in his conversion that he may live. — Ezekiel 33:11
```

| day | section printed |
| --- | --- |
| most of the year | Alleluia |
| Lent and Holy Week | Verse Before the Gospel |
| Easter Vigil, which lacks both | the first stanza of the last Responsorial Psalm |

A Holy Week acclamation such as "Hail to you, our King" is liturgical text without a scripture reference, so it prints without one. Where an acclamation offers an alternative after "Or", the first one is printed. A day with several Masses lists an empty item beside the Masses, and bibverse moves on to the next item that holds a verse.

The feed changes day at 04:30 US Eastern time, so before then it still holds yesterday.

The exit status is 0 on success. On failure it is 1, with the reason on standard error:

| message includes | cause |
| --- | --- |
| `network is unreachable`, `no such host` | the host is offline or DNS fails |
| `Client.Timeout exceeded` | the feed took longer than 15 seconds |
| an HTTP status such as `403 Forbidden` | USCCB refused the request |
| `no verse in the feed` | the markup changed and no section before a Gospel matched |

A daily CI run reads the live feed to catch a markup change early.

TLS roots come from the host when it has a bundle. Without one, the binary falls back to the Mozilla roots compiled into it, so a minimal system or an empty container still verifies the site.

## Build from source

Go 1.27 or newer.

```sh
git clone https://github.com/h3nc4/bibverse.git
cd bibverse
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bibverse ./cmd/bibverse
go test -race ./...
```

The build runs with `CGO_ENABLED=0`, while `-race` wants cgo and a C compiler.

## License

<!-- vale off -->
bibverse is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version. See [LICENSE](LICENSE).
<!-- vale on -->

The name and the idea come from [LordUbuntu/bibverse](https://github.com/LordUbuntu/bibverse), Copyright (C) 2023 Jacobus Burger, released under the MIT License.

The readings are fetched from USCCB each time bibverse runs, and the repository and its binaries contain none of their text. The Lectionary for Mass is Copyright © 2001, 1998, 1997, 1986, 1970 Confraternity of Christian Doctrine.
