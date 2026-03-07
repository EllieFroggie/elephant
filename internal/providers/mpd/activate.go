package main

import (
	"math/rand/v2"
	"net"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/fhs/gompd/v2/mpd"
)

func Activate(single bool, identifier, action, query, args string, format uint8, conn net.Conn) {
	mpdConnection, err := getMpdConnection()
	if err != nil {
		return
	}
	defer releaseMpdConnection(mpdConnection)

	mpdConnection.Consume(true)

	if strings.HasPrefix(identifier, "queue:") {

		id, _ := strconv.Atoi(strings.TrimPrefix(identifier, "queue:"))
		switch action {
		case "play":
			mpdConnection.MoveID(id, 1)
			mpdConnection.Next()
		case "remove_from_queue":
			mpdConnection.DeleteID(id)
		case "clear_queue":
			ClearQueue()
		}

	} else {

		switch action {
		case "play":
			mpdConnection.Clear()
			mpdConnection.Add(identifier)
			mpdConnection.Play(-1)
		case "playpause":
			st, _ := mpdConnection.Status()
			if st["state"] == "pause" {
				mpdConnection.Play(-1)
			} else {
				mpdConnection.Pause(true)
			}
		case "add_to_queue":
			mpdConnection.Add(identifier)
		case "play_next":
			mpdConnection.AddID(identifier, 1)
		case "play_album":
			handlePlayAlbum(mpdConnection, identifier)
		case "shuffle_library":
			handleShuffle(mpdConnection)
		case "toggle_preview":
			showAlbumArt = !showAlbumArt
		}

	}

}

func handlePlayAlbum(mpdConnection *mpd.Client, identifier string) {
	songs, _ := mpdConnection.ListInfo(identifier)
	if len(songs) == 0 {
		return
	}

	album := GetField(songs[0], "Album", "album")
	artist := GetField(songs[0], "Artist", "artist")

	var albumSongs []mpd.Attrs
	var err error

	if album != "" && artist != "" {
		albumSongs, err = mpdConnection.Find("artist", artist, "album", album)
	}

	if err != nil || len(albumSongs) == 0 {
		dir := filepath.Dir(identifier)
		albumSongs, _ = mpdConnection.ListInfo(dir)
	}

	sort.Slice(albumSongs, func(i, j int) bool {
		trA, _ := strconv.Atoi(strings.Split(GetField(albumSongs[i], "Track", "track"), "/")[0])
		trB, _ := strconv.Atoi(strings.Split(GetField(albumSongs[j], "Track", "track"), "/")[0])
		return trA < trB
	})

	mpdConnection.Clear()
	for _, s := range albumSongs {
		if s["file"] != "" {
			mpdConnection.Add(s["file"])
		}
	}
	mpdConnection.Play(0)
}

func handleShuffle(mpdConnection *mpd.Client) {
	allFiles, err := mpdConnection.List("file")
	if err != nil || len(allFiles) == 0 {
		return
	}
	rand.Shuffle(len(allFiles), func(i, j int) { allFiles[i], allFiles[j] = allFiles[j], allFiles[i] })

	limit := config.MaxShuffle
	if len(allFiles) < limit {
		limit = len(allFiles)
	}

	mpdConnection.Clear()
	for i := 0; i < limit; i++ {
		mpdConnection.Add(allFiles[i])
	}
	mpdConnection.Play(0)
}

func ClearQueue() {
	mpdConnection, err := getMpdConnection()
	if err != nil {
		return
	}
	defer releaseMpdConnection(mpdConnection)

	status, _ := mpdConnection.Status()
	currentID := status["songid"]
	queue, _ := mpdConnection.PlaylistInfo(-1, -1)

	for _, t := range queue {
		if t["Id"] != currentID {
			id, _ := strconv.Atoi(t["Id"])
			mpdConnection.DeleteID(id)
		}
	}
}
