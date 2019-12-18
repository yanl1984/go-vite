package defi

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
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

func PrepareInvest(db vm_db.VmDb, address types.Address, bizType int32, loanAvailable, stakeAmount *big.Int, availableHeight, stakeHeightMin, stakeSVIPHeight uint64) (baseInvested, loanInvested *big.Int, durationHeight uint64, err error) {
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
	totalAvailable := new(big.Int).Add(loanAvailable, baseAvailable)
	switch bizType {
	case InvestForMining:
		if totalAvailable.Cmp(dex.StakeForMiningMinAmount) < 0 {
			err = ExceedFundAvailableErr
			return
		} else if availableHeight < stakeHeightMin {
			err = AvailableHeightNotValidForInvestErr
			return
		}
		durationHeight = stakeHeightMin
		loanInvested, baseInvested = getInvestedAmount(loanAvailable, stakeAmount)
	case InvestForSVIP:
		if totalAvailable.Cmp(dex.StakeForSuperVIPAmount) < 0 {
			err = ExceedFundAvailableErr
			return
		} else if availableHeight < stakeSVIPHeight {
			err = AvailableHeightNotValidForInvestErr
			return
		}
		durationHeight = stakeSVIPHeight
		loanInvested, baseInvested = getInvestedAmount(loanAvailable, dex.StakeForSuperVIPAmount)
	case InvestForQuota:
		if totalAvailable.Cmp(dex.StakeForMiningMinAmount) < 0 {
			err = ExceedFundAvailableErr
			return
		} else if availableHeight < stakeHeightMin {
			err = AvailableHeightNotValidForInvestErr
			return
		}
		durationHeight = stakeHeightMin
		loanInvested, baseInvested = getInvestedAmount(loanAvailable, stakeAmount)
	case InvestForSBP:
		if totalAvailable.Cmp(stakeAmount) < 0 {
			err = ExceedFundAvailableErr
			return
		} else if availableHeight < stakeHeightMin {
			err = AvailableHeightNotValidForInvestErr
			return
		}
		durationHeight = stakeHeightMin
		loanInvested, baseInvested = getInvestedAmount(loanAvailable, stakeAmount)
	}
	return
}

func NewInvest(db vm_db.VmDb, gs util.GlobalStatus, address types.Address, loan *Loan, bizType int32, beneficiary types.Address, loanInvested, baseInvested *big.Int, durationHeight uint64) *Invest {
	invest := &Invest{}
	invest.Id = NewInvestSerialNo(db)
	invest.LoanId = loan.Id
	invest.Address = address.Bytes()
	invest.LoanAmount = loanInvested.Bytes()
	invest.BaseAmount = baseInvested.Bytes()
	invest.BizType = bizType
	invest.Beneficial = beneficiary.Bytes()
	invest.CreateHeight = gs.SnapshotBlock().Height
	invest.ExpireHeight = invest.CreateHeight + durationHeight
	invest.Status = InvestPending
	invest.Created = GetDeFiTimestamp(db)
	return invest
}

func DoDexInvest(invest *Invest, bizType uint8, amount *big.Int) ([]*ledger.AccountBlock, error) {
	if data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundDelegateInvest, invest.Id, invest.Address, uint8(bizType), invest.Beneficial); err != nil {
		return nil, err
	} else {
		return []*ledger.AccountBlock{
			{
				AccountAddress: types.AddressDeFi,
				ToAddress:      types.AddressDexFund,
				BlockType:      ledger.BlockTypeSendCall,
				Amount:         amount,
				TokenId:        ledger.ViteTokenId,
				Data:           data,
			},
		}, nil
	}
}

func DoCancelDexInvest(investId uint64) (blocks []*ledger.AccountBlock, err error) {
	if data, err := abi.ABIDexFund.PackMethod(abi.MethodNameDexFundCancelDelegateInvest, investId); err != nil {
		return nil, err
	} else {
		return []*ledger.AccountBlock{
			{
				AccountAddress: types.AddressDeFi,
				ToAddress:      types.AddressDexFund,
				BlockType:      ledger.BlockTypeSendCall,
				Amount:         big.NewInt(0),
				TokenId:        ledger.ViteTokenId,
				Data:           data,
			},
		}, nil
	}
}

func DoQuotaInvest(db vm_db.VmDb, invest *Invest, amount *big.Int, stakeHeight uint64, block *ledger.AccountBlock) (blocks []*ledger.AccountBlock, err error) {
	var (
		data    []byte
		stakeId types.Hash
	)
	if data, err = abi.ABIQuota.PackMethod(abi.MethodNameStakeWithCallback, types.AddressDeFi, stakeHeight); err != nil {
		return nil, err
	}
	blocks = []*ledger.AccountBlock{
		{
			AccountAddress: types.AddressDeFi,
			ToAddress:      types.AddressQuota,
			BlockType:      ledger.BlockTypeSendCall,
			Amount:         amount,
			TokenId:        ledger.ViteTokenId,
			Data:           data,
		},
	}
	stakeId = util.ComputeSendBlockHash(block, blocks[0], 0)
	invest.InvestHash = stakeId.Bytes()
	SaveInvest(db, invest)
	SaveInvestQuotaInfo(db, stakeId, invest, amount)
	return
}

func DoCancelQuotaInvest(investId []byte) (blocks []*ledger.AccountBlock, err error) {
	stakeId, _ := types.BytesToHash(investId)
	if data, err := abi.ABIQuota.PackMethod(abi.MethodNameCancelStakeWithCallback, stakeId); err != nil {
		return nil, err
	} else {
		return []*ledger.AccountBlock{
			{
				AccountAddress: types.AddressDeFi,
				ToAddress:      types.AddressQuota,
				BlockType:      ledger.BlockTypeSendCall,
				Amount:         big.NewInt(0),
				TokenId:        ledger.ViteTokenId,
				Data:           data,
			},
		}, nil
	}
}

func DoRegisterSBP(db vm_db.VmDb, invest *Invest, param *ParamRegisterSBP, amount *big.Int, block *ledger.AccountBlock) (blocks []*ledger.AccountBlock, err error) {
	var (
		data           []byte
		registrationId types.Hash
	)
	if data, err = abi.ABIGovernance.PackMethod(abi.MethodNameRegisterV3, param.SbpName, param.BlockProducingAddress, param.RewardWithdrawAddress); err != nil {
		return nil, err
	} else {
		return []*ledger.AccountBlock{
			{
				AccountAddress: types.AddressDeFi,
				ToAddress:      types.AddressGovernance,
				BlockType:      ledger.BlockTypeSendCall,
				Amount:         amount,
				TokenId:        ledger.ViteTokenId,
				Data:           data,
			},
		}, nil
	}
	registrationId = util.ComputeSendBlockHash(block, blocks[0], 0)
	invest.InvestHash = registrationId.Bytes()
	SaveInvest(db, invest)
	SaveSBPRegistration(db, registrationId, param, invest)
	return
}

func DoRevokeSBP(db vm_db.VmDb, traceHash []byte) (blocks []*ledger.AccountBlock, err error) {
	if info, ok := GetSBPRegistration(db, traceHash); !ok {
		panic(SBPRegistrationNotExistsErr)
	} else {
		if data, err := abi.ABIGovernance.PackMethod(abi.MethodNameRevokeV3, info.Name); err != nil {
			return nil, err
		} else {
			return []*ledger.AccountBlock{
				{
					AccountAddress: types.AddressDeFi,
					ToAddress:      types.AddressQuota,
					BlockType:      ledger.BlockTypeSendCall,
					Amount:         big.NewInt(0),
					TokenId:        ledger.ViteTokenId,
					Data:           data,
				},
			}, nil
		}
	}
}

func DoUpdateSBP(db vm_db.VmDb, traceHash []byte, param *ParamUpdateSBPRegistration) (blocks []*ledger.AccountBlock, err error) {
	if info, ok := GetSBPRegistration(db, traceHash); !ok {
		panic(SBPRegistrationNotExistsErr)
	} else {
		if common.IsOperationValidWithMask(param.OperationCode, UpdateBlockProducintAddress) {
			if data, err := abi.ABIGovernance.PackMethod(abi.MethodNameUpdateBlockProducintAddressV3, info.Name, param.BlockProducingAddress); err != nil {
				return nil, err
			} else {
				blocks = []*ledger.AccountBlock{
					{
						AccountAddress: types.AddressDeFi,
						ToAddress:      types.AddressGovernance,
						BlockType:      ledger.BlockTypeSendCall,
						Amount:         big.NewInt(0),
						TokenId:        ledger.ViteTokenId,
						Data:           data,
					},
				}
			}
		}
		if common.IsOperationValidWithMask(param.OperationCode, UpdateSBPRewardWithdrawAddress) {
			if data, err := abi.ABIGovernance.PackMethod(abi.MethodNameUpdateSBPRewardWithdrawAddress, info.Name, param.RewardWithdrawAddress); err != nil {
				return nil, err
			} else {
				blocks = append(blocks, &ledger.AccountBlock{
					AccountAddress: types.AddressDeFi,
					ToAddress:      types.AddressGovernance,
					BlockType:      ledger.BlockTypeSendCall,
					Amount:         big.NewInt(0),
					TokenId:        ledger.ViteTokenId,
					Data:           data,
				})
			}
		}
		return
	}
}

func DoRefundInvest(db vm_db.VmDb, invest *Invest) {
	OnAccRefundInvest(db, invest.Address, invest.LoanAmount, invest.BaseAmount)
	OnLoanCancelInvest(db, invest.LoanId, invest.LoanAmount)
	invest.Status = InvestCancelled
	DeleteInvest(db, invest.Id)
	DeleteInvestToLoanIndex(db, invest)
}

func DoRefundQuotaInvest(db vm_db.VmDb, invest *Invest) {
	DoRefundInvest(db, invest)
	invest.Status = InvestCancelled
	DeleteInvestQuotaInfo(db, invest.InvestHash)
}

func DoRefundSBPInvest(db vm_db.VmDb, invest *Invest) {
	DoRefundInvest(db, invest)
	invest.Status = InvestCancelled
	DeleteSBPRegistration(db, invest.InvestHash)
}

func HandleDexRefundOnFail(db vm_db.VmDb, block *ledger.AccountBlock) (blocks []*ledger.AccountBlock, err error) {
	param := new(dex.ParamDelegateInvest)
	if err = abi.ABIDexFund.UnpackMethod(param, abi.MethodNameDexFundDelegateInvest, block.Data); err != nil {
		return
	}
	if invest, ok := GetInvest(db, param.InvestId); ok && invest.Status == InvestPending {
		DoRefundInvest(db, invest)
	} else {
		err = InvestNotExistsErr
	}
	return
}

func HandleGovernanceFeedback(db vm_db.VmDb, block *ledger.AccountBlock) (blocks []*ledger.AccountBlock, err error) {
	var (
		isRevoke bool
		param    = new(abi.ParamRegister)
		sbpName  = new(string)
	)
	if err = abi.ABIGovernance.UnpackMethod(param, abi.MethodNameRegisterV3, block.Data); err == nil {
		*sbpName = param.SbpName
	} else if err = abi.ABIGovernance.UnpackMethod(sbpName, abi.MethodNameRevokeV3, block.Data); err == nil {
		isRevoke = true
	} else {
		err = InvalidInputParamErr
		return
	}
	var iterator interfaces.StorageIterator
	iterator, err = db.NewStorageIterator(sbpRegistrationKeyPrefix)
	if err != nil {
		panic(err)
	}
	defer iterator.Release()
	for {
		var regValue []byte
		if !iterator.Next() {
			if iterator.Error() != nil {
				panic(iterator.Error())
			}
			break
		}
		regValue = iterator.Value()
		sbpRegistration := &SBPRegistration{}
		sbpRegistration.DeSerialize(regValue)
		if sbpRegistration.Name == param.SbpName {
			if invest, ok := GetInvest(db, sbpRegistration.InvestId); !ok {
				err = InvestNotExistsErr
				return
			} else {
				if isRevoke && invest.Status != InvestCancelling || !isRevoke && invest.Status != InvestPending {
					err = InvalidInvestStatusErr
				} else {
					DoRefundSBPInvest(db, invest)
				}
			}
			return
		}
	}
	err = SBPRegistrationNotExistsErr
	return
}

func IsInvestExpired(gs util.GlobalStatus, invest *Invest) bool {
	return gs.SnapshotBlock().Height >= invest.ExpireHeight
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
