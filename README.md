# epomaker-glyph

A small tool to send the current date and time to an Epomaker Glyph keyboard.

## Install

Download a [pre-build binary](https://github.com/tetafro/epomaker-glyph/releases).

Or build from source:

```sh
sudo apt install libudev-dev
go install github.com:tetafro/epomaker-glyph@latest
```

## Usage

List all HID devices (to find the product ID):

```sh
epomaker-glyph -list
```

Send the current time:

```sh
sudo epomaker-glyph -pid 0x5002
```

Send a specific time:

```sh
sudo epomaker-glyph -pid 0x5002 -time "2026-04-10T15:00:00"
```

---

Original project (Python, Epomaker RT100 keyboard): [strodgers/epomaker-controller](https://github.com/strodgers/epomaker-controller).
