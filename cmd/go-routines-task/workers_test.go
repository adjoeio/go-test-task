package main

import (
	"testing"
)

func TestWorkersProduceExpectedNumberOfMessages(t *testing.T) {
	const n = 2    // NoOfWorkers
	const itr = 10 // NoOfJobsPerWorker

	ch := make(chan string)
	Wg.Add(n)
	go CloseWorkersChannel(ch)

	for i := 1; i <= n; i++ {
		go LoadWorker(i, itr, ch)
	}

	// Receive messages from channel
	count := 0
	for range ch {
		count++
	}

	// Check if the number of messages is equal to the number of workers * number of jobs per worker
	if count != n*itr {
		t.Error("For", n, "workers and", itr, "iterations, expected", n*itr, "messages, got", count)
	}
}
