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

type streamboxHandler struct {
	streams *[]twitch.StreamS
	mutex *sync.RWMutex
}

func newStreamboxHandler(streams *[]twitch.StreamS, mutex *sync.RWMutex) *streamboxHandler {
	log.Println("newStreamboxHandler")
	return &streamboxHandler{streams: streams, mutex: mutex}
}

func (sb *streamboxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("ServeHTTP")
	w.Write([]byte("<html><body><ol>"))
	sb.mutex.RLock()
	for _, s := range *sb.streams {
		//fmt.Printf("%d - %s (%s) Status: %s Viewers: %d Url: %s Views: %d Name %s\n", i+1, s.Name, s.Game, s.Channel.Status, s.Viewers, s.Channel.Url, s.Channel.Views, s.Channel.Name)
		w.Write([]byte("<li>"))
		w.Write([]byte("Status: " + s.Channel.Status + " Game: " +s.Game + " (" + strconv.Itoa(s.Viewers) + ")"))
		w.Write([]byte("</li>"))
	}
	sb.mutex.RUnlock()
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

func Scheduler(streams *[]twitch.StreamS, mutex *sync.RWMutex) {
	refreshStreams := time.Tick(5000 * time.Millisecond)
	for {
		select {
		case <-refreshStreams:
			st := getStreams()
			mutex.Lock()
			*streams = st
			mutex.Unlock()
		}
	}
}

func main() {
	log.Println("Starting up server...")
	mutex := &sync.RWMutex{}
	streams := getStreams()

	go Scheduler(&streams, mutex)

	mux := http.NewServeMux()

	mux.Handle("/streambox", newStreamboxHandler(&streams, mutex))

	log.Println("Listening...")
	http.ListenAndServe(":8080", mux)
}
