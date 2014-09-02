package main

import (
	"bufio"
	"code.google.com/p/gcfg"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mrshankly/go-twitch/twitch"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type StreamList struct {
	sync.RWMutex
	Streams []twitch.StreamS
}

type Config struct {
	Server struct {
		Port          int
		TwitchRefresh int
		TwitchRetry   int
	}
	Logging struct {
		ErrorLog  string
		EventLog  string
		AccessLog string
	}
	DataSources struct {
		MainDatabase string
	}
}

var cfg = &Config{}

type LogChans struct {
	Error  chan string
	Event  chan string
	Access chan string
}

var Loggers = &LogChans{
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
		w.Write([]byte("<li>Status: " + s.Channel.Status + " Game: " + s.Game + " (" + strconv.Itoa(s.Viewers) + ")</li>"))
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

func flushWriterWorker(writer *bufio.Writer, flushInterval int, mutex *sync.Mutex) {
	flushLog := time.Tick(time.Second * time.Duration(flushInterval))
	for {
		select {
		case <-flushLog:
			mutex.Lock()
			writer.Flush()
			mutex.Unlock()
		}
	}
}

func compressWorker(bufLogWriter *bufio.Writer, logWriter *os.File, logfile string, mutex *sync.Mutex) {
	sizeCheck := time.Tick(time.Second * 5)
	for {
		select {
		case <-sizeCheck:
			stats, err := logWriter.Stat()
			if err != nil {
				panic(err)
			}
			log.Println("File size:", stats.Size())
			if stats.Size() > 240 {
				mutex.Lock()
				if err = logWriter.Close(); err != nil {
					panic(err)
				}
				os.Rename(logfile, logfile+".bkp")
				lw, err := os.OpenFile(logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
				if err != nil {
					panic(err)
				}
				*logWriter = *lw
				bf := bufio.NewWriterSize(logWriter, 65535)
				if err != nil {
					panic(err)
				}
				*bufLogWriter = *bf
				mutex.Unlock()
			}
		}
	}

}

func logWorker(logFile string, flushInterval int, input chan string, safeQuit *sync.WaitGroup) {
	logMutex := &sync.Mutex{}
	logWriter, err := os.OpenFile(logFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
	if err != nil {
		panic(err)
	}

	defer func() {
		logMutex.Lock()
		if err = logWriter.Close(); err != nil {
			panic(err)
		}
		log.Println("Closed", logFile)
		logMutex.Unlock()
		safeQuit.Done()
	}()

	bufLogWriter := bufio.NewWriterSize(logWriter, 65535)
	logger := log.New(bufLogWriter, "", log.Ldate|log.Ltime)

	if flushInterval > 0 {
		go flushWriterWorker(bufLogWriter, flushInterval, logMutex)
	}

	go compressWorker(bufLogWriter, logWriter, logFile, logMutex)

	for msg := range input {
		logMutex.Lock()
		logger.Println(msg)
		if flushInterval == 0 {
			bufLogWriter.Flush()
		}
		logMutex.Unlock()
	}
}

func getChannelLists(activeChannelIds []string, limit int) []string {
	var start, end int
	var channelLists []string
	for i := 0; i <= ((len(activeChannelIds) - 1) / limit); i++ {
		start = i * limit

		if (i+1)*limit < len(activeChannelIds) {
			end = (i + 1) * limit
		} else {
			end = len(activeChannelIds)
		}
		filterStringsSlice := activeChannelIds[start:end]
		channelLists = append(channelLists, strings.Join(filterStringsSlice, ","))
	}
	return channelLists
}

// @todo retry for a fixed amount of times if the download fails.
func downloadStreams(output chan []twitch.StreamS, done chan bool, channelList string) {
	defer func() { done <- true }()

	var err error
	var streams *twitch.StreamsS

	do := true

	for i := 0; do == true; i++ {
		do = false
		client := twitch.NewClient(&http.Client{})
		opt := &twitch.ListOptions{
			Limit:   100,
			Offset:  0,
			Channel: channelList,
		}

		streams, err = client.Streams.List(opt)
		if err != nil {
			LogError(err.Error())
			if i < cfg.Server.TwitchRetry {
				do = true
			} else {
				LogError("Failed getting some data from Twitch. Internal streamlist can be incomplete.")
			}
		}
	}
	output <- streams.Streams
}

func getStreams(activeChannelIds []string) []twitch.StreamS {
	channelLists := getChannelLists(activeChannelIds, 100)

	LogEvent("Getting streams from Twitch.")

	if len(channelLists) < 1 {
		var emptyStreamList []twitch.StreamS
		return emptyStreamList
	}

	output := make(chan []twitch.StreamS)
	done := make(chan bool)
	for _, channelList := range channelLists {
		go downloadStreams(output, done, channelList)
	}

	var activeStreams, downloadedStreams []twitch.StreamS
	for i := 0; i < len(channelLists); {
		select {
		case downloadedStreams = <-output:
			activeStreams = append(activeStreams, downloadedStreams...)
		case <-done:
			i++
		}
	}

	return activeStreams
}

func Scheduler(streamList *StreamList) {
	log.Println("refresh time is", cfg.Server.TwitchRefresh, "seconds")
	refreshStreams := time.Tick(time.Second * time.Duration(cfg.Server.TwitchRefresh))
	for {
		select {
		case <-refreshStreams:
			st := getStreams(getActiveChannels(cfg.DataSources.MainDatabase))
			streamList.Lock()
			streamList.Streams = st
			streamList.Unlock()
		}
	}
}

func getActiveChannels(dataSourceName string) []string {
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		panic(err)
	}
	rows, err := db.Query("SELECT channel_id FROM streambox_channels WHERE enabled = 1")
	if err != nil {
		log.Fatal(err)
	}
	var enabledChannels []string
	defer rows.Close()
	for rows.Next() {
		var channel_id string
		if err := rows.Scan(&channel_id); err != nil {
			panic(err)
		}
		enabledChannels = append(enabledChannels, channel_id)
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}

	return enabledChannels
}

func Init(safeQuit *sync.WaitGroup) *StreamList {
	err := gcfg.ReadFileInto(cfg, "streambox.gcfg")
	if err != nil {
		log.Fatal(err)
	}
	safeQuit.Add(3)
	go logWorker(cfg.Logging.ErrorLog, 0, Loggers.Error, safeQuit)
	go logWorker(cfg.Logging.EventLog, 0, Loggers.Event, safeQuit)
	go logWorker(cfg.Logging.AccessLog, 30, Loggers.Access, safeQuit)

	LogEvent("Starting up server.")

	streamList := new(StreamList)
	streamList.Streams = getStreams(getActiveChannels(cfg.DataSources.MainDatabase))
	return streamList
}

func main() {
	log.Println("Starting up server...")
	var safeQuit sync.WaitGroup
	streamList := Init(&safeQuit)

	go Scheduler(streamList)

	go func() {
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		log.Println(<-ch)
		close(Loggers.Event)
		close(Loggers.Access)
		close(Loggers.Error)
		safeQuit.Wait()
		log.Println("Normal Shutdown")
		os.Exit(0)
	}()

	mux := http.NewServeMux()

	mux.Handle("/streambox", newStreamboxHandler(streamList))

	log.Println("Listening...")
	http.ListenAndServe(":"+strconv.Itoa(cfg.Server.Port), mux)
}
