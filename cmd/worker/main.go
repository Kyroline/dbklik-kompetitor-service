// Command worker is the background-job entrypoint. Wire module
// schedulers/queues here as they're added under infrastructure/scheduler.
package main

import "log"

func main() {
	log.Println("worker: no jobs registered yet")
}
