package dashboard

import (
	"sync"
	"testing"
)

func TestEmbeddedAssetsInitializeConcurrently(t *testing.T) {
	assetsFS = nil
	assetsOnce = sync.Once{}

	const workers = 32
	errors := make(chan error, workers)
	var wait sync.WaitGroup
	wait.Add(workers)
	for range workers {
		go func() {
			defer wait.Done()
			file, err := getAssetsFS().Open("js/main.js")
			if err == nil {
				err = file.Close()
			}
			errors <- err
		}()
	}
	wait.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Fatalf("opening an embedded asset concurrently: %v", err)
		}
	}
}
