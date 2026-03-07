#!/bin/bash
export GOTOOLCHAIN=go1.25.7
export GOEXPERIMENT=nodwarf5
export CGO_ENABLED=1
export GOAMD64=v1
go build -buildmode=plugin -ldflags="-s -w"  -trimpath -o mpd.so .

cp mpd.so $HOME/.config/elephant/providers/

