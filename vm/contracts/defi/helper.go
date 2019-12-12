package defi

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/common"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

var (
	LoanRateCardinalNum int32 = 1e6
	MinDayRate          int32 = 1 // 1/1,000,000
	MaxDayRate          int32 = LoanRateCardinalNum

	MinSubDays int32 = 1
	MaxSubDays int32 = 7

	commonTokenPow = big.NewInt(1e18)
	minShareAmount = new(big.Int).Mul(big.NewInt(10), commonTokenPow)
)

func CheckLoanParam(param *ParamNewLoan) error {
	if param.Token != ledger.ViteTokenId || param.DayRate <= MinDayRate || param.DayRate >= MaxDayRate ||
		param.ShareAmount.Cmp(minShareAmount) < 0 || param.Shares <= 0 ||
		param.SubscribeDays < MinSubDays || param.SubscribeDays > MaxSubDays || param.ExpireDays <= 0 {
		return InvalidInputParamErr
	} else {
		return nil
	}
}

func NewLoan(address types.Address, db vm_db.VmDb, param *ParamNewLoan, interest *big.Int) *Loan {
	loan := &Loan{}
	loan.Id = NewLoanSerialNo(db)
	loan.Address = address.Bytes()
	loan.Token = param.Token.Bytes()
	loan.ShareAmount = param.ShareAmount.Bytes()
	loan.Shares = param.Shares
	loan.Interest = interest.Bytes()
	loan.DayRate = param.DayRate
	loan.SubscribeDays = param.SubscribeDays
	loan.ExpireDays = param.ExpireDays
	loan.Status = Open
	loan.Created = GetDeFiTimestamp(db)
	return loan
}

func NewSubscription(address types.Address, db vm_db.VmDb, param *ParamSubscribe, loan *Loan) *Subscription {
	sub := &Subscription{}
	sub.LoanId = param.LoanId
	sub.Address = address.Bytes()
	sub.Token = loan.Token
	sub.Shares = param.Shares
	sub.ShareAmount = loan.ShareAmount
	sub.Status = Open
	sub.Created = GetDeFiTimestamp(db)
	return sub
}

func OnLoanSubscribed(db vm_db.VmDb, gs util.GlobalStatus, loan *Loan, shares int32) {
	loan.SubscribedShares = loan.SubscribedShares + shares
	loan.Updated = GetDeFiTimestamp(db)
	if loan.Shares == loan.SubscribedShares {
		loan.Status = Success
		loan.ExpireHeight = GetExpireHeight(gs, loan.ExpireDays)
		loan.StartTime = loan.Updated
		OnAccLoanSuccess(db, loan.Address, loan.Interest)
	}
	SaveLoan(db, loan)
	AddLoanUpdateEvent(db, loan)
	if loan.Status == Success {
		subscriptionsPrefix := append(subscriptionKeyPrefix, common.Uint64ToBytes(loan.Id)...)
		iterator, err := db.NewStorageIterator(subscriptionsPrefix)
		if err != nil {
			panic(err)
		}
		defer iterator.Release()
		leaveLoanInterest := new(big.Int).SetBytes(loan.Interest)
		for {
			if !iterator.Next() {
				if iterator.Error() != nil {
					panic(iterator.Error())
				}
				break
			}
			subVal := iterator.Value()
			if len(subVal) == 0 {
				continue
			}
			sub := &Subscription{}
			if err = sub.DeSerialize(subVal); err != nil {
				panic(err)
			}
			sub.Status = Success
			sub.Updated = loan.Updated
			interest := CalculateInterest(sub.Shares, new(big.Int).SetBytes(sub.ShareAmount), loan.DayRate, loan.ExpireDays)
			if leaveLoanInterest.Cmp(interest) < 0 {
				interest = leaveLoanInterest
				leaveLoanInterest = big.NewInt(0)
			} else {
				leaveLoanInterest.Sub(leaveLoanInterest, interest)
			}
			sub.Interest = interest.Bytes()
			OnAccSubscribeSuccess(db, sub.Address, interest, CalculateAmount(sub.Shares, sub.ShareAmount))
			AddSubscriptionUpdateEvent(db, sub)
			SaveSubscription(db, sub)
		}
	}
}

func CalculateInterest(shares int32, shareAmount *big.Int, dayRate, days int32) *big.Int {
	totalRate := dayRate * days
	totalAmount := CalculateAmount1(shares, shareAmount)
	return new(big.Int).SetBytes(common.CalculateAmountForRate(totalAmount.Bytes(), totalRate, LoanRateCardinalNum))
}

func CalculateAmount(shares int32, shareAmount []byte) *big.Int {
	return CalculateAmount1(shares, new(big.Int).SetBytes(shareAmount))
}

func CalculateAmount1(shares int32, shareAmount *big.Int) *big.Int {
	return new(big.Int).Mul(big.NewInt(int64(shares)), shareAmount)
}
