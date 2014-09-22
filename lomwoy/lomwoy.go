// Lomwoy theme
package lomwoy

import (
	//"fmt"
	"github.com/mrshankly/go-twitch/twitch"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	//"strconv"
	//"strings"
)

type lomwoyTheme struct {
	Streams []twitch.StreamS
	W       *http.ResponseWriter
	Data    *lomwoyData
}

type lomwoyData struct {
	ColorScheme      ColorScheme
	FeaturedStreams  map[string][]twitch.StreamS
	SecondaryStreams map[string][]twitch.StreamS
	DisplayFeatured  bool
}

type ColorScheme struct {
	Background string
	Header     string
	HeaderFont string
	Font       string
}

func NewLomwoyTheme(streams []twitch.StreamS, w *http.ResponseWriter, values url.Values) *lomwoyTheme {
	l := new(lomwoyTheme)
	l.Streams = streams
	l.W = w
	l.Data = new(lomwoyData)
	l.setFeaturedStreams(values)
	l.setSecondaryStreams(values, 5)
	if len(l.Data.FeaturedStreams) > 0 {
		l.Data.DisplayFeatured = true
	} else {
		l.Data.DisplayFeatured = false
	}
	return l
}

func (theme *lomwoyTheme) Render() {
	t := template.Must(template.ParseFiles("lomwoy/templates/base.html"))
	t.Execute(*theme.W, theme.Data)
}

func (theme *lomwoyTheme) setColorScheme() {
	cs := new(ColorScheme)
	cs.Background = "d9dde0"
	cs.Header = "254673"
	cs.HeaderFont = "ffffff"
	cs.Font = "13173e"
	theme.Data.ColorScheme = *cs
}

func (theme *lomwoyTheme) setFeaturedStreams(queryParams url.Values) {
	theme.Data.FeaturedStreams = make(map[string][]twitch.StreamS)
	featuredChannelsString := queryParams.Get("f")
	if len(featuredChannelsString) < 1 {
		return
	}
	featuredChannels := strings.Split(featuredChannelsString, "|")

	for i, stream := range theme.Streams {
		for j, channel := range featuredChannels {
			if stream.Channel.Name == channel {
				featuredChannels = append(featuredChannels[:j], featuredChannels[j+1:]...)
				theme.Streams = append(theme.Streams[:i], theme.Streams[i+1:]...)
				if _, present := theme.Data.FeaturedStreams[stream.Game]; present {
					theme.Data.FeaturedStreams[stream.Game] = append(theme.Data.FeaturedStreams[stream.Game], stream)
				} else {
					theme.Data.FeaturedStreams[stream.Game] = []twitch.StreamS{stream}
				}
				break
			}
		}
		if len(featuredChannels) < 1 {
			break
		}
	}
}

func (theme *lomwoyTheme) setSecondaryStreams(queryParams url.Values, streamCount int) {
	theme.Data.SecondaryStreams = make(map[string][]twitch.StreamS)
	gameListString := queryParams.Get("g")
	if len(gameListString) < 1 {
		return
	}
	gameList := strings.Split(gameListString, "|")

	for _, game := range gameList {
		streamsAdded := 0
		for _, stream := range theme.Streams {
			if stream.Game == game {
				streamsAdded++
				theme.Data.SecondaryStreams[game] = append(theme.Data.SecondaryStreams[game], stream)
			} else {
				log.Println(stream.Game, "!=", game)
			}
			if streamsAdded == streamCount {
				break
			}
		}

	}
}
