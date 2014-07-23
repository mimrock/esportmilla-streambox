package main

import (
	//"fmt"
	"github.com/mrshankly/go-twitch/twitch"
	"log"
	"net/http"
	"time"
	"strconv"
	"sync"
	"code.google.com/p/gcfg"
)


type StreamList struct {
	sync.RWMutex
	Streams []twitch.StreamS
}

type Config struct {
	Global struct {
		Refresh int
	}
}

type streamboxHandler struct {
	StreamList *StreamList
}

func newStreamboxHandler(streamList *StreamList) *streamboxHandler {
	log.Println("newStreamboxHandler")
	return &streamboxHandler{StreamList: streamList}
}

func (sb *streamboxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("ServeHTTP")
	w.Write([]byte("<html><body><ol>"))
	sb.StreamList.RLock()
	for _, s := range sb.StreamList.Streams {
		//fmt.Printf("%d - %s (%s) Status: %s Viewers: %d Url: %s Views: %d Name %s\n", i+1, s.Name, s.Game, s.Channel.Status, s.Viewers, s.Channel.Url, s.Channel.Views, s.Channel.Name)
		w.Write([]byte("<li>"))
		w.Write([]byte("Status: " + s.Channel.Status + " Game: " +s.Game + " (" + strconv.Itoa(s.Viewers) + ")"))
		w.Write([]byte("</li>"))
	}
	sb.StreamList.RUnlock()
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

func Scheduler(streamList *StreamList, cfg *Config) {
	log.Println("refresh time is", cfg.Global.Refresh, "seconds")
	refreshStreams := time.Tick(time.Second * time.Duration(cfg.Global.Refresh))
	for {
		select {
		case <-refreshStreams:
			st := getStreams()
			streamList.Lock()
			streamList.Streams = st
			streamList.Unlock()
		}
	}
}

func Init() (*StreamList, *Config) {
	var cfg Config
	err := gcfg.ReadFileInto(&cfg, "streambox.gcfg")
	if err != nil {
		log.Fatal(err)
	}
	streamList := new(StreamList)
	streamList.Streams = getStreams()
	return streamList, &cfg
}

func main() {
	log.Println("Starting up server...")
	streamList, cfg := Init()

	go Scheduler(streamList, cfg)

	mux := http.NewServeMux()

	mux.Handle("/streambox", newStreamboxHandler(streamList))

	log.Println("Listening...")
	http.ListenAndServe(":8080", mux)
}
