package workerpool

import "errors"

var (
	ErrPoolIsClosed             = errors.New("Пул уже закрыт.")
	ErrPoolIsAlreadyInitialized = errors.New("Попытка запустить уже запущенный пул.")
)
