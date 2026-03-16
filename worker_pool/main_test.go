package workerpool

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestDifferentWorkerTaskRatios(t *testing.T) {
	tests := []struct {
		name    string
		workers int
		tasks   int
	}{
		{"больше задач чем воркеров", 10, 30},
		{"больше воркеров чем задач", 10, 3},
		{"один воркер", 1, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancelFunc := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancelFunc()

			wp := NewWorkerPool[int](tt.workers)
			resCh, err := wp.Start(ctx)
			if err != nil {
				t.Fatalf("Ошибка при старте пула: %s", err)
			}

			var count atomic.Int32
			done := make(chan struct{})
			go func() {
				for range resCh {
					count.Add(1)
				}
				close(done)
			}()

			for i := range tt.tasks {
				wp.AddTask(func() int { return i })
			}

			err = wp.Stop()
			if err != nil {
				t.Errorf("Ошибка при закрытии пула: %s", err)
			}

			<-done

			if count.Load() != int32(tt.tasks) {
				t.Errorf("Ожидалось %d результатов, получилось %d", tt.tasks, count.Load())
			}
		})
	}
}

func TestStopByContext(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	wp := NewWorkerPool[string](2)
	resCh, err := wp.Start(ctx)
	if err != nil {
		t.Fatalf("Ошибка при старте пула: %s", err)
	}

	cancelFunc()

	select {
	case _, ok := <-resCh:
		if ok {
			t.Error("Канал должен быть закрыт, но получили значение")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Таймаут: канал результатов не закрылся после отмены контекста")
	}

	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Errorf("Ожидался context.Canceled, получили %v", ctx.Err())
	}
}

func TestStopManual(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelFunc()

	wp := NewWorkerPool[int](3)
	resCh, err := wp.Start(ctx)
	if err != nil {
		t.Fatalf("Ошибка при старте пула: %s", err)
	}

	done := make(chan struct{})
	go func() {
		for range resCh {
		}
		close(done)
	}()

	wp.AddTask(func() int { return 42 })

	err = wp.Stop()
	if err != nil {
		t.Errorf("Ошибка при закрытии пула: %s", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Таймаут: канал результатов не закрылся после Stop")
	}
}

func TestDoubleStop(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelFunc()

	wp := NewWorkerPool[int](2)
	_, err := wp.Start(ctx)
	if err != nil {
		t.Fatalf("Ошибка при старте пула: %s", err)
	}

	err = wp.Stop()
	if err != nil {
		t.Errorf("Первый Stop вернул ошибку: %s", err)
	}

	err = wp.Stop()
	if err == nil {
		t.Error("Второй Stop должен вернуть ошибку")
	}
}

func TestAddTaskAfterStop(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelFunc()

	wp := NewWorkerPool[string](2)
	_, err := wp.Start(ctx)
	if err != nil {
		t.Fatalf("Ошибка при старте пула: %s", err)
	}

	wp.Stop()

	err = wp.AddTask(func() string { return "test" })
	if err == nil {
		t.Error("AddTask после Stop должен вернуть ошибку")
	}
}
