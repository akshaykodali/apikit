package apikit

import (
	"sync"
	"time"
)

func CreateLogger(wg *sync.WaitGroup, httpLogs <-chan Log, flusher Flusher) {
	wg.Add(1)

	go func() {
		defer wg.Done()

		buffer := make([]Log, 0, 1000)

		var interval time.Duration
		if flusher.Interval() == 0 {
			interval = 5 * time.Second
		} else {
			interval = flusher.Interval()
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case log, ok := <-httpLogs:
				if !ok {
					if len(buffer) > 0 {
						flusher.Flush(buffer)
					}

					return
				}

				buffer = append(buffer, log)
			case <-ticker.C:
				if len(buffer) > 0 {
					flusher.Flush(buffer)
					buffer = make([]Log, 0, 1000)
				}
			}
		}
	}()
}
