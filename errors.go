package source

import (
	"errors"
	"fmt"
)

var (
	// ErrNilOption is returned by NewClient if an option is nil.
	ErrNilOption = errors.New("source: nil option")

	// ErrNonASCII is returned if a command with non-ASCII characters is attempted.
	ErrNonASCII = errors.New("source: non-ascii body")

	// ErrAuthFailure is returned if the client failed to authenticate.
	ErrAuthFailure = errors.New("source: authentication failure")
)

// ErrMalformedResponse is returned if the response from the server is malformed.
type ErrMalformedResponse string

func (e ErrMalformedResponse) Error() string {
	return fmt.Sprintf("source: malformed response %v", string(e))
}
