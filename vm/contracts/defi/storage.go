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
	fundKeyPrefix              = []byte("fd:") //fund:address
	loanKeyPrefix              = []byte("ln:") //loan:loanId 3+8
	loanSerialNoKey            = []byte("lnSn:")
	subscriptionKeyPrefix      = []byte("sb:")    //subscription:loanId+address 3+8+20 = 31
	investKeyPrefix            = []byte("ivt:")   // invest:serialNo 4+8 = 19
	investToLoanIndexKeyPrefix = []byte("iv2LI:") // investId:
	investSerialNoKey          = []byte("ivtSn:") //invest:
	investQuotaInfoKeyPrefix   = []byte("ivQ:")

	sbpRegistrationKeyPrefix    = []byte("ivSp:")

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

//invest type
const (
	InvestForMining = iota + 1
	InvestForSVIP
	InvestForSBP
	InvestForQuota
)

//invest status
const (
	InvestPending = iota + 1
	InvestSuccess
	InvestCancelling
)

const (
	RefundFailedInvest = iota + 1
	RefundCancelledInvest
)

const (
	UpdateBlockProducintAddress = 1
	UpdateSBPRewardWithdrawAddress = 2
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

type ParamInvest struct {
	LoanId      uint64
	BizType     int32
	Amount      *big.Int
	Beneficiary types.Address
}

type ParamRefundInvest struct {
	InvestHashes []byte
	Reason       uint8
}

type ParamRegisterSBP struct {
	LoanId                uint64
	SbpName               string
	BlockProducingAddress types.Address
	RewardWithdrawAddress types.Address
}

type ParamUpdateSBPRegistration struct {
	InvestId              uint64
	OperationCode         uint8
	BlockProducingAddress types.Address
	RewardWithdrawAddress types.Address
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

type InvestQuotaInfo struct {
	defiproto.InvestQuotaInfo
}

func (ivq *InvestQuotaInfo) Serialize() (data []byte, err error) {
	return proto.Marshal(&ivq.InvestQuotaInfo)
}

func (ivq *InvestQuotaInfo) DeSerialize(data []byte) error {
	info := defiproto.InvestQuotaInfo{}
	if err := proto.Unmarshal(data, &info); err != nil {
		return err
	} else {
		ivq.InvestQuotaInfo = info
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

func OnAccLoanSuccess(db vm_db.VmDb, address []byte, loan *Loan) (*defiproto.Account, error) {
	addr, _ := types.BytesToAddress(address)
	return updateFund(db, addr, ledger.ViteTokenId.Bytes(), func(acc *defiproto.Account) (*defiproto.Account, error) {
		if common.CmpForBigInt(acc.BaseAccount.Locked, loan.Interest) < 0 {
			return nil, ExceedFundAvailableErr
		} else {
			acc.BaseAccount.Locked = common.SubBigInt(acc.BaseAccount.Locked, loan.Interest).Bytes()
		}
		acc.LoanAccount.Available = common.AddBigInt(acc.LoanAccount.Available, CalculateAmount(loan.Shares, loan.ShareAmount).Bytes())
		return acc, nil
	})
}

func OnAccLoanCancelled(db vm_db.VmDb, address types.Address, interest []byte) (*defiproto.Account, error) {
	return updateFund(db, address, ledger.ViteTokenId.Bytes(), func(acc *defiproto.Account) (*defiproto.Account, error) {
		if common.CmpForBigInt(acc.BaseAccount.Locked, interest) < 0 {
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

func OnAccInvest(db vm_db.VmDb, address types.Address, loanAmount, baseAmount *big.Int) (*defiproto.Account, error) {
	return updateFund(db, address, ledger.ViteTokenId.Bytes(), func(acc *defiproto.Account) (*defiproto.Account, error) {
		if loanAmount.Sign() != 0 {
			loanAvailable := new(big.Int).SetBytes(acc.LoanAccount.Available)
			if loanAvailable.Cmp(loanAmount) < 0 {
				return nil, ExceedFundAvailableErr
			} else {
				acc.LoanAccount.Available = loanAvailable.Sub(loanAvailable, loanAmount).Bytes()
				acc.LoanAccount.Invested = common.AddBigInt(acc.LoanAccount.Invested, loanAmount.Bytes())
			}
		}
		if baseAmount.Sign() != 0 {
			baseAvailable := new(big.Int).SetBytes(acc.BaseAccount.Available)
			if baseAvailable.Cmp(baseAmount) < 0 {
				return nil, ExceedFundAvailableErr
			} else {
				acc.BaseAccount.Available = baseAvailable.Sub(baseAvailable, baseAmount).Bytes()
				acc.BaseAccount.Invested = common.AddBigInt(acc.BaseAccount.Invested, baseAmount.Bytes())
			}
		}
		return acc, nil
	})
}

func OnAccRefundInvest(db vm_db.VmDb, address []byte, loanAmount, baseAmount []byte) (*defiproto.Account, error) {
	addr, _ := types.BytesToAddress(address)
	return updateFund(db, addr, ledger.ViteTokenId.Bytes(), func(acc *defiproto.Account) (*defiproto.Account, error) {
		if len(loanAmount) != 0 {
			if common.CmpForBigInt(acc.LoanAccount.Invested, loanAmount) < 0 {
				return nil, ExceedFundAvailableErr
			} else {
				acc.LoanAccount.Invested = common.SubBigIntAbs(acc.LoanAccount.Invested, loanAmount)
				acc.LoanAccount.Available = common.AddBigInt(acc.LoanAccount.Available, loanAmount)
			}
		}
		if len(baseAmount) > 0 {
			if common.CmpForBigInt(acc.BaseAccount.Invested, baseAmount) < 0 {
				return nil, ExceedFundAvailableErr
			} else {
				acc.BaseAccount.Invested = common.SubBigInt(acc.BaseAccount.Invested, baseAmount).Bytes()
				acc.BaseAccount.Available = common.AddBigInt(acc.BaseAccount.Available, baseAmount)
			}
		}
		return acc, nil
	})
}

func OnLoanInvest(db vm_db.VmDb, loan *Loan, amount *big.Int) {
	loan.Invested = common.AddBigInt(loan.Invested, amount.Bytes())
	SaveLoan(db, loan)
}

func OnLoanCancelInvest(db vm_db.VmDb, loanId uint64, amount []byte) error {
	if loan, ok := GetLoanById(db, loanId); !ok {
		return LoanNotExistsErr
	} else {
		if common.CmpForBigInt(loan.Invested, amount) < 0 {
			return ExceedFundAvailableErr
		} else {
			loan.Invested = common.SubBigIntAbs(loan.Invested, amount)
			SaveLoan(db, loan)
		}
	}
	return nil
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

func DeleteLoan(db vm_db.VmDb, loan *Loan) {
	common.SerializeToDb(db, getLoanKey(loan.Id), loan)
}

func GetLoanById(db vm_db.VmDb, loanId uint64) (loan *Loan, ok bool) {
	loan = &Loan{}
	ok = common.DeserializeFromDb(db, getLoanKey(loanId), loan)
	return
}

func GetLoanAvailable(gs util.GlobalStatus, loan *Loan) (available *big.Int, availableHeight uint64) {
	available = new(big.Int).Sub(CalculateAmount(loan.Shares, loan.ShareAmount), new(big.Int).SetBytes(loan.Invested))
	availableHeight = loan.ExpireHeight - gs.SnapshotBlock().Height
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
	return append(subscriptionKeyPrefix, append(common.Uint64ToBytes(loanId), address...)...)
}

func SaveInvest(db vm_db.VmDb, invest *Invest) {
	common.SerializeToDb(db, getInvestKey(invest.Id), invest)
}

func GetInvest(db vm_db.VmDb, investId uint64) (invest *Invest, ok bool) {
	invest = &Invest{}
	ok = common.DeserializeFromDb(db, getInvestKey(investId), invest)
	return
}

func ConfirmInvest(db vm_db.VmDb, invest *Invest) {
	invest.Status = InvestSuccess
	common.SerializeToDb(db, getInvestKey(invest.Id), invest)
}

func CancellingInvest(db vm_db.VmDb, invest *Invest) {
	invest.Status = InvestCancelling
	common.SerializeToDb(db, getInvestKey(invest.Id), invest)
}

func DeleteInvest(db vm_db.VmDb, investId uint64) {
	common.SetValueToDb(db, getInvestKey(investId), nil)
}

func getInvestKey(investId uint64) []byte {
	return append(investKeyPrefix, common.Uint64ToBytes(investId)...)
}

func SaveInvestToLoanIndex(db vm_db.VmDb, invest *Invest) {
	common.SetValueToDb(db, getInvestToLoanIndexKey(invest.LoanId, invest.Id), []byte{1})
}

func DeleteInvestToLoanIndex(db vm_db.VmDb, invest *Invest) {
	common.SetValueToDb(db, getInvestToLoanIndexKey(invest.LoanId, invest.Id), nil)
}

func getInvestToLoanIndexKey(loanId, investId uint64) []byte {
	return append(append(investToLoanIndexKeyPrefix, common.Uint64ToBytes(loanId)...), common.Uint64ToBytes(investId)...)
}

func NewLoanSerialNo(db vm_db.VmDb) (serialNo uint64) {
	return newSerialNo(db, loanSerialNoKey)
}

func NewInvestSerialNo(db vm_db.VmDb) (serialNo uint64) {
	return newSerialNo(db, investSerialNoKey)
}

func GetInvestQuotaInfo(db vm_db.VmDb, hash []byte) (info *InvestQuotaInfo, ok bool) {
	info = &InvestQuotaInfo{}
	ok = common.DeserializeFromDb(db, GetInvestQuotaInfoKey(hash), info)
	return
}

func SaveInvestQuotaInfo(db vm_db.VmDb, hash types.Hash, invest *Invest, amount *big.Int) {
	info := &InvestQuotaInfo{}
	info.Address = invest.Address
	info.Amount = amount.Bytes()
	info.InvestId = invest.Id
	common.SerializeToDb(db, GetInvestQuotaInfoKey(hash.Bytes()), info)
}

func DeleteInvestQuotaInfo(db vm_db.VmDb, hash []byte) {
	common.SetValueToDb(db, GetInvestQuotaInfoKey(hash), nil)
}

func GetInvestQuotaInfoKey(hash []byte) []byte {
	return append(investQuotaInfoKeyPrefix, hash[len(investQuotaInfoKeyPrefix):]...)
}

func GetSBPRegistration(db vm_db.VmDb, hash []byte) (info *SBPRegistration, ok bool) {
	info = &SBPRegistration{}
	ok = common.DeserializeFromDb(db, GetSBPRegistrationKey(hash), info)
	return
}

func SaveSBPRegistration(db vm_db.VmDb, hash types.Hash, param *ParamRegisterSBP, invest *Invest) {
	info := &SBPRegistration{}
	info.InvestId = invest.Id
	info.Name = param.SbpName
	info.ProducingAddress = param.BlockProducingAddress.Bytes()
	info.RewardWithdrawAddress = param.RewardWithdrawAddress.Bytes()
	common.SerializeToDb(db, GetSBPRegistrationKey(hash.Bytes()), info)
}

func DeleteSBPRegistration(db vm_db.VmDb, hash []byte) {
	common.SetValueToDb(db, GetSBPRegistrationKey(hash), nil)
}

func GetSBPRegistrationKey(hash []byte) []byte {
	return append(sbpRegistrationKeyPrefix, hash[len(sbpRegistrationKeyPrefix):]...)
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
