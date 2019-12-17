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
	AvailableHeightNotValidForInvestErr = errors.New("loan expire height not valid for invest")
	InvalidSourceAddressErr             = errors.New("invalid source address")
	InvestNotExistsErr                  = errors.New("invest not exists")
	InvestNotExpiredErr                 = errors.New("invest not expired")
	InvalidQuotaInvestErr               = errors.New("invalid quota invest")
	SBPRegistrationNotExistsErr         = errors.New("sbp registration not exists")

	ExceedFundAvailableErr              = errors.New("exceed fund available")
)
