package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/abenz1267/elephant/v2/internal/util"
	"github.com/abenz1267/elephant/v2/pkg/pb/pb"
	"github.com/fhs/gompd/v2/mpd"
)

func Query(conn net.Conn, query string, single, exact bool, _ uint8) []*pb.QueryResponse_Item {
	entries := []*pb.QueryResponse_Item{}
	searchType, searchTerm := "any", query
	isAlbumSearch, isArtistSearch, isGenreSearch := false, false, false

	if query == "" || utf8.RuneCountInString(query) < 1 {
		if nowPlaying := NowPlayingMPD(); nowPlaying != nil {
			queue := GetQueueItems()
			for i, entry := range queue {
				if i == 0 {
					entry.Text = fmt.Sprintf("%s: %s", "Next", strings.TrimPrefix(entry.Text, "0. "))
				}
				entries = append(entries, entry)
			}
			entries = append(entries, nowPlaying)
		}
		return entries
	}

	if strings.HasPrefix(query, config.AlbumPrefix) {
		searchType, searchTerm, isAlbumSearch = "album", strings.TrimSpace(strings.TrimPrefix(query, config.AlbumPrefix)), true
	}

	if strings.HasPrefix(query, config.ArtistPrefix) {
		searchType, searchTerm, isArtistSearch = "artist", strings.TrimSpace(strings.TrimPrefix(query, config.ArtistPrefix)), true
	}

	if strings.HasPrefix(query, config.GenrePrefix) {
		searchType, searchTerm, isGenreSearch = "genre", strings.TrimSpace(strings.TrimPrefix(query, config.GenrePrefix)), true
	}

	queryResults := SearchMpd(searchTerm, searchType)

	if !isAlbumSearch && !isArtistSearch && !isGenreSearch {
		sort.Slice(queryResults, func(i, j int) bool {
			return CalculateScore(queryResults[i], searchTerm) > CalculateScore(queryResults[j], searchTerm)
		})
	}

	for i, res := range queryResults {
		track, disc, title := GetField(res, "Track", "track"), GetField(res, "Disc", "disc"), GetField(res, "Title", "title")
		artist, album, date := GetField(res, "Artist", "artist"), GetField(res, "Album", "album"), GetField(res, "Date", "date")

		displayText := title
		if isAlbumSearch || isArtistSearch {
			displayText = fmt.Sprintf("%s. %s", track, title)
		}
		if dNum, _ := strconv.Atoi(strings.Split(disc, "/")[0]); dNum > 1 {
			displayText = fmt.Sprintf("%d-%s. %s", dNum, track, title)
		}

		subTextInfo := fmt.Sprintf("%s ~ %s", artist, album)
		if isArtistSearch && date != "" {
			subTextInfo = fmt.Sprintf("%s [%s] ~ %s", artist, date, album)
		}

		finalScore := int32(1000 - i)
		if !isAlbumSearch && !isArtistSearch {
			finalScore = CalculateScore(res, searchTerm)
		}

		item := &pb.QueryResponse_Item{
			Text:       displayText,
			Subtext:    subTextInfo,
			Identifier: res["file"],
			Provider:   Name,
			Actions:    []string{"play", "pause", "add_to_queue", "play_album", "play_next", "toggle_preview"},
			Score:      finalScore,
		}

		if showAlbumArt {
			item.Preview = GetAlbumArt(res["file"])
			item.PreviewType = util.PreviewTypeFile
		} else {
			item.Preview = GetTrackInfoText(res)
			item.PreviewType = util.PreviewTypeText
		}

		entries = append(entries, item)
	}

	return entries
}

func SearchMpd(queryString string, searchType string) []mpd.Attrs {
	mpdConnection, err := getMpdConnection() // take a shot every time you see these 5 lines of code
	if err != nil {
		return nil
	}
	defer releaseMpdConnection(mpdConnection)

	results, err := mpdConnection.Search(searchType, queryString)
	if (err != nil || len(results) == 0) && strings.Contains(queryString, " ") {
		results, _ = mpdConnection.Search(searchType, strings.Split(queryString, " ")[0])
	}

	if len(results) == 0 {
		return nil
	}

	if searchType == "album" || searchType == "artist" {
		sort.Slice(results, func(i, j int) bool {
			a, b := results[i], results[j]
			if searchType == "artist" {
				dA, dB := GetField(a, "Date", "date"), GetField(b, "Date", "date")
				if dA != dB {
					if config.SortBy == "newest" {
						return dA > dB
					}
					return dA < dB
				}
			}
			alA, alB := GetField(a, "Album", "album"), GetField(b, "Album", "album")
			if alA != alB {
				return alA < alB
			}
			return GetField(a, "Track", "track") < GetField(b, "Track", "track")
		})
	}

	return results
}

func GetGenres(current mpd.Attrs, client *mpd.Client) ([]string, error) {
	return client.List("genre", "file", current["file"])
}

func NowPlayingMPD() *pb.QueryResponse_Item {
	mpdConnection, err := getMpdConnection()
	if err != nil {
		return nil
	}
	defer releaseMpdConnection(mpdConnection)

	current, err := mpdConnection.CurrentSong()
	if err != nil || current == nil {
		return nil
	}

	genres, _ := GetGenres(current, mpdConnection)
	for _, genre := range genres {
		slog.Debug(fmt.Sprintf("%v", genre))
	}

	name, desc := fmt.Sprintf("Now Playing: %s", current["Title"]), fmt.Sprintf("%s - %s", current["Artist"], current["Album"])
	if current["Title"] == "" {
		name, desc = "Nothing Playing", "It's pretty quiet in here..."
	}

	item := &pb.QueryResponse_Item{
		Text:        name,
		Subtext:     desc,
		Identifier:  current["file"],
		Provider:    Name,
		Actions:     []string{"playpause", "shuffle_library", "clear_queue", "toggle_preview"},
		Preview:     GetAlbumArt(current["file"]),
		PreviewType: util.PreviewTypeFile, Score: 1000,
	}

	if showAlbumArt {
		item.Preview = GetAlbumArt(current["file"])
		item.PreviewType = util.PreviewTypeFile
	} else {
		item.Preview = GetTrackInfoText(current)
		item.PreviewType = util.PreviewTypeText
	}

	return item

}

func GetQueueItems() []*pb.QueryResponse_Item {
	mpdConnection, err := getMpdConnection()
	if err != nil {
		return nil
	}
	defer releaseMpdConnection(mpdConnection)

	queue, _ := mpdConnection.PlaylistInfo(-1, -1)
	entries := []*pb.QueryResponse_Item{}

	for i, track := range queue {
		if i == 0 {
			continue
		}
		score, _ := strconv.Atoi(track["Pos"])
		entries = append(entries, &pb.QueryResponse_Item{
			Text:        fmt.Sprintf("%s. %s", strconv.Itoa(i-1), track["Title"]),
			Subtext:     track["Artist"],
			Identifier:  "queue:" + track["Id"],
			Provider:    Name,
			Actions:     []string{"play", "remove_from_queue", "clear_queue"},
			Preview:     GetAlbumArt(track["file"]),
			PreviewType: util.PreviewTypeFile, Score: int32(999 - score),
		})
	}
	return entries
}

func CalculateScore(res mpd.Attrs, query string) int32 {
	normalizedQuery, normalizedTitle := NormalizeFields(query), NormalizeFields(GetField(res, "Title", "title"))
	var score int32
	if normalizedTitle == normalizedQuery {
		score = 500
	} else if strings.HasPrefix(normalizedTitle, normalizedQuery) {
		score = 250
	} else if strings.Contains(normalizedTitle, normalizedQuery) {
		score = 128
	}
	return score
}

func GetField(m map[string]string, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func NormalizeFields(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "&", "and")

	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == ' ' {
			b.WriteRune(r)
		}
	}

	return strings.Join(strings.Fields(b.String()), " ")
}

func GetAlbumArt(trackPath string) string {
	if config.MusicDir == "" {
		return ""
	}
	dir := filepath.Dir(filepath.Join(config.MusicDir, trackPath))
	exts, names := []string{".jpg", ".jpeg", ".png", ".webp"}, []string{"cover", "Cover", "folder", "Folder"}
	for _, name := range names {
		for _, ext := range exts {
			if _, err := os.Stat(filepath.Join(dir, name+ext)); err == nil {
				return filepath.Join(dir, name+ext)
			}
		}
	}
	return ""
}

func GetTrackInfoText(res mpd.Attrs) string {
	title := GetField(res, "Title", "title")
	artist := GetField(res, "Artist", "artist")
	album := GetField(res, "Album", "album")
	date := GetField(res, "Date", "date")
	genre := GetField(res, "Genre", "genre")
	format := GetField(res, "Format", "format")
	file := GetField(res, "file", "file")

	return fmt.Sprintf("Title:  %s\nArtist: %s\nAlbum:  %s\nGenre:  %s\nYear:   %s\nFormat: %s\nFile:   %s",
		title, artist, album, genre, date, format, file)
}
