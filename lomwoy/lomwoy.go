// Lomwoy theme
package lomwoy

import (
	"github.com/mrshankly/go-twitch/twitch"
	//"log"
	"strconv"
)

type lomwoyTheme struct {
	Streams []twitch.StreamS
}

func NewLomwoyTheme(streams []twitch.StreamS) *lomwoyTheme {
	//log.Println("newLomwoyTheme")
	l := new(lomwoyTheme)
	l.Streams = streams
	return l
}

func (theme *lomwoyTheme) Render() []byte {
	var output string
	output = "<html><body><ol>"

	for _, s := range theme.Streams {
		//fmt.Printf("%d - %s (%s) Status: %s Viewers: %d Url: %s Views: %d Name %s\n", i+1, s.Name, s.Game, s.Channel.Status, s.Viewers, s.Channel.Url, s.Channel.Views, s.Channel.Name)
		output += "<li>Status: " + s.Channel.Status + " Game: " + s.Game + " (" + strconv.Itoa(s.Viewers) + ")</li>"
	}

	output += "</ol></body></html>"
	return []byte(output)
}
