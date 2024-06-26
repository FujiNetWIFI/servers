package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestConcurrentGameSlice_Init_Zero_Len(t *testing.T) {

	t.Run("ConcurrentGameSlice init must have 0 elements", func(t *testing.T) {

		cgs := NewConcurrentGameSlice()

		if len := cgs.Len(); len != 0 {
			t.Errorf("ConcurrentGameSlice.Len() = %v, want %v", len, 0)
		}
	})
}

func TestConcurrentGameSlice_GameAtPos_Failures_0Elem(t *testing.T) {

	t.Run("ConcurrentGameSlice GameAtPos check (0 elements) failures", func(t *testing.T) {

		cgs := NewConcurrentGameSlice()

		game, ok := cgs.GetAtPos(-1)

		if ok {
			t.Errorf("ConcurrentGameSlice.GetAtPos[-1] must fail, but is returning: %+v", game)
		}

		game, ok = cgs.GetAtPos(0)

		if ok {
			t.Errorf("ConcurrentGameSlice.GetAtPos[0] must fail, but is returning: %+v", game)
		}

		game, ok = cgs.GetAtPos(1)

		if ok {
			t.Errorf("ConcurrentGameSlice.GetAtPos[1] must fail, but is returning: %+v", game)
		}

	})
}

func TestConcurrentGameSlice_Variable_Len(t *testing.T) {

	t.Run("ConcurrentGameSlice Len() check (255 elem)", func(t *testing.T) {

		cgs := NewConcurrentGameSlice()

		for i := 0; i < 255; i++ {
			name := fmt.Sprintf("ConcurrentGameSlice #%d", i)
			url := "http://example.com/"
			cgs.Append(makeGame(name, url))

			len := cgs.Len()

			if len != i+1 {
				t.Errorf("ConcurrentGameSlice.Len(%d) = %v, want %v", i, len, i)
			}

		}
	})
}

func TestConcurrentGameSlice_GameAtPos_Failures_255Elem(t *testing.T) {

	t.Run("ConcurrentGameSlice GameAtPos check (255 elements) failures", func(t *testing.T) {

		cgs := NewConcurrentGameSlice()

		for i := 0; i < 255; i++ {
			name := fmt.Sprintf("ConcurrentGameSlice #%d", i)
			url := "http://example.com/"
			cgs.Append(makeGame(name, url))
		}

		game, ok := cgs.GetAtPos(-1)

		if ok {
			t.Errorf("ConcurrentGameSlice.GetAtPos[-1] must fail, but is returning: %+v", game)
		}

		game, ok = cgs.GetAtPos(255)

		if ok {
			t.Errorf("ConcurrentGameSlice.GetAtPos[255] must fail, but is returning: %+v", game)
		}

	})
}

func TestConcurrentGameSlice_GameAtPos_HappyFlow_255Elem(t *testing.T) {

	t.Run("ConcurrentGameSlice GameAtPos check (255 elements) happy flow", func(t *testing.T) {

		cgs := NewConcurrentGameSlice()

		for i := 0; i < 255; i++ {
			name := fmt.Sprintf("ConcurrentGameSlice #%d", i)
			url := "http://example.com"
			cgs.Append(makeGame(name, url))
		}

		for pos := 0; pos < 255; pos++ {

			game, ok := cgs.GetAtPos(pos)

			if !ok {
				t.Errorf("ConcurrentGameSlice.GetAtPos[%d] must not fail", pos)
			}

			ExpectedName := fmt.Sprintf("ConcurrentGameSlice #%d", pos)
			ExpectedServerURL := fmt.Sprintf("http://example.com/games/%d/", pos)

			if game.Name != ExpectedName ||
				game.ServerUrl != ExpectedServerURL {
				t.Errorf("ConcurrentGameSlice.GetAtPos[%d] = %+v, want %+v", pos, game, makeGame(ExpectedName, "http://example.com/"))
			}
		}
	})
}

func TestConcurrentGameSlice_AllAsMap_255Elem(t *testing.T) {

	t.Run("ConcurrentGameSlice AllAsMap() check (255 elements)", func(t *testing.T) {

		cgs := NewConcurrentGameSlice()

		for pos := 0; pos < 255; pos++ {
			name := fmt.Sprintf("ConcurrentGameSlice #%d", pos)
			url := "http://example.com"
			cgs.Append(makeGame(name, url))

			DataAsMap := cgs.AllAsMap()

			if len(DataAsMap) != pos+1 {
				t.Errorf("ConcurrentGameSlice.AllAsMap() has len = %d, want %d", len(DataAsMap), pos)
			}
		}
	})
}

func TestConcurrentGameSlice_AllAsMap_0Elem(t *testing.T) {

	t.Run("ConcurrentGameSlice AllAsMap() check (0 elements)", func(t *testing.T) {

		cgs := NewConcurrentGameSlice()

		DataAsMap := cgs.AllAsMap()

		if len(DataAsMap) != 0 {
			t.Errorf("ConcurrentGameSlice.AllAsMap() has len = %d, want %d", len(DataAsMap), 0)
		}

	})
}

func TestConcurrentGameSlice_MultithreadAppend(t *testing.T) {
	t.Run("ConcurrentGameSlice Append (multithread with [1, 13] goroutines and [1, 256] elements each)", func(t *testing.T) {

		ngoroutines := rand.Intn(13 + 1)
		nelements := rand.Intn(255 + 1)

		cgs := NewConcurrentGameSlice()

		wg := new(sync.WaitGroup)

		for i := 0; i < ngoroutines; i++ {
			wg.Add(1)
			go appender(wg, &cgs, nelements)
		}

		wg.Wait()

		if cgs.Len() != ngoroutines*nelements {
			t.Errorf("ConcurrentGameSlice.Append() (multithread) has len = %d, want %d (%d goroutines, %d elem each)", cgs.Len(), ngoroutines*nelements, ngoroutines, nelements)
		}

	})
}

func appender(wg *sync.WaitGroup, cgs *ConcurrentGameSlice, NumInserts int) {

	for i := 0; i < NumInserts; i++ {

		Name := fmt.Sprintf("ConcurrentGameSlice #%d/%d", goid(), i)
		ServerURL := fmt.Sprintf("http://example.com/games/%d/", i)
		cgs.Append(makeGame(Name, ServerURL))
	}

	wg.Done()

}

func goid() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}
