package apikit

import (
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateLogger(wg *sync.WaitGroup, db *pgxpool.Pool, httpLogs <-chan Log, flush func(*pgxpool.Pool, *[]Log)) {
	wg.Add(1)

	go func() {
		defer wg.Done()

		buffer := make([]Log, 0, 1000)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case log, ok := <-httpLogs:
				if !ok {
					if len(buffer) > 0 {
						flush(db, &buffer)
					}

					return
				}

				buffer = append(buffer, log)
			case <-ticker.C:
				if len(buffer) > 0 {
					flush(db, &buffer)
				}
			}
		}
	}()
}
