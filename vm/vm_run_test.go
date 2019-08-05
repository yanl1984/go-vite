package vm

import (
	"encoding/hex"
	"encoding/json"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"
)

type VMRunTestCase struct {
	// global status
	SbHeight uint64
	// block
	BlockType     byte
	SendBlockType byte
	FromAddress   types.Address
	ToAddress     types.Address
	Data          string
	Amount        string
	TokenId       types.TokenTypeId
	Fee           string
	Code          string
	// environment
	PledgeBeneficialAmount string
	PreStorage             map[string]string
	PreBalanceMap          map[types.TokenTypeId]string
	ContractMetaMap        map[types.Address]*ledger.ContractMeta
	// result
	Err           string
	IsRetry       bool
	Success       bool
	Quota         uint64
	QuotaUsed     uint64
	SendBlockList []*TestCaseSendBlock
	LogList       []TestLog
	Storage       map[string]string
	BalanceMap    map[types.TokenTypeId]string
}

var (
	quotaInfoList = commonQuotaInfoList()
	prevHash, _   = types.HexToHash("82a8ecfe0df3dea6256651ee3130747386d4d6ab61201ce0050a6fe394a0f595")
	// testAddr,_ = types.HexToAddress("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	// testContractAddr,_ = types.HexToAddress("vite_a3ab3f8ce81936636af4c6f4da41612f11136d71f53bf8fa86")
)

func TestVM_RunV2(t *testing.T) {
	testDir := "./test/run_test/"
	testFiles, ok := ioutil.ReadDir(testDir)
	if ok != nil {
		t.Fatalf("read dir failed, %v", ok)
	}
	for _, testFile := range testFiles {
		if testFile.IsDir() {
			continue
		}
		/*if testFile.Name() != "contract.json" {
			continue
		}*/
		file, ok := os.Open(testDir + testFile.Name())
		if ok != nil {
			t.Fatalf("open test file failed, %v", ok)
		}
		testCaseMap := new(map[string]VMRunTestCase)
		if ok := json.NewDecoder(file).Decode(testCaseMap); ok != nil {
			t.Fatalf("decode test file failed, %v", ok)
		}
		for k, testCase := range *testCaseMap {
			currentTime := time.Now()
			latestSnapshotBlock := &ledger.SnapshotBlock{
				Height:    testCase.SbHeight,
				Timestamp: &currentTime,
			}
			pledgeBeneficialAmount, ok := new(big.Int).SetString(testCase.PledgeBeneficialAmount, 16)
			if !ok {
				t.Fatal("invalid test case data", "filename", testFile.Name(), "caseName", k, "pledgeBeneficialAmount", testCase.PledgeBeneficialAmount)
			}
			code, parseErr := hex.DecodeString(testCase.Code)
			if parseErr != nil {
				t.Fatal("invalid test case code", "filename", testFile.Name(), "caseName", k, "code", testCase.Code, "err", parseErr)
			}

			sendBlock := &ledger.AccountBlock{
				Amount:  big.NewInt(0),
				TokenId: testCase.TokenId,
				Fee:     big.NewInt(0),
			}
			if len(testCase.Fee) > 0 {
				sendBlock.Fee, ok = new(big.Int).SetString(testCase.Fee, 16)
				if !ok {
					t.Fatal("invalid test case data", "filename", testFile.Name(), "caseName", k, "fee", testCase.Fee)
				}
			}
			if len(testCase.Amount) > 0 {
				sendBlock.Amount, ok = new(big.Int).SetString(testCase.Amount, 16)
				if !ok {
					t.Fatal("invalid test case data", "filename", testFile.Name(), "caseName", k, "amount", testCase.Amount)
				}
			}
			if len(testCase.Data) > 0 {
				sendBlock.Data, parseErr = hex.DecodeString(testCase.Data)
				if parseErr != nil {
					t.Fatal("invalid test case data", "filename", testFile.Name(), "caseName", k, "data", testCase.Data)
				}
			}

			var db *mockDB
			var vmBlock *vm_db.VmAccountBlock
			var isRetry bool
			var err error
			if ledger.IsSendBlock(testCase.BlockType) {
				prevBlock := &ledger.AccountBlock{
					BlockType:      ledger.BlockTypeReceive,
					Height:         1,
					Hash:           prevHash,
					PrevHash:       types.ZERO_HASH,
					AccountAddress: testCase.FromAddress,
				}
				sendBlock.PrevHash = prevBlock.Hash
				sendBlock.Height = prevBlock.Height + 1
				sendBlock.BlockType = testCase.BlockType
				sendBlock.AccountAddress = testCase.FromAddress
				sendBlock.ToAddress = testCase.ToAddress
				var newDbErr error
				db, newDbErr = NewMockDB(&testCase.FromAddress, latestSnapshotBlock, prevBlock, quotaInfoList, pledgeBeneficialAmount, testCase.BalanceMap, testCase.Storage, testCase.ContractMetaMap, code)
				if newDbErr != nil {
					t.Fatal("new mock db failed", "filename", testFile.Name(), "caseName", k, "err", newDbErr)
				}
				vm := NewVM(nil)
				vmBlock, isRetry, err = vm.RunV2(db, sendBlock, nil, nil)
			} else if ledger.IsReceiveBlock(testCase.BlockType) {
				sendBlock.BlockType = testCase.SendBlockType
				sendBlock.AccountAddress = testCase.FromAddress
				sendBlock.ToAddress = testCase.ToAddress
				var prevBlock, receiveBlock *ledger.AccountBlock
				if testCase.SendBlockType == ledger.BlockTypeSendCreate {
					receiveBlock = &ledger.AccountBlock{
						BlockType:      testCase.BlockType,
						PrevHash:       types.Hash{},
						Height:         1,
						AccountAddress: testCase.ToAddress,
					}
				} else {
					prevBlock := &ledger.AccountBlock{
						BlockType:      ledger.BlockTypeReceive,
						Height:         1,
						Hash:           prevHash,
						PrevHash:       types.ZERO_HASH,
						AccountAddress: testCase.ToAddress,
					}
					receiveBlock = &ledger.AccountBlock{
						BlockType:      testCase.BlockType,
						PrevHash:       prevBlock.Hash,
						Height:         prevBlock.Height + 1,
						AccountAddress: testCase.ToAddress,
					}
				}
				var newDbErr error
				db, newDbErr = NewMockDB(&testCase.ToAddress, latestSnapshotBlock, prevBlock, quotaInfoList, pledgeBeneficialAmount, testCase.BalanceMap, testCase.Storage, testCase.ContractMetaMap, code)
				if newDbErr != nil {
					t.Fatal("new mock db failed", "filename", testFile.Name(), "caseName", k, "err", newDbErr)
				}
				vm := NewVM(nil)
				vmBlock, isRetry, err = vm.RunV2(db, receiveBlock, sendBlock, nil)
			} else {
				t.Fatal("invalid test case block type", "filename", testFile.Name(), "caseName", k, "blockType", testCase.BlockType)
			}
			if !errorEquals(testCase.Err, err) {
				t.Fatal("invalid test case run result, err", "filename", testFile.Name(), "caseName", k, "expected", testCase.Err, "got", err)
			} else if testCase.IsRetry != isRetry {
				t.Fatal("invalid test case run result, isRetry", "filename", testFile.Name(), "caseName", k, "expected", testCase.IsRetry, "got", isRetry)
			}
			if testCase.Success {
				if vmBlock == nil {
					t.Fatal("invalid test case run result, vmBlock", "filename", testFile.Name(), "caseName", k, "expected", "exist", "got", "nil")
				} else if testCase.BlockType != vmBlock.AccountBlock.BlockType {
					t.Fatal("invalid test case run result, blockType", "filename", testFile.Name(), "caseName", k, "expected", testCase.BlockType, "got", vmBlock.AccountBlock.BlockType)
				} else if testCase.Quota != vmBlock.AccountBlock.Quota {
					t.Fatal("invalid test case run result, quota", "filename", testFile.Name(), "caseName", k, "expected", testCase.Quota, "got", vmBlock.AccountBlock.Quota)
				} else if testCase.QuotaUsed != vmBlock.AccountBlock.QuotaUsed {
					t.Fatal("invalid test case run result, quotaUsed", "filename", testFile.Name(), "caseName", k, "expected", testCase.QuotaUsed, "got", vmBlock.AccountBlock.QuotaUsed)
				} else if checkBalanceResult := checkBalanceMap(testCase.BalanceMap, db.balanceMap); len(checkBalanceResult) > 0 {
					t.Fatal("invalid test case run result, balanceMap", "filename", testFile.Name(), "caseName", k, checkBalanceResult)
				} else if checkStorageResult := checkStorageMap(testCase.Storage, db.storageMap); len(checkStorageResult) > 0 {
					t.Fatal("invalid test case run result, storageMap", "filename", testFile.Name(), "caseName", k, checkStorageResult)
				} else if checkSendBlockListResult := checkSendBlockList(testCase.SendBlockList, vmBlock.AccountBlock.SendBlockList); len(checkSendBlockListResult) > 0 {
					t.Fatal("invalid test case run result, sendBlockList", "filename", testFile.Name(), "caseName", k, checkSendBlockListResult)
				} else if checkLogListResult := checkLogList(testCase.LogList, db.logList); len(checkLogListResult) > 0 {
					t.Fatal("invalid test case run result, logList", "filename", testFile.Name(), "caseName", k, checkLogListResult)
				} else if expected := db.GetLogListHash(); expected != vmBlock.AccountBlock.LogHash {
					t.Fatal("invalid test case run result, logHash", "filename", testFile.Name(), "caseName", k, "expected", expected, "got", vmBlock.AccountBlock.LogHash)
				}
				// TODO check data
			} else if vmBlock != nil {
				t.Fatal("invalid test case run result, vmBlock", "filename", testFile.Name(), "caseName", k, "expected", "nil", "got", vmBlock.AccountBlock)
			}
		}
	}
}

func commonQuotaInfoList() []types.QuotaInfo {
	quotaInfoList := make([]types.QuotaInfo, 0, 75)
	for i := 0; i < 75; i++ {
		quotaInfoList = append(quotaInfoList, types.QuotaInfo{BlockCount: 0, QuotaTotal: 0, QuotaUsedTotal: 0})
	}
	return quotaInfoList
}

func errorEquals(expected string, got error) bool {
	if (len(expected) == 0 && got == nil) || (len(expected) > 0 && got != nil && expected == got.Error()) {
		return true
	}
	return false
}

func checkBalanceMap(expected map[types.TokenTypeId]string, got map[types.TokenTypeId]*big.Int) string {
	gotCount := 0
	for _, v := range got {
		if v.Sign() > 0 {
			gotCount = gotCount + 1
		}
	}
	expectedCount := len(expected)
	if expectedCount != gotCount {
		return "balanceMap len, expected " + strconv.Itoa(expectedCount) + ", got " + strconv.Itoa(gotCount)
	}
	for k, v := range got {
		if v.Sign() == 0 {
			continue
		}
		expectedV, ok := new(big.Int).SetString(expected[k], 16)
		if !ok {
			return "balanceMap amount, " + expected[k]
		}
		if v.Cmp(expectedV) != 0 {
			return k.String() + " token balance, expect " + expectedV.String() + ", got " + v.String()
		}
	}
	return ""
}

func checkStorageMap(expected, got map[string]string) string {
	gotCount := 0
	for _, v := range got {
		if len(v) > 0 {
			gotCount = gotCount + 1
		}
	}
	expectedCount := len(expected)
	if expectedCount != gotCount {
		return "storageMap len, expected " + strconv.Itoa(expectedCount) + ", got " + strconv.Itoa(gotCount)
	}
	for k, v := range got {
		if len(v) == 0 {
			continue
		}
		if expectedV, ok := expected[k]; !ok || expectedV != v {
			return k + " storage, expect " + expectedV + ", got " + v
		}
	}
	return ""
}
