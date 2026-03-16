package workerpool

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type workerPool[T any] struct {
	workersNum int
	tasksCh    chan func() T
	resCh      chan T
	doneCh     chan bool
	isClosed   atomic.Bool
}

func NewWorkerPool[T any](workersNum int) *workerPool[T] {
	return &workerPool[T]{
		workersNum: workersNum,
		tasksCh:    make(chan func() T),
		resCh:      make(chan T),
		doneCh:     make(chan bool),
		isClosed:   atomic.Bool{},
	}
}

func (w *workerPool[T]) Start(ctx context.Context) (<-chan T, error) {
	if w.isClosed.Load() {
		return nil, ErrPoolIsAlreadyInitialized
	}

	wg := sync.WaitGroup{}

	for i := 0; i < w.workersNum; i++ {
		wg.Go(
			func() {
				for t := range w.tasksCh {
					res := t()
					w.resCh <- res
				}
			})
	}

	go func() {
		select {
		case <-w.doneCh:
			fmt.Println("Пул будет закрыт по запросу.")
		case <-ctx.Done():
			fmt.Println("Пул будет закрыт по контексту.")
			w.Stop()
		}
	}()

	go func() {
		wg.Wait()
		close(w.resCh)
		fmt.Println("Пул закрыт.")
	}()

	return w.resCh, nil
}

func (w *workerPool[T]) Stop() error {
	if w.isClosed.Swap(true) {
		return ErrPoolIsClosed
	}

	fmt.Println("Запрошено прерывание работы пула, ожидаем выполнения оставшихся задач.")

	close(w.doneCh)
	close(w.tasksCh)

	return nil
}

func (w *workerPool[T]) AddTask(task func() T) error {
	if w.isClosed.Load() {
		return ErrPoolIsClosed
	}

	select {
	case w.tasksCh <- task:
		fmt.Println("Задача отправлена в пул.")
	case <-w.doneCh:
		return ErrPoolIsClosed
	}

	return nil
}
