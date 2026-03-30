#!/bin/bash
export GOTOOLCHAIN=go1.26.1
export GOEXPERIMENT=nodwarf5
export CGO_ENABLED=1
export GOAMD64=v1
go build -buildmode=plugin -ldflags="-s -w"  -trimpath -o mpd.so .

rm $HOME/.config/elephant/providers/mpd.so
mv mpd.so $HOME/.config/elephant/providers/

