package main

import (
	log "github.com/sirupsen/logrus"
)

func main() {
	const NoOfWorkers = 2
	const NoOfJobsPerWorker = 10

	ch := make(chan string)
	Wg.Add(NoOfWorkers)

	// Close the channel after all the workers finish their job
	go CloseWorkersChannel(ch)

	for i := 1; i <= NoOfWorkers; i++ {
		// Start the worker
		go LoadWorker(i, NoOfJobsPerWorker, ch)
	}

	// Receive messages from channel
	for message := range ch {
		log.Info(message)
	}
}
