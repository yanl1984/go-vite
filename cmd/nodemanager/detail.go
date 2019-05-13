package nodemanager

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"os"
	"strings"
)

type accountDetail struct {
	// result
	vcpFinalBalance  *big.Int
	viteFinalBalance *big.Int
	// vcp detail
	vcpAccountBalance *big.Int
	vcpOnroadBalance  *big.Int
	vcpOnroadHashList []types.Hash
	// vite detail
	viteAccountBalance           *big.Int
	viteOnroadBalance            *big.Int
	viteOnroadHashList           []types.Hash
	viteToContractOnroadBalance  *big.Int
	viteToContractOnroadHashList []types.Hash
	viteRegisterRefundBalance    *big.Int
	viteRegisterRefundNameList   []string
	viteContractRefundFee        *big.Int
	viteContractRefundBalance    *big.Int
	viteContractAddrList         []types.Address
	// info
	vitePledgeBeneficialMap map[types.Address]*big.Int
	viteVoteSbpName         string
	viteTokenList           []types.TokenTypeId
	viteTokenRefundFee      *big.Int
	viteTokenRefundList     []types.TokenTypeId
}
type accountDetailMap map[types.Address]*accountDetail

func newAccountDetails() accountDetailMap {
	return make(map[types.Address]*accountDetail)
}

func newAccountDetail() *accountDetail {
	return &accountDetail{
		vcpFinalBalance:              big.NewInt(0),
		viteFinalBalance:             big.NewInt(0),
		vcpAccountBalance:            big.NewInt(0),
		vcpOnroadBalance:             big.NewInt(0),
		vcpOnroadHashList:            make([]types.Hash, 0),
		viteAccountBalance:           big.NewInt(0),
		viteOnroadBalance:            big.NewInt(0),
		viteOnroadHashList:           make([]types.Hash, 0),
		viteToContractOnroadBalance:  big.NewInt(0),
		viteToContractOnroadHashList: make([]types.Hash, 0),
		viteRegisterRefundBalance:    big.NewInt(0),
		viteRegisterRefundNameList:   make([]string, 0),
		viteContractRefundFee:        big.NewInt(0),
		viteContractRefundBalance:    big.NewInt(0),
		viteContractAddrList:         make([]types.Address, 0),
		vitePledgeBeneficialMap:      make(map[types.Address]*big.Int),
		viteTokenList:                make([]types.TokenTypeId, 0),
		viteTokenRefundFee:           big.NewInt(0),
		viteTokenRefundList:          make([]types.TokenTypeId, 0),
	}
}
func (m accountDetailMap) accountDetail(account types.Address) *accountDetail {
	if d, ok := m[account]; ok {
		return d
	}
	m[account] = newAccountDetail()
	return m[account]

}
func (m accountDetailMap) setVcpAccountBalance(account types.Address, amount *big.Int) {
	d := m.accountDetail(account)
	d.vcpAccountBalance.Set(amount)
	d.vcpFinalBalance.Add(d.vcpFinalBalance, amount)
}
func (m accountDetailMap) addVcpOnroadBalance(account types.Address, amount *big.Int, hash types.Hash) {
	d := m.accountDetail(account)
	d.vcpOnroadBalance.Add(d.vcpOnroadBalance, amount)
	d.vcpFinalBalance.Add(d.vcpFinalBalance, amount)
	d.vcpOnroadHashList = append(d.vcpOnroadHashList, hash)
}
func (m accountDetailMap) setViteAccountBalance(account types.Address, amount *big.Int) {
	d := m.accountDetail(account)
	d.viteAccountBalance.Set(amount)
	d.viteFinalBalance.Add(d.viteFinalBalance, amount)
}
func (m accountDetailMap) addViteAccountBalance(account types.Address, amount *big.Int) {
	d := m.accountDetail(account)
	d.viteAccountBalance.Add(d.viteAccountBalance, amount)
	d.viteFinalBalance.Add(d.viteFinalBalance, amount)
}
func (m accountDetailMap) addViteOnroadBalance(account types.Address, amount *big.Int, hash types.Hash) {
	d := m.accountDetail(account)
	d.viteOnroadBalance.Add(d.viteOnroadBalance, amount)
	d.viteFinalBalance.Add(d.viteFinalBalance, amount)
	d.viteOnroadHashList = append(d.viteOnroadHashList, hash)
}
func (m accountDetailMap) addViteToContractOnroadBalance(account types.Address, amount *big.Int, hash types.Hash) {
	d := m.accountDetail(account)
	d.viteToContractOnroadBalance.Add(d.viteToContractOnroadBalance, amount)
	d.viteFinalBalance.Add(d.viteFinalBalance, amount)
	d.viteToContractOnroadHashList = append(d.viteToContractOnroadHashList, hash)
}
func (m accountDetailMap) addRegisterRefund(account types.Address, name string) {
	d := m.accountDetail(account)
	d.viteFinalBalance.Add(d.viteFinalBalance, registerRefundPledgeAmount)
	d.viteRegisterRefundBalance.Add(d.viteRegisterRefundBalance, registerRefundPledgeAmount)
	d.viteRegisterRefundNameList = append(d.viteRegisterRefundNameList, name)
}
func (m accountDetailMap) addContractRefund(account types.Address, addr types.Address, amount *big.Int) {
	d := m.accountDetail(account)
	d.viteFinalBalance.Add(d.viteFinalBalance, amount)
	d.viteFinalBalance.Add(d.viteFinalBalance, contractFee)
	d.viteContractRefundFee.Add(d.viteContractRefundFee, contractFee)
	d.viteContractRefundBalance.Add(d.viteContractRefundBalance, amount)
	d.viteContractAddrList = append(d.viteContractAddrList, addr)
}
func (m accountDetailMap) addPledgeInfo(account types.Address, beneficial types.Address, amount *big.Int) {
	d := m.accountDetail(account)
	d.vitePledgeBeneficialMap[beneficial] = amount
}
func (m accountDetailMap) addVoteInfo(account types.Address, name string) {
	d := m.accountDetail(account)
	d.viteVoteSbpName = name
}
func (m accountDetailMap) addToken(account types.Address, id types.TokenTypeId) {
	d := m.accountDetail(account)
	d.viteTokenList = append(d.viteTokenList, id)
}
func (m accountDetailMap) addTokenRefund(account types.Address, id types.TokenTypeId) {
	d := m.accountDetail(account)
	d.viteTokenRefundList = append(d.viteTokenRefundList, id)
	d.viteTokenRefundFee.Add(d.viteTokenRefundFee, mintageFee)
	d.viteFinalBalance.Add(d.viteFinalBalance, mintageFee)
}

func (d accountDetail) calcVcpFinalBalance() *big.Int {
	amount := new(big.Int).Add(d.vcpAccountBalance, d.vcpOnroadBalance)
	return amount
}
func (d accountDetail) calcViteFinalBalance() *big.Int {
	amount := new(big.Int).Add(d.viteAccountBalance, d.viteOnroadBalance)
	amount.Add(amount, d.viteToContractOnroadBalance)
	amount.Add(amount, d.viteRegisterRefundBalance)
	amount.Add(amount, d.viteContractRefundFee)
	amount.Add(amount, d.viteContractRefundBalance)
	amount.Add(amount, d.viteTokenRefundFee)
	return amount
}

var seperateStr = "\t"

func (m accountDetailMap) print() {
	str := "address" + seperateStr +
		"vcpFinalBalance" + seperateStr +
		"viteFinalBalance" + seperateStr +
		"vcpAccountBalance" + seperateStr +
		"vcpOnroadBalance" + seperateStr +
		"vcpOnroadHashList" + seperateStr +
		"viteAccountBalance" + seperateStr +
		"viteOnroadBalance" + seperateStr +
		"viteOnroadHashList" + seperateStr +
		"viteToContractOnroadBalance" + seperateStr +
		"viteToContractOnroadHashList" + seperateStr +
		"viteRegisterRefundBalance" + seperateStr +
		"viteRegisterRefundNameList" + seperateStr +
		"viteContractRefundFee " + seperateStr +
		"viteContractRefundBalance" + seperateStr +
		"viteContractAddrList" + seperateStr +
		"vitePledgeBeneficialMap" + seperateStr +
		"viteVoteSbpName" + seperateStr +
		"viteTokenList" + seperateStr +
		"viteTokenRefundFee" + seperateStr +
		"viteTokenRefundList" + "\n"
	for addr, detail := range m {
		str = str + addr.String() + seperateStr +
			detail.vcpFinalBalance.String() + seperateStr +
			detail.viteFinalBalance.String() + seperateStr +
			detail.vcpAccountBalance.String() + seperateStr +
			detail.vcpOnroadBalance.String() + seperateStr +
			printHashList(detail.vcpOnroadHashList) + seperateStr +
			detail.viteAccountBalance.String() + seperateStr +
			detail.viteOnroadBalance.String() + seperateStr +
			printHashList(detail.viteOnroadHashList) + seperateStr +
			detail.viteToContractOnroadBalance.String() + seperateStr +
			printHashList(detail.viteToContractOnroadHashList) + seperateStr +
			detail.viteRegisterRefundBalance.String() + seperateStr +
			printStringList(detail.viteRegisterRefundNameList) + seperateStr +
			detail.viteContractRefundFee.String() + seperateStr +
			detail.viteContractRefundBalance.String() + seperateStr +
			printAddrList(detail.viteContractAddrList) + seperateStr +
			pringPledgeBeneficialMap(detail.vitePledgeBeneficialMap) + seperateStr +
			detail.viteVoteSbpName + seperateStr +
			printTokenList(detail.viteTokenList) + seperateStr +
			detail.viteTokenRefundFee.String() + seperateStr +
			printTokenList(detail.viteTokenRefundList) + "\n"
	}
	writeFile("/Users/chenping/Desktop/export.csv", str)
}
func printHashList(l []types.Hash) string {
	str := ""
	if len(l) == 0 {
		return str
	}
	for _, h := range l {
		str = str + h.String() + ","
	}
	return str[:len(str)-1]
}
func printAddrList(l []types.Address) string {
	str := ""
	if len(l) == 0 {
		return str
	}
	for _, h := range l {
		str = str + h.String() + ","
	}
	return str[:len(str)-1]
}
func printTokenList(l []types.TokenTypeId) string {
	str := ""
	if len(l) == 0 {
		return str
	}
	for _, h := range l {
		str = str + h.String() + ","
	}
	return str[:len(str)-1]
}
func printStringList(l []string) string {
	return strings.Join(l, ",")
}
func pringPledgeBeneficialMap(m map[types.Address]*big.Int) string {
	str := ""
	if len(m) == 0 {
		return str
	}
	for k, v := range m {
		str = str + k.String() + ":" + v.String() + ","
	}
	return str[:len(str)-1]
}

func writeFile(name, content string) {
	fileObj, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Failed to open the file", err.Error())
		os.Exit(2)
	}
	defer fileObj.Close()
	if _, err := fileObj.WriteString(content); err == nil {
		fmt.Println("Successful writing to the file with os.OpenFile and *File.WriteString method.", content)
	}
}
