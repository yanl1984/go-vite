package defi

import (
	"errors"
)

var (
	InvalidInputParamErr = errors.New("invalid input param")

	ExceedFundAvailableErr       = errors.New("exceed fund available")
)