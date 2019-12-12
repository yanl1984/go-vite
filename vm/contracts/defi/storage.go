package defi

import (
	"bytes"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/common"
	defiproto "github.com/vitelabs/go-vite/vm/contracts/defi/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

var (
	fundKeyPrefix         = []byte("fd:") //fund:address
	loanKeyPrefix         = []byte("ln:") //loan:loanId 3+8
	loanSerialNoKey       = []byte("lnSn:")
	subscriptionKeyPrefix = []byte("sb:")    //subscription:loanId+address 3+8+20 = 31
	investKeyPrefix       = []byte("ivt:")   // invest:serialNo 4+8 = 19
	investSerialNoKey     = []byte("ivtSn:") //invest:

	defiTimestampKey = []byte("tmst:")
)

const (
	Open = iota + 1
	Success
	Failed
	Expired
	FailedRefunded
	ExpiredRefuned
)

type ParamWithdraw struct {
	Token  types.TokenTypeId
	Amount *big.Int
}

type ParamNewLoan struct {
	Token         types.TokenTypeId
	DayRate       int32
	ShareAmount   *big.Int
	Shares        int32
	SubscribeDays int32
	ExpireDays    int32
}

type ParamSubscribe struct {
	LoanId uint64
	Shares int32
}

type ParamStakeForMining struct {
	ActionType uint8 // 1: stake 2: cancel stake
	Amount     *big.Int
}

type ParamStakeForVIP struct {
	ActionType uint8 // 1: stake 2: cancel stake
}

type Fund struct {
	defiproto.Fund
}

func (fd *Fund) Serialize() (data []byte, err error) {
	return proto.Marshal(&fd.Fund)
}

func (fd *Fund) DeSerialize(data []byte) (err error) {
	protoFund := defiproto.Fund{}
	if err := proto.Unmarshal(data, &protoFund); err != nil {
		return err
	} else {
		fd.Fund = protoFund
		return nil
	}
}

type Loan struct {
	defiproto.Loan
}

func (ln *Loan) Serialize() (data []byte, err error) {
	return proto.Marshal(&ln.Loan)
}

func (ln *Loan) DeSerialize(data []byte) error {
	loan := defiproto.Loan{}
	if err := proto.Unmarshal(data, &loan); err != nil {
		return err
	} else {
		ln.Loan = loan
		return nil
	}
}

type Subscription struct {
	defiproto.Subscription
}

func (sb *Subscription) Serialize() (data []byte, err error) {
	return proto.Marshal(&sb.Subscription)
}

func (sb *Subscription) DeSerialize(data []byte) error {
	subscription := defiproto.Subscription{}
	if err := proto.Unmarshal(data, &subscription); err != nil {
		return err
	} else {
		sb.Subscription = subscription
		return nil
	}
}

type Invest struct {
	defiproto.Invest
}

func (iv *Invest) Serialize() (data []byte, err error) {
	return proto.Marshal(&iv.Invest)
}

func (iv *Invest) DeSerialize(data []byte) error {
	invest := defiproto.Invest{}
	if err := proto.Unmarshal(data, &invest); err != nil {
		return err
	} else {
		iv.Invest = invest
		return nil
	}
}

type SBPRegistration struct {
	defiproto.SBPRegistration
}

func (sbpr *SBPRegistration) Serialize() (data []byte, err error) {
	return proto.Marshal(&sbpr.SBPRegistration)
}

func (sbpr *SBPRegistration) DeSerialize(data []byte) error {
	sbpRegistration := defiproto.SBPRegistration{}
	if err := proto.Unmarshal(data, &sbpRegistration); err != nil {
		return err
	} else {
		sbpr.SBPRegistration = sbpRegistration
		return nil
	}
}

func GetAccountInfo(fund *Fund, token types.TokenTypeId) (*defiproto.Account, bool) {
	for _, a := range fund.Accounts {
		if bytes.Equal(token.Bytes(), a.Token) {
			return a, true
		}
	}
	return newAccount(token.Bytes()), false
}

func OnAccDeposit(db vm_db.VmDb, address types.Address, token types.TokenTypeId, amount *big.Int) (updatedAcc *defiproto.Account) {
	fund, _ := GetFund(db, address)
	var foundToken bool
	for _, acc := range fund.Accounts {
		if bytes.Equal(acc.Token, token.Bytes()) {
			acc.BaseAccount.Available = common.AddBigInt(acc.BaseAccount.Available, amount.Bytes())
			updatedAcc = acc
			foundToken = true
			break
		}
	}
	if !foundToken {
		updatedAcc = newAccount(token.Bytes())
		updatedAcc.BaseAccount.Available = amount.Bytes()
		fund.Accounts = append(fund.Accounts, updatedAcc)
	}
	SaveFund(db, address, fund)
	return
}

func OnAccWithdraw(db vm_db.VmDb, address types.Address, tokenId []byte, amount *big.Int) (*defiproto.Account, error) {
	return updateFund(db, address, tokenId, func(acc *defiproto.Account) (*defiproto.Account, error) {
		available := new(big.Int).SetBytes(acc.BaseAccount.Available)
		if available.Cmp(amount) < 0 {
			return nil, ExceedFundAvailableErr
		} else {
			acc.BaseAccount.Available = available.Sub(available, amount).Bytes()
		}
		return acc, nil
	})
}

func OnAccNewLoan(db vm_db.VmDb, address types.Address, interest *big.Int) (*defiproto.Account, error) {
	return updateFund(db, address, ledger.ViteTokenId.Bytes(), func(acc *defiproto.Account) (*defiproto.Account, error) {
		available := new(big.Int).SetBytes(acc.BaseAccount.Available)
		if available.Cmp(interest) < 0 {
			return nil, ExceedFundAvailableErr
		} else {
			acc.BaseAccount.Available = available.Sub(available, interest).Bytes()
			acc.BaseAccount.Locked = common.AddBigInt(acc.BaseAccount.Locked, interest.Bytes())
		}
		return acc, nil
	})
}

func OnAccLoanSuccess(db vm_db.VmDb, address []byte, interest []byte) (*defiproto.Account, error) {
	addr, _ := types.BytesToAddress(address)
	return updateFund(db, addr, ledger.ViteTokenId.Bytes(), func(acc *defiproto.Account) (*defiproto.Account, error) {
		if common.CmpForBigInt(acc.BaseAccount.Locked, interest)< 0 {
			return nil, ExceedFundAvailableErr
		} else {
			acc.BaseAccount.Locked = common.SubBigInt(acc.BaseAccount.Locked, interest).Bytes()
		}
		return acc, nil
	})
}

func OnAccLoanCancelled(db vm_db.VmDb, address types.Address, interest []byte) (*defiproto.Account, error) {
	return updateFund(db, address, ledger.ViteTokenId.Bytes(), func(acc *defiproto.Account) (*defiproto.Account, error) {
		if common.CmpForBigInt(acc.BaseAccount.Locked, interest)< 0 {
			return nil, ExceedFundAvailableErr
		} else {
			acc.BaseAccount.Available = common.AddBigInt(acc.BaseAccount.Available, interest)
		}
		return acc, nil
	})
}

func OnAccSubscribe(db vm_db.VmDb, address types.Address, amount *big.Int) (*defiproto.Account, error) {
	return updateFund(db, address, ledger.ViteTokenId.Bytes(), func(acc *defiproto.Account) (*defiproto.Account, error) {
		available := new(big.Int).SetBytes(acc.BaseAccount.Available)
		if available.Cmp(amount) < 0 {
			return nil, ExceedFundAvailableErr
		} else {
			acc.BaseAccount.Available = available.Sub(available, amount).Bytes()
			acc.BaseAccount.Subscribed = common.AddBigInt(acc.BaseAccount.Subscribed, amount.Bytes())
		}
		return acc, nil
	})
}

func OnAccSubscribeSuccess(db vm_db.VmDb, address []byte, interest, amount *big.Int) (*defiproto.Account, error) {
	addr, _ := types.BytesToAddress(address)
	return updateFund(db, addr, ledger.ViteTokenId.Bytes(), func(acc *defiproto.Account) (*defiproto.Account, error) {
		subscribed := new(big.Int).SetBytes(acc.BaseAccount.Subscribed)
		if subscribed.Cmp(amount) < 0 {
			return nil, ExceedFundAvailableErr
		} else {
			acc.BaseAccount.Available = common.AddBigInt(acc.BaseAccount.Available, interest.Bytes())
			acc.BaseAccount.Subscribed = common.SubBigInt(acc.BaseAccount.Subscribed, amount.Bytes()).Bytes()
		}
		return acc, nil
	})
}

func OnAccInvest(db vm_db.VmDb, address types.Address, tokenId []byte, baseInvest, loanInvest *big.Int) (*defiproto.Account, error) {
	return updateFund(db, address, tokenId, func(acc *defiproto.Account) (*defiproto.Account, error) {
		if baseInvest != nil {
			baseAvailable := new(big.Int).SetBytes(acc.BaseAccount.Available)
			if baseAvailable.Cmp(baseInvest) < 0 {
				return nil, ExceedFundAvailableErr
			} else {
				acc.BaseAccount.Available = baseAvailable.Sub(baseAvailable, baseInvest).Bytes()
				acc.BaseAccount.Invested = common.AddBigInt(acc.BaseAccount.Invested, baseInvest.Bytes())
			}
		}
		if loanInvest != nil {
			loanAvailable := new(big.Int).SetBytes(acc.LoanAccount.Available)
			if loanAvailable.Cmp(loanInvest) < 0 {
				return nil, ExceedFundAvailableErr
			} else {
				acc.LoanAccount.Available = loanAvailable.Sub(loanAvailable, loanInvest).Bytes()
				acc.LoanAccount.Invested = common.AddBigInt(acc.LoanAccount.Invested, loanInvest.Bytes())
			}
		}
		return acc, nil
	})
}

func updateFund(db vm_db.VmDb, address types.Address, tokenId []byte, updateAccFunc func(*defiproto.Account) (*defiproto.Account, error)) (updatedAcc *defiproto.Account, err error) {
	if fund, ok := GetFund(db, address); ok {
		var foundAcc bool
		for _, acc := range fund.Accounts {
			if bytes.Equal(acc.Token, tokenId) {
				foundAcc = true
				if updatedAcc, err = updateAccFunc(acc); err != nil {
					return
				}
				break
			}
		}
		if foundAcc {
			SaveFund(db, address, fund)
		} else {
			err = ExceedFundAvailableErr
		}
	} else {
		err = ExceedFundAvailableErr
	}
	return
}

func GetFund(db vm_db.VmDb, address types.Address) (fund *Fund, ok bool) {
	fund = &Fund{}
	ok = common.DeserializeFromDb(db, GetFundKey(address), fund)
	return
}

func SaveFund(db vm_db.VmDb, address types.Address, fund *Fund) {
	common.SerializeToDb(db, GetFundKey(address), fund)
}

func GetFundKey(address types.Address) []byte {
	return append(fundKeyPrefix, address.Bytes()...)
}

func SaveLoan(db vm_db.VmDb, loan *Loan) {
	common.SerializeToDb(db, getLoanKey(loan.Id), loan)
}

func GetLoanById(db vm_db.VmDb, loanId uint64) (loan *Loan, ok bool) {
	loan = &Loan{}
	ok = common.DeserializeFromDb(db, getLoanKey(loanId), loan)
	return
}

func IsOpenLoanSubscribeFail(db vm_db.VmDb, loan *Loan) bool {
	return loan.Status == Open && GetDeFiTimestamp(db) > GetLoanFailTime(loan)
}

func getLoanKey(loanId uint64) []byte {
	return append(loanKeyPrefix, common.Uint64ToBytes(loanId)...)
}

func SaveSubscription(db vm_db.VmDb, sub *Subscription) {
	common.SerializeToDb(db, getSubscriptionKey(sub.LoanId, sub.Address), sub)
}

func GetSubscription(db vm_db.VmDb, loanId uint64, address []byte) (sub *Subscription, ok bool) {
	sub = &Subscription{}
	ok = common.DeserializeFromDb(db, getSubscriptionKey(loanId, address), sub)
	return
}

func getSubscriptionKey(loanId uint64, address []byte) []byte {
	return append(common.Uint64ToBytes(loanId), address...)
}

func NewLoanSerialNo(db vm_db.VmDb) (serialNo uint64) {
	return newSerialNo(db, loanSerialNoKey)
}

func NewSubscriptionSerialNo(db vm_db.VmDb) (serialNo uint64) {
	return newSerialNo(db, investSerialNoKey)
}

func NewInvestSerialNo(db vm_db.VmDb) (serialNo uint64) {
	return newSerialNo(db, investSerialNoKey)
}

func GetDeFiTimestamp(db vm_db.VmDb) int64 {
	if bs := common.GetValueFromDb(db, defiTimestampKey); len(bs) == 8 {
		return int64(common.BytesToUint64(bs))
	} else {
		return 0
	}
}

func GetExpireHeight(gs util.GlobalStatus, days int32) uint64 {
	return gs.SnapshotBlock().Height + uint64(days*24*3600)
}

func GetLoanFailTime(loan *Loan) int64 {
	return loan.Created + int64(loan.SubscribeDays*24*3600)
}

func newAccount(token []byte) *defiproto.Account {
	account := &defiproto.Account{}
	account.Token = token
	account.BaseAccount = &defiproto.BaseAccount{}
	account.LoanAccount = &defiproto.LoanAccount{}
	return account
}

func newSerialNo(db vm_db.VmDb, key []byte) (serialNo uint64) {
	if data := common.GetValueFromDb(db, key); len(data) == 8 {
		serialNo = common.BytesToUint64(data)
		serialNo++
	} else {
		serialNo = 1
	}
	common.SetValueToDb(db, key, common.Uint64ToBytes(serialNo))
	return
}
