package main

import (
	"fmt"
	"os"

	"github.com/abenz1267/elephant/v2/pkg/common"
	"github.com/abenz1267/elephant/v2/pkg/pb/pb"
	"github.com/fhs/gompd/v2/mpd"
)

var (
	Name         = "mpd"
	NamePretty   = "Music Player Daemon"
	config       *Config
	mpdPool      chan *mpd.Client
	showAlbumArt = true
)

type Config struct {
	common.Config `koanf:",squash"`
	MpdSocket     string `koanf:"mpd_socket"`
	MusicDir      string `koanf:"music_dir"`
	AlbumPrefix   string `koanf:"albums_shortcut"`
	ArtistPrefix  string `koanf:"artists_shortcut"`
	GenrePrefix   string `koanf:"genres_shortcut"`
	SortBy        string `koanf:"sort_by"`
	MaxShuffle    int    `koanf:"max_shuffle_tracks"`
	MaxPoolSize   int    `koanf:"max_mpd_connections"`
}

func Setup() {
	config = &Config{
		MaxPoolSize:  5,
		MaxShuffle:   100,
		SortBy:       "newest",
		AlbumPrefix:  "@",
		ArtistPrefix: "!",
		GenrePrefix:  "#",
		MusicDir:     fmt.Sprintf("%s/Music", os.Getenv("HOME")),
		MpdSocket:    "localhost:6600",
		Config: common.Config{
			Icon:     "mpd",
			MinScore: 10,
		},
	}

	common.LoadConfig(Name, config)

	if config.NamePretty != "" {
		NamePretty = config.NamePretty
	}

	initPool(config.MaxPoolSize)
}

func initPool(size int) {
	mpdPool = make(chan *mpd.Client, size)
}

func getMpdConnection() (*mpd.Client, error) {
	select {
	case conn := <-mpdPool:
		if err := conn.Ping(); err != nil {
			conn.Close()
			return mpd.Dial("tcp", config.MpdSocket)
		}
		return conn, nil
	default:
		return mpd.Dial("tcp", config.MpdSocket)
	}
}

func releaseMpdConnection(conn *mpd.Client) {
	if conn == nil {
		return
	}

	select {
	case mpdPool <- conn:
	default:
		conn.Close()
	}
}

func Available() bool {
	return true
}

func Icon() string { return config.Icon }

func PrintDoc() {}

func HideFromProviderlist() bool { return false }

func State(p string) *pb.ProviderStateResponse { return &pb.ProviderStateResponse{} }
