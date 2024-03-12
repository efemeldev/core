package main

import (
	"fmt"
	"syscall"
)

func null[T any]() T {
	var zero T
	return zero
}

func exit(message error) {
	fmt.Println(message)
	syscall.Exit(1)
}

func handleError[T interface{}](data T, err error) T {
	if err != nil {
		exit(err)
	}

	return data
}
