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
	wgexit := &sync.WaitGroup{}
	m := newNoMonitor(0, false)
	m.blocked = make(chan interface{})
	go m.monitor()
	for i := 0; i < clients; i++ {
		wgsub.Add(1)
		wgexit.Add(1)
		go func(i int) {
			client, err := m.Sub()
			if err != nil {
				t.FailNow()
			}
			wgsub.Done()
			wgunsub.Wait()
			m.Unsub(client)
			wgexit.Done()
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
	wgexit.Wait()
}

func TestSkipUnsub(t *testing.T) {
	clients := 5
	wgsub := &sync.WaitGroup{}
	wgunsub := &sync.WaitGroup{}
	wgunsub.Add(1)
	wgexit := &sync.WaitGroup{}
	buffer := 4
	m := newNoMonitor(uint(buffer), true)
	m.chanlens = make(chan []int)
	go m.monitor()
	for i := 0; i < clients; i++ {
		wgsub.Add(1)
		wgexit.Add(1)
		go func(i int) {
			client, err := m.Sub()
			if err != nil {
				t.FailNow()
			}
			wgsub.Done()
			wgunsub.Wait()
			m.Unsub(client)
			wgexit.Done()
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
	wgbroad.Wait()
	expected := []int{4, 4, 4, 4, 4}
	if !reflect.DeepEqual(<-m.chanlens, expected) {
		t.Errorf("chan lenghts do not match")
	}
	wgunsub.Done()
	wgexit.Wait()
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

func TestAfterKill(t *testing.T) {
	tt := []struct {
		buf  uint
		drop bool
	}{
		{0, true},
		{0, false},
		{1, true},
		{1, false},
	}
	for _, tc := range tt {
		m := New(tc.buf, tc.drop)
		client, _ := m.Sub()
		m.Kill()
		// These should not block
		m.Kill()
		m.Unsub(client)
		m.Sub()
		m.Broadcast("test")
		m.Reset()
	}
}

func TestLen(t *testing.T) {
	m := New(0, false)
	m.Sub()
	m.Sub()
	c, _ := m.Sub()
	m.Unsub(c)
	m.Sub()
	if m.Len() != 3 {
		t.Errorf("expected len to be 2 got %d", m.Len())
	}
}
