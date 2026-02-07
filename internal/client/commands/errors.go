package commands

import (
	"errors"
	"fmt"
)

var (
	ErrCLIError = errors.New("CLI error")

	ErrMissingArgument = fmt.Errorf("%w: missing argument", ErrCLIError)
)
