package main

import (
	//"fmt"
	"github.com/mrshankly/go-twitch/twitch"
	"log"
	"net/http"
	"time"
	"strconv"
	"sync"
)


type StreamList struct {
	sync.RWMutex
	streams []twitch.StreamS
}

type streamboxHandler struct {
	streamList *StreamList
}

func newStreamboxHandler(streamList *StreamList) *streamboxHandler {
	log.Println("newStreamboxHandler")
	return &streamboxHandler{streamList: streamList}
}

func (sb *streamboxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("ServeHTTP")
	w.Write([]byte("<html><body><ol>"))
	sb.streamList.RLock()
	for _, s := range sb.streamList.streams {
		//fmt.Printf("%d - %s (%s) Status: %s Viewers: %d Url: %s Views: %d Name %s\n", i+1, s.Name, s.Game, s.Channel.Status, s.Viewers, s.Channel.Url, s.Channel.Views, s.Channel.Name)
		w.Write([]byte("<li>"))
		w.Write([]byte("Status: " + s.Channel.Status + " Game: " +s.Game + " (" + strconv.Itoa(s.Viewers) + ")"))
		w.Write([]byte("</li>"))
	}
	sb.streamList.RUnlock()
 	w.Write([]byte("</ol></body></html>"))
}

func getStreams() []twitch.StreamS {
	log.Println("Getting streams...")
	client := twitch.NewClient(&http.Client{})
	opt := &twitch.ListOptions{
		Limit:   100,
		Offset:  0,
		//Channel: "tsm_theoddone,trumpsc,hotshotgg,athenelive",
	}

	streams, err := client.Streams.List(opt)
	if err != nil {
		//log.Fatal(err)
		log.Println(err)
	}
	return streams.Streams
}

func Scheduler(streamList *StreamList) {
	refreshStreams := time.Tick(5000 * time.Millisecond)
	for {
		select {
		case <-refreshStreams:
			st := getStreams()
			streamList.Lock()
			streamList.streams = st
			streamList.Unlock()
		}
	}
}

func main() {
	log.Println("Starting up server...")
	streamList := new(StreamList)
	streamList.streams = getStreams()

	go Scheduler(streamList)

	mux := http.NewServeMux()

	mux.Handle("/streambox", newStreamboxHandler(streamList))

	log.Println("Listening...")
	http.ListenAndServe(":8080", mux)
}
