package errutil

import (
	"errors"
	"net"
	"syscall"
)

func IsConnRefused(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Op == "dial" && errors.Is(err, syscall.ECONNREFUSED) {
			return true
		}
	}
	return false
}
