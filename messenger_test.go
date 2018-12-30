package messenger

import (
	"reflect"
	"sync"
	"testing"
)

func TestUnbuffered(t *testing.T) {
	clients := 5
	wgsub := &sync.WaitGroup{}
	wgsent := &sync.WaitGroup{}
	m := New(0, false)
	mu := &sync.Mutex{}
	results := make([]interface{}, 0)
	for i := 0; i < clients; i++ {
		wgsub.Add(1)
		wgsent.Add(1)
		go func(i int) {
			client, err := m.Sub()
			if err != nil {
				t.FailNow()
			}
			wgsub.Done()
			value := <-client
			mu.Lock()
			results = append(results, value)
			mu.Unlock()
			wgsent.Done()
		}(i)
	}
	wgsub.Wait()
	m.Broadcast("test")
	wgsent.Wait()
	if len(results) != clients {
		t.Errorf("expected %d messages, got %d", clients, len(results))
	}
	for _, v := range results {
		if v != "test" {
			t.Errorf("expected message 'test', got %s", v)
		}
	}
}

func TestBufferedReset(t *testing.T) {
	var buffer = 5
	clients := 5
	wgsub := &sync.WaitGroup{}
	wgsent := &sync.WaitGroup{}
	wgreset := &sync.WaitGroup{}
	wgreset.Add(1)
	m := New(uint(buffer), false)
	mu := &sync.Mutex{}
	results := make([]interface{}, 0)
	for i := 0; i < clients; i++ {
		wgsub.Add(1)
		wgsent.Add(1)
		go func(i int) {
			client, err := m.Sub()
			if err != nil {
				t.FailNow()
			}
			wgsub.Done()
			wgreset.Wait()
			for {
				value, ok := <-client
				if !ok {
					break
				}
				mu.Lock()
				results = append(results, value)
				mu.Unlock()
			}
			wgsent.Done()
		}(i)
	}
	wgsub.Wait()
	for i := 0; i < buffer; i++ {
		m.Broadcast("test")
	}
	m.Reset()
	wgreset.Done()
	wgsent.Wait()
	if len(results) != buffer*clients {
		t.Errorf("expected %d messages, got %d", buffer*clients, len(results))
	}
	for _, v := range results {
		if v != "test" {
			t.Errorf("expected message 'test', got %s", v)
		}
	}
}

func TestBufferedSkip(t *testing.T) {
	var buffer = 5
	clients := 5
	wgsub := &sync.WaitGroup{}
	wgsent := &sync.WaitGroup{}
	wgreset := &sync.WaitGroup{}
	wgreset.Add(1)
	m := New(uint(buffer), true)
	mu := &sync.Mutex{}
	results := make([]interface{}, 0)
	for i := 0; i < clients; i++ {
		wgsub.Add(1)
		wgsent.Add(1)
		go func(i int) {
			client, err := m.Sub()
			if err != nil {
				t.FailNow()
			}
			wgsub.Done()
			wgreset.Wait()
			for {
				value, ok := <-client
				if !ok {
					break
				}
				mu.Lock()
				results = append(results, value)
				mu.Unlock()
			}
			wgsent.Done()
		}(i)
	}
	wgsub.Wait()
	for i := 0; i < buffer; i++ {
		m.Broadcast("test")
	}
	for i := 0; i < buffer*5; i++ {
		m.Broadcast("wrong")
	}
	m.Reset()
	wgreset.Done()
	wgsent.Wait()
	if len(results) != buffer*clients {
		t.Errorf("expected %d messages, got %d", buffer*clients, len(results))
	}
	for _, v := range results {
		if v != "test" {
			t.Errorf("expected message 'test', got %s", v)
		}
	}
}

func TestBlockUnsub(t *testing.T) {
	clients := 5
	wgsub := &sync.WaitGroup{}
	wgunsub := &sync.WaitGroup{}
	wgunsub.Add(1)
	m := newNoMonitor(0, false)
	m.blocked = make(chan interface{})
	go m.monitor()
	for i := 0; i < clients; i++ {
		wgsub.Add(1)
		go func(i int) {
			client, err := m.Sub()
			if err != nil {
				t.FailNow()
			}
			wgsub.Done()
			wgunsub.Wait()
			m.Unsub(client)
		}(i)
	}
	wgsub.Wait()
	wgbroad := &sync.WaitGroup{}
	wgbroad.Add(1)
	go func() {
		for i := 0; i < 5; i++ {
			m.Broadcast("test")
		}
		wgbroad.Done()
	}()
	<-m.blocked
	wgunsub.Done()
	wgbroad.Wait()
}

func TestSkipUnsub(t *testing.T) {
	clients := 5
	wgsub := &sync.WaitGroup{}
	wgunsub := &sync.WaitGroup{}
	wgunsub.Add(1)
	buffer := 4
	m := newNoMonitor(uint(buffer), true)
	m.chanlens = make(chan []int)
	go m.monitor()
	for i := 0; i < clients; i++ {
		wgsub.Add(1)
		go func(i int) {
			client, err := m.Sub()
			if err != nil {
				t.FailNow()
			}
			wgsub.Done()
			wgunsub.Wait()
			m.Unsub(client)
		}(i)
	}
	wgsub.Wait()
	wgbroad := &sync.WaitGroup{}
	wgbroad.Add(1)
	go func() {
		for i := 0; i < 5; i++ {
			m.Broadcast("test")
		}
		wgbroad.Done()
	}()
	expected := []int{4, 4, 4, 4, 4}
	for {
		if reflect.DeepEqual(<-m.chanlens, expected) {
			break
		}
	}
	wgunsub.Done()
	wgbroad.Wait()
}

func TestKill(t *testing.T) {
	clients := 5
	wgsub := &sync.WaitGroup{}
	wgkill := &sync.WaitGroup{}
	m := New(5, true)
	for i := 0; i < clients; i++ {
		wgsub.Add(1)
		wgkill.Add(1)
		go func(i int) {
			client, err := m.Sub()
			if err != nil {
				t.FailNow()
			}
			wgsub.Done()
			for {
				_, ok := <-client
				if !ok {
					break
				}
			}
			wgkill.Done()
		}(i)
	}
	wgsub.Wait()
	m.Kill()
	wgkill.Wait()
}

func TestKillSub(t *testing.T) {
	m := New(0, false)
	m.Kill()
	for i := 0; i < 5; i++ {
		_, err := m.Sub()
		if err == nil {
			t.Error("Sub after kill didn't error")
		}
	}
}

func TestKillUnsub(t *testing.T) {
	m := New(0, false)
	client, err := m.Sub()
	if err != nil {
		t.Error("sub failed")
		t.FailNow()
	}
	m.Kill()
	m.Unsub(client)
}

func TestAfterKill(t *testing.T) {
	tt := []bool{true, false}
	for _, v := range tt {
		m := New(0, v)
		client, _ := m.Sub()
		m.Kill()
		m.Kill()
		m.Unsub(client)
		m.Sub()
		m.Broadcast("test")
		m.Reset()
	}
}
