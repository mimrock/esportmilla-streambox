package main

import (
	"fmt"
	"github.com/mrshankly/go-twitch/twitch"
	"log"
	"net/http"
	"time"
	"strconv"
	"sync"
	"code.google.com/p/gcfg"
	"bufio"
	"os"
)


type StreamList struct {
	sync.RWMutex
	Streams []twitch.StreamS
}

type Config struct {
	Server struct {
		Port int
		TwitchRefresh int
	}
	Logging struct {
		ErrorLog string
		EventLog string
		AccessLog string
	}
}

type LogChans struct {
	Error chan string
	Event chan string
	Access chan string
}

var Loggers  = &LogChans {
	make(chan string),
	make(chan string),
	make(chan string),
}

type streamboxHandler struct {
	StreamList *StreamList
}

func newStreamboxHandler(streamList *StreamList) *streamboxHandler {
	log.Println("newStreamboxHandler")
	return &streamboxHandler{StreamList: streamList}
}

func (sb *streamboxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accessMsg := fmt.Sprintf("%v %v from %v Headers: %+v", r.Method, r.RequestURI, r.RemoteAddr, r.Header)
	LogAccess(accessMsg)
	w.Write([]byte("<html><body><ol>"))
	sb.StreamList.RLock()
	for _, s := range sb.StreamList.Streams {
		//fmt.Printf("%d - %s (%s) Status: %s Viewers: %d Url: %s Views: %d Name %s\n", i+1, s.Name, s.Game, s.Channel.Status, s.Viewers, s.Channel.Url, s.Channel.Views, s.Channel.Name)
		w.Write([]byte("<li>Status: " + s.Channel.Status + " Game: " +s.Game + " (" + strconv.Itoa(s.Viewers) + ")</li>"))
	}
	sb.StreamList.RUnlock()
 	w.Write([]byte("</ol></body></html>"))
}

func LogError(msg string) {
	Loggers.Error <- msg
}

func LogEvent(msg string) {
	Loggers.Event <- msg
}

func LogAccess(msg string) {
	Loggers.Access <- msg
}

func flushWriterWorker(writer *bufio.Writer, flushInterval int) {
	flushLog := time.Tick(time.Second * time.Duration(flushInterval))
	for {
		select {
		case <-flushLog:
			writer.Flush()
		}
	}
}

// @todo Graceful stopping by closing threads (last parts of logs are not even written to the disk)
// @todo Add timestamps on log entries
func logWorker(logFile string, flushInterval int, input chan string) {
    logWriter, err := os.Create(logFile)
    if err != nil { panic(err) }

    defer func() {
        if err = logWriter.Close(); err != nil {
            panic(err)
        }
    }()

    bufLogWriter := bufio.NewWriterSize(logWriter, 65535)
    logger := log.New(bufLogWriter, "logger: ", log.Ltime & log.Lshortfile)

    for {
    	select {
    	case msg := <-input:
    		logger.Println(msg)
    		if (flushInterval == 0) {
    			bufLogWriter.Flush()
    		}
    	}
    }

    if (flushInterval > 0) {
    	go flushWriterWorker(bufLogWriter, flushInterval)
	}
}

func getStreams() []twitch.StreamS {
	LogEvent("Getting streams from Twitch.")
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
	log.Println("refresh time is", cfg.Server.TwitchRefresh, "seconds")
	refreshStreams := time.Tick(time.Second * time.Duration(cfg.Server.TwitchRefresh))
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

	go logWorker(cfg.Logging.ErrorLog, 0, Loggers.Error)
	go logWorker(cfg.Logging.EventLog, 0, Loggers.Event)
	go logWorker(cfg.Logging.AccessLog, 30, Loggers.Access)

	LogEvent("Starting up server.")


	streamList := new(StreamList)
	streamList.Streams = getStreams()
	return streamList, &cfg
}

func main() {
	log.Println("Starting up server...")
	streamList, cfg := Init()

	go Scheduler(streamList, cfg)

	defer func() {
        LogEvent("Shutdown.")
    }()

	mux := http.NewServeMux()

	mux.Handle("/streambox", newStreamboxHandler(streamList))

	log.Println("Listening...")
	http.ListenAndServe(":" + strconv.Itoa(cfg.Server.Port), mux)
}
