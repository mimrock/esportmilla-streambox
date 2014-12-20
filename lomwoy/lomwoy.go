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
)

type StreamDisplay struct {
	twitch.StreamS
	Featured bool
}

type lomwoyTheme struct {
	Streams []twitch.StreamS
	W       *http.ResponseWriter
	Data    *lomwoyData
}

type lomwoyData struct {
	ColorScheme      ColorScheme
	PrimaryStreams   map[string][]StreamDisplay
	SecondaryStreams map[string][]StreamDisplay
	DisplayFeatured  bool
	DisplaySecondary bool
}

type ColorScheme struct {
	Background string
	Header     string
	HeaderFont string
	Font       string
}

func NewLomwoyTheme(streams []twitch.StreamS, w *http.ResponseWriter, values url.Values) *lomwoyTheme {
	//TODO:
	// Add Featured streams to the Top list (see todo)
	// Iterate through all games
	// Check each game if it has less stream than X
	// If yes, add some streams to it
	// Render secondary block
	l := new(lomwoyTheme)
	l.Streams = streams
	l.W = w
	l.Data = new(lomwoyData)
	l.Data.PrimaryStreams = make(map[string][]StreamDisplay)
	l.Data.SecondaryStreams = make(map[string][]StreamDisplay)
	l.setFeaturedStreams(values)
	l.addStreamsByGame(l.Data.PrimaryStreams, values, 5)
	l.addStreamsByGame(l.Data.SecondaryStreams, values, 100)
	// TODO Fill up primary box
	//l.setSecondaryStreams(values, 5)
	l.setColorScheme()
	if len(l.Data.SecondaryStreams) > 0 {
		l.Data.DisplaySecondary = true
	} else {
		l.Data.DisplaySecondary = false
	}
	return l
}

func (theme *lomwoyTheme) Render() {
	t := template.Must(template.ParseFiles("lomwoy/templates/base.html"))
	t.Execute(*theme.W, theme.Data)
	// TODO Always have a primary stream block. If there are no featured streams
	// Then the top X streams should be in the primary block.
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
				streamDisplay := StreamDisplay{stream, true}
				if _, present := theme.Data.PrimaryStreams[stream.Game]; present {
					theme.Data.PrimaryStreams[stream.Game] = append(theme.Data.PrimaryStreams[stream.Game], streamDisplay)
				} else {
					theme.Data.PrimaryStreams[stream.Game] = []StreamDisplay{streamDisplay}
				}
				break
			}
		}
		if len(featuredChannels) < 1 {
			break
		}
	}
}

func (theme *lomwoyTheme) addStreamsByGame(streams map[string][]StreamDisplay, queryParams url.Values, maxStreams int) {
	// theme.Data.SecondaryStreams = make(map[string][]twitch.StreamS)
	gameListString := queryParams.Get("g")
	if len(gameListString) < 1 {
		return
	}
	gameList := strings.Split(gameListString, "|")

	for _, game := range gameList {
		//log.Println("Game:", game)
		for i := 0; i < len(theme.Streams); i++ {
			//log.Println("i:", i, "stream Name:", theme.Streams[i].Channel.Name)
			if len(streams[game]) >= maxStreams {
				//log.Println("There is enough streams for this game. Break!", len(streams[game]))
				break
			}
			if theme.Streams[i].Game == game {
				//theme.Data.SecondaryStreams[game] = append(theme.Data.SecondaryStreams[game], stream)
				if _, ok := streams[game]; !ok {
					log.Println(game, "is missing")
					streams[game] = []StreamDisplay{}
				}
				sd := StreamDisplay{theme.Streams[i], false}
				streams[game] = append(streams[game], sd)
				deleteFromStreams(&theme.Streams, i)
				i--
			}
		}
	}
}

func deleteFromStreams(s *[]twitch.StreamS, place int) {
	a := *s
	if len(a)+1 < place {
		return
	} else {
		//copy(a[place:], a[place+1:])
		// a[len(a)-1] = nil // or the zero value of T
		//a = a[:len(a)-1]
		a = append(a[:place], a[place+1:]...)
	}
	*s = a
}
