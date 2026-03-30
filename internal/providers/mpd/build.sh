#!/bin/bash
go build -buildvcs=false -buildmode=plugin -trimpath -o mpd.so .

rm $HOME/.config/elephant/providers/mpd.so
mv mpd.so $HOME/.config/elephant/providers/

