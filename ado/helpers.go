package ado

func Pointer[T any](val T) *T {
	return &val
}