package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// Wg is a wait group used for consumers when workers finished their work
var Wg sync.WaitGroup
var client = &http.Client{}

func LoadWorker(id int, itr int, ch chan<- string) {
	defer Wg.Done()

	for i := 0; i < itr; i++ {
		// simulate a random time to wait for
		randomWaitTime := time.Duration(rand.Int63n(int64(time.Second)))
		time.Sleep(randomWaitTime)
		// call the api
		req, err := http.NewRequest("GET", "http://localhost:8000/v1/", nil)
		if err != nil {
			panic(err)
		}
		req.Header.Set("test", "key")
		resp, err := client.Do(req)
		defer resp.Body.Close()
		if err != nil {
			panic(err)
		}

		// send the message to the channel
		ch <- fmt.Sprintf("Id %d: is on %d iteration resp %d", id, i, resp.Body)
	}
}

func CloseWorkersChannel(ch chan string) {
	Wg.Wait()
	// close the channel
	close(ch)
}
