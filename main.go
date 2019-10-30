package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"time"

	datadog "github.com/DataDog/opencensus-go-exporter-datadog"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
)

func main() {
	exporter, err := datadog.NewExporter(datadog.Options{
		Namespace: "cprepro",
		OnError: func(err error) {
			panic(err)
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer exporter.Stop()
	view.RegisterExporter(exporter)
	view.Register(
		ochttp.ServerLatencyView,
	)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	h := &ochttp.Handler{
		Handler: http.HandlerFunc(handler),
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	mch := make(chan minmax)
	go setLatency(srv.URL, mch)
	go produceLatency(mch)

	runPinger(srv.URL, ch)

	fmt.Println("EXITING")
}

func runPinger(url string, done chan os.Signal) {
Outer:
	for {
		resp, err := http.Get(url)
		if err != nil {
			panic(err)
		}
		_, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		resp.Body.Close()
		select {
		case <-done:
			break Outer
		case <-shouldExit:
			break Outer
		default:
		}
	}
}

var durCh = make(chan time.Duration)
var shouldExit = make(chan struct{})

type simulator struct {
	dur time.Duration
	m   minmax
}

type minmax struct {
	min, max int
}

func setLatency(url string, ch chan minmax) {
	for _, sim := range [...]simulator{
		{
			dur: time.Minute * 4,
			m:   minmax{200, 400},
		},
		{
			dur: time.Minute * 6,
			m:   minmax{1000, 2000},
		},
		{
			dur: time.Minute * 4,
			m:   minmax{200, 400},
		},
	} {
		ch <- sim.m
		time.Sleep(sim.dur)
	}
	close(shouldExit)
}

func produceLatency(ch chan minmax) {
	m := <-ch
	for {
		select {
		case m = <-ch:
		case <-shouldExit:
			return
		default:
		}
		dur := time.Millisecond * time.Duration(m.min+rand.Intn(m.max-m.min))
		fmt.Println("MIN", m.min, "MAX", m.max, "DUR", dur)
		durCh <- dur
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(<-durCh)
	w.WriteHeader(200)
	w.Write([]byte("cool\n"))
}
