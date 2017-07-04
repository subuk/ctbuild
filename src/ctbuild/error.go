package main

import (
	"fmt"
)

type Error struct {
	Orig    error
	Message string
}

func (err Error) Error() string {
	if err.Orig != nil {
		return fmt.Sprintf("%s: %s", err.Message, err.Orig)
	} else {
		return fmt.Sprintf("%s", err.Message)
	}
}
