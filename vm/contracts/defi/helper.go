package defi

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/common"
	defiproto "github.com/vitelabs/go-vite/vm/contracts/defi/proto"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

var (
	LoanRateCardinalNum int32 = 1e6
	MinDayRate          int32 = 1 // 1/1,000,000
	MaxDayRate                = LoanRateCardinalNum

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

func DoSubscribe(db vm_db.VmDb, gs util.GlobalStatus, loan *Loan, shares int32) {
	loan.SubscribedShares = loan.SubscribedShares + shares
	loan.Updated = GetDeFiTimestamp(db)
	if loan.Shares == loan.SubscribedShares {
		loan.Status = Success
		loan.ExpireHeight = GetExpireHeight(gs, loan.ExpireDays)
		loan.StartTime = loan.Updated
		OnAccLoanSuccess(db, loan.Address, loan)
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

func PrepareInvest(db vm_db.VmDb, address types.Address, param *ParamInvest, leavedAmount, stakeAmountMin *big.Int, availableHeight, stakeHeight, stakeSVIPHeight uint64) (loanInvested, baseInvested *big.Int, durationHeight uint64, err error) {
	var (
		baseAvailable = new(big.Int)
		fund          *Fund
		acc           *defiproto.Account
		ok            bool
	)
	if fund, ok = GetFund(db, address); ok {
		if acc, ok = GetAccountInfo(fund, ledger.ViteTokenId); ok {
			baseAvailable.SetBytes(acc.BaseAccount.Available)
		}
	}
	totalAmount := new(big.Int).Add(leavedAmount, baseAvailable)
	switch param.BizType {
	case StakeForMining:
		if totalAmount.Cmp(dex.StakeForMiningMinAmount) < 0 {
			err = InvestAmountNotValidErr
			return
		} else if availableHeight < stakeHeight {
			err = AvailableHeightNotValidForInvestErr
			return
		}
		durationHeight = stakeHeight
		loanInvested, baseInvested = getInvestedAmount(leavedAmount, dex.StakeForMiningMinAmount)
	case StakeForSVIP:
		if totalAmount.Cmp(dex.StakeForSuperVIPAmount) < 0 {
			err = InvestAmountNotValidErr
			return
		} else if availableHeight < stakeSVIPHeight {
			err = AvailableHeightNotValidForInvestErr
			return
		}
		durationHeight = stakeSVIPHeight
		loanInvested, baseInvested = getInvestedAmount(leavedAmount, dex.StakeForSuperVIPAmount)
	case StakeForQuota:
		if totalAmount.Cmp(stakeAmountMin) < 0 {
			err = InvestAmountNotValidErr
			return
		} else if availableHeight < stakeHeight {
			err = AvailableHeightNotValidForInvestErr
			return
		}
		durationHeight = stakeHeight
		loanInvested, baseInvested = getInvestedAmount(leavedAmount, dex.StakeForSuperVIPAmount)
	}
	_, err = OnAccInvest(db, address, loanInvested, baseInvested)
	return
}

func DoStakeForMining(db vm_db.VmDb) ([]*ledger.AccountBlock, error) {

}

func DoStakeForSVIP(db vm_db.VmDb) ([]*ledger.AccountBlock, error) {

}

func getInvestedAmount(leavedLoanAmount *big.Int, needInvestAmount *big.Int) (loanInvested, baseInvested *big.Int) {
	if leavedLoanAmount.Cmp(needInvestAmount) < 0 {
		loanInvested = new(big.Int).Set(leavedLoanAmount)
		baseInvested = new(big.Int).Sub(needInvestAmount, loanInvested)
	} else {
		loanInvested = needInvestAmount
		baseInvested = new(big.Int)
	}
	return
}

func NewInvest(db vm_db.VmDb, gs util.GlobalStatus, address types.Address, loan *Loan, param *ParamInvest, loanInvested, baseInvested *big.Int, durationHeight uint64) *Invest {
	invest := &Invest{}
	invest.Id = NewInvestSerialNo(db)
	invest.LoanId = loan.Id
	invest.Address = address.Bytes()
	invest.LoanAmount = loanInvested.Bytes()
	invest.BaseAmount = baseInvested.Bytes()
	invest.BizType = param.BizType
	invest.Beneficial = param.Beneficiary.Bytes()
	invest.CreateHeight = gs.SnapshotBlock().Height
	invest.ExpireHeight = invest.CreateHeight + durationHeight
	invest.Status = InvestPending
	invest.Created = GetDeFiTimestamp(db)
	return invest
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
