package defi

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/common"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
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
	loan.Status = LoanOpen
	loan.Created = GetDeFiTimestamp(db)
	return loan
}

func OnLoanInvest(db vm_db.VmDb, loan *Loan, amount *big.Int) {
	loan.Invested = common.AddBigInt(loan.Invested, amount.Bytes())
	SaveLoan(db, loan)
}

func OnLoanCancelInvest(db vm_db.VmDb, loan *Loan, amount []byte) {
	if common.CmpForBigInt(loan.Invested, amount) < 0 {
		panic(ExceedFundAvailableErr)
	} else {
		loan.Invested = common.SubBigIntAbs(loan.Invested, amount)
		SaveLoan(db, loan)
	}
}

func DoRefundLoan(db vm_db.VmDb, loan *Loan) {
	address, _ := types.BytesToAddress(loan.Address)
	switch loan.Status {
	case LoanFailed:
		OnAccLoanFailed(db, address, loan.Interest)
		AddBaseAccountEvent(db, loan.Address, BaseLoanInterestRelease, 0, loan.Id, loan.Interest)
	case LoanExpired:
		amount := CalculateAmount(loan.Shares, loan.ShareAmount)
		OnAccLoanExpired(db, address, amount)
		AddLoanAccountEvent(db, loan.Address, LoanExpiredRefund, 0, loan.Id, amount.Bytes())
	}
	if loan.SubscribedShares > 0 {
		refundLoanSubscriptions(db, loan)
	}
	DeleteLoan(db, loan)
	AddLoanUpdateEvent(db, loan)
}

func DoCancelExpiredLoanInvests(db vm_db.VmDb, loan *Loan) (blocks []*ledger.AccountBlock, err error) {
	var iterator interfaces.StorageIterator
	iterator, err = db.NewStorageIterator(append(investToLoanIndexKeyPrefix, common.Uint64ToBytes(loan.Id)...))
	if err != nil {
		panic(err)
	}
	defer iterator.Release()
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				panic(iterator.Error())
			}
			break
		}
		data := iterator.Value()
		if len(data) != 8 {
			panic(InnerError)
		}
		investId := common.BytesToUint64(data)
		var blks []*ledger.AccountBlock
		if invest, ok := GetInvest(db, investId); ok && invest.Status == InvestSuccess {
			switch invest.BizType {
			case InvestForMining, InvestForSVIP:
				blks, err = DoCancelDexInvest(investId)
			case InvestForQuota:
				blks, err = DoCancelQuotaInvest(invest.InvestHash)
			case InvestForSBP:
				blks, err = DoRevokeSBP(db, invest.InvestHash)
			}
		}
		if err != nil {
			return
		}
		blocks = append(blocks, blks...)
	}
	return
}

func refundLoanSubscriptions(db vm_db.VmDb, loan *Loan) {
	iterator, err := db.NewStorageIterator(append(subscriptionKeyPrefix, common.Uint64ToBytes(loan.Id)...))
	if err != nil {
		panic(err)
	}
	defer iterator.Release()
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				panic(iterator.Error())
			}
			break
		}
		data := iterator.Value()
		sub := &Subscription{}
		if err = sub.DeSerialize(data); err != nil {
			panic(err)
		}
		amount := CalculateAmount(sub.Shares, sub.ShareAmount)
		if loan.Status == LoanExpiredRefunded {
			OnAccRefundSuccessSubscription(db, sub.Address, amount)
			AddBaseAccountEvent(db, sub.Address, BaseSubscribeExpiredRefund, 0, loan.Id, amount.Bytes())
		} else if loan.Status == LoanFailed {
			OnAccRefundFailedSubscription(db, sub.Address, amount)
			AddBaseAccountEvent(db, sub.Address, BaseSubscribeFailedRelease, 0, loan.Id, amount.Bytes())
		}
	}
}

func NewSubscription(address types.Address, db vm_db.VmDb, param *ParamSubscribe, loan *Loan) *Subscription {
	sub := &Subscription{}
	sub.LoanId = param.LoanId
	sub.Address = address.Bytes()
	sub.Token = loan.Token
	sub.Shares = param.Shares
	sub.ShareAmount = loan.ShareAmount
	sub.Status = LoanOpen
	sub.Created = GetDeFiTimestamp(db)
	return sub
}

func DoSubscribe(db vm_db.VmDb, gs util.GlobalStatus, loan *Loan, shares int32) {
	loan.SubscribedShares = loan.SubscribedShares + shares
	loan.Updated = GetDeFiTimestamp(db)
	if loan.Shares == loan.SubscribedShares {
		loan.Status = LoanSuccess
		loan.ExpireHeight = GetExpireHeight(gs, loan.ExpireDays)
		loan.StartTime = loan.Updated
		OnAccLoanSuccess(db, loan.Address, loan)
		AddBaseAccountEvent(db, loan.Address, BaseLoanInterestReduce, 0, loan.Id, loan.Interest)
		AddBaseAccountEvent(db, loan.Address, LoanNewSuccessLoan, 0, loan.Id, CalculateAmount(loan.Shares, loan.ShareAmount).Bytes())
	}
	SaveLoan(db, loan)
	AddLoanUpdateEvent(db, loan)
	if loan.Status == LoanSuccess {
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
			sub.Status = LoanSuccess
			sub.Updated = loan.Updated
			interest := CalculateInterest(sub.Shares, new(big.Int).SetBytes(sub.ShareAmount), loan.DayRate, loan.ExpireDays)
			if leaveLoanInterest.Cmp(interest) < 0 {
				interest = leaveLoanInterest
				leaveLoanInterest = big.NewInt(0)
			} else {
				leaveLoanInterest.Sub(leaveLoanInterest, interest)
			}
			sub.Interest = interest.Bytes()
			amount := CalculateAmount(sub.Shares, sub.ShareAmount)
			SaveSubscription(db, sub)
			AddSubscriptionUpdateEvent(db, sub)
			OnAccSubscribeSuccess(db, sub.Address, interest, amount)
			AddBaseAccountEvent(db, sub.Address, BaseSubscribeSuccessReduce, 0, loan.Id, amount.Bytes())
			AddBaseAccountEvent(db, sub.Address, BaseSubscribeInterestIncome, 0, loan.Id, sub.Interest)
		}
	}
}
