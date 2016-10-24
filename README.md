# DLite

## Building

install dependencies

```sh
brew install opam golang
opam init
eval `opam config env`
opam pin add qcow-format git://github.com/mirage/ocaml-qcow#master
opam install uri qcow-format
go get -u github.com/jteeuwen/go-bindata/...
```

update dependencies (use this if you've already built the project before)

```sh
git submodule update --init
opam update
opam upgrade
```

build the binary

```sh
go generate
go build
```

## TODO

- write uninstall command to remove daemon plist, resolver config, nfs exports, ssh config, and user's instance
