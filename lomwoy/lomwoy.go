// Lomwoy theme
package lomwoy

import (
	//"fmt"
	"errors"
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
	Background       string
	HeaderBackground string
	HeaderFont       string
	FeaturedFont     string
	Font             string
}

func NewColorScheme() *ColorScheme {
	// Default colors.
	cs := new(ColorScheme)
	cs.Background = "d9dde0"
	cs.HeaderBackground = "254673"
	cs.HeaderFont = "ffffff"
	cs.FeaturedFont = "13173e"
	cs.Font = "13173e"
	return cs
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
	l.setColorScheme(values)
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

func (theme *lomwoyTheme) setColorScheme(queryParams url.Values) {
	cs := NewColorScheme()
	bkg := queryParams.Get("bkg")
	if err := validateColor(bkg); err == nil {
		cs.Background = bkg
	}

	header := queryParams.Get("headbkg")
	if err := validateColor(header); err == nil {
		cs.HeaderBackground = header
	}

	headerFont := queryParams.Get("headfont")
	if err := validateColor(headerFont); err == nil {
		cs.HeaderFont = headerFont
	}

	featuredFont := queryParams.Get("featfont")
	if err := validateColor(featuredFont); err == nil {
		cs.FeaturedFont = featuredFont
	}

	font := queryParams.Get("font")
	if err := validateColor(font); err == nil {
		cs.FeaturedFont = font
	}

	theme.Data.ColorScheme = *cs
}

func (theme *lomwoyTheme) setFeaturedStreams(queryParams url.Values) {
	featuredChannelsString := queryParams.Get("f")
	if len(featuredChannelsString) < 1 {
		return
	}
	featuredChannels := strings.Split(featuredChannelsString, "|")

	//for i, stream := range theme.Streams {
	for i := 0; i < len(theme.Streams); i++ {
		stream := theme.Streams[i]
		for j := 0; j < len(featuredChannels); j++ {
			channel := featuredChannels[j]
			//for j, channel := range featuredChannels {
			if stream.Channel.Name == channel {
				featuredChannels = append(featuredChannels[:j], featuredChannels[j+1:]...)
				theme.Streams = append(theme.Streams[:i], theme.Streams[i+1:]...)
				streamDisplay := StreamDisplay{stream, true}
				if _, present := theme.Data.PrimaryStreams[stream.Game]; present {
					theme.Data.PrimaryStreams[stream.Game] = append(theme.Data.PrimaryStreams[stream.Game], streamDisplay)
				} else {
					theme.Data.PrimaryStreams[stream.Game] = []StreamDisplay{streamDisplay}
				}
				i--
				j--
				break
			}
		}
		if len(featuredChannels) < 1 {
			break
		}
	}
}

func (theme *lomwoyTheme) addStreamsByGame(streams map[string][]StreamDisplay, queryParams url.Values, maxStreams int) {
	gameListString := queryParams.Get("g")
	if len(gameListString) < 1 {
		return
	}
	gameList := strings.Split(gameListString, "|")

	for _, game := range gameList {
		for i := 0; i < len(theme.Streams); i++ {
			if len(streams[game]) >= maxStreams {
				//There is enough streams for this game.
				break
			}
			if theme.Streams[i].Game == game {
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

func validateColor(color string) error {
	if len(color) != 3 && len(color) != 6 {
		return errors.New("Invalid length. Color length must be 3 or 6 chars.")
	}
	for _, code := range color {
		if !((code >= 48 && code <= 57) || (code >= 65 && code <= 70) || (code >= 97 && code <= 102)) {
			return errors.New("Invalid char.")
		}
	}
	return nil
}
