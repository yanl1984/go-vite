package defi

import (
	"errors"
)

var (
	InvalidInputParamErr = errors.New("invalid input param")

	LoanNotExistsErr                    = errors.New("loan not exists")
	LoanSubscribeFailed                 = errors.New("loan subscribe failed")
	InvalidLoanStatusForCancelErr       = errors.New("invalid loan status for cancel")
	OnlyOwnerAllowErr                   = errors.New("only owner allow")
	InvestAmountNotValidErr             = errors.New("invest amount not valid")
	AvailableHeightNotValidForInvestErr = errors.New("loan expire height not valid for invest")

	ExceedFundAvailableErr = errors.New("exceed fund available")
)
