package exchange

import "fmt"

type NoTopicError struct {
	File string
}

func (e *NoTopicError) Error() string {
	return fmt.Sprintf("file %s has no topic", e.File)
}

type EmptyMessageError struct {
	File string
}

func (e *EmptyMessageError) Error() string {
	return fmt.Sprintf("file %s has an empty message", e.File)
}
