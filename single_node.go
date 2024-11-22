package main

import (
	"sync"
)

func SingleNode(toCall string) []byte {
	responseChannel := make(chan *Response, *totalCalls*2)

	benchTime := NewTimer()
	benchTime.Reset()
	//TODO check ulimit
	wg := &sync.WaitGroup{}

	for i := 0; i < *numConnections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			StartClient(
				toCall,
				*headers,
				*requestBody,
				*proxy,
				*method,
				*disableKeepAlives,
				responseChannel,
				*totalCalls,
			)
		}()
	}
	wg.Wait()

	result := CalcStats(
		responseChannel,
		benchTime.Duration(),
	)
	return result
}
