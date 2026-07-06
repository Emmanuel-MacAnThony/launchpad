package result

type Result[T any] struct {
	Value T
	Err   error
}

func Ok[T any](v T) Result[T] {
	return Result[T]{Value: v}
}

func Fail[T any](err error) Result[T] {
	return Result[T]{Err: err}
}

func (r Result[T]) IsOk() bool {
	return r.Err == nil
}
