package messenger_test

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ugjka/messenger"
)

func ExampleNew() {
	log.SetFlags(log.Lmicroseconds)
	log.SetPrefix("time ")
	m := messenger.New(0, false)
	wg := &sync.WaitGroup{}
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(i int, m *messenger.Messenger) {
			defer wg.Done()
			time.Sleep(time.Millisecond * time.Duration(i*20))
			client, err := m.Sub()
			if err != nil {
				log.Printf("Client %d: %v\n", i, err)
				return
			}
			log.Printf("Client %d subscribed\n", i)
			timeout := time.After(time.Millisecond * time.Duration(i*100))
			for {
				select {
				case msg := <-client:
					log.Printf("Client %d got message: %s\n", i, msg)
				case <-timeout:
					m.Unsub(client)
					log.Printf("Client %d unsubscribed\n", i)
					return
				}
			}
		}(i, m)
	}
	for i := 0; i < 10; i++ {
		time.Sleep(time.Millisecond * 50)
		m.Broadcast(fmt.Sprintf("nr.%d", i))
	}
	wg.Wait()
}
