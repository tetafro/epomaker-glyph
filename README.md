# epomaker-glyph

A small tool to send the current date and time to an Epomaker Glyph keyboard.

## Build

```sh
sudo apt install libudev-dev
go build -o ./bin/epomaker-glyph .
```


## Usage

List all HID devices (to find the product ID):

```sh
./bin/epomaker-glyph -list
```

Send the current time:

```sh
sudo ./bin/epomaker-glyph -pid 0x5002
```

Send a specific time:

```sh
sudo ./bin/epomaker-glyph -pid 0x5002 --time "2026-04-10T15:00:00Z00:00"
```

---

Original project (Python, Epomaker RT100 keyboard): [strodgers/epomaker-controller](https://github.com/strodgers/epomaker-controller).
