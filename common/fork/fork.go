package fork

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/config"
)

var forkPoints config.ForkPoints

type ForkPointItem struct {
	config.ForkPoint
	ForkName string
}

type ForkPointList []*ForkPointItem
type ForkPointMap map[string]*ForkPointItem

var forkPointList ForkPointList
var forkPointMap ForkPointMap

func (a ForkPointList) Len() int           { return len(a) }
func (a ForkPointList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ForkPointList) Less(i, j int) bool { return a[i].Height < a[j].Height }

func IsInitForkPoint() bool {
	return forkPointMap != nil
}
func SetForkPoints(points *config.ForkPoints) {
	if points != nil {
		forkPoints = *points

		t := reflect.TypeOf(forkPoints)
		v := reflect.ValueOf(forkPoints)
		forkPointMap = make(ForkPointMap)

		for k := 0; k < t.NumField(); k++ {
			forkPoint := v.Field(k).Interface().(*config.ForkPoint)

			forkName := t.Field(k).Name
			forkPointItem := &ForkPointItem{
				ForkPoint: *forkPoint,
				ForkName:  forkName,
			}

			// set fork point list
			forkPointList = append(forkPointList, forkPointItem)
			// set fork point map
			forkPointMap[forkName] = forkPointItem
		}

		sort.Sort(forkPointList)
	}
}

func CheckForkPoints(points config.ForkPoints) error {
	t := reflect.TypeOf(points)
	v := reflect.ValueOf(points)

	for k := 0; k < t.NumField(); k++ {
		forkPoint := v.Field(k).Interface().(*config.ForkPoint)

		if forkPoint == nil {
			return errors.New(fmt.Sprintf("The fork point %s can't be nil. the `ForkPoints` config in genesis.json is not correct, "+
				"you can remove the `ForkPoints` key in genesis.json then use the default config of `ForkPoints`", t.Field(k).Name))
		}

		if forkPoint.Height <= 0 {
			return errors.New(fmt.Sprintf("The height of fork point %s is 0. "+
				"the `ForkPoints` config in genesis.json is not correct, you can remove the `ForkPoints` key in genesis.json then use the default config of `ForkPoints`", t.Field(k).Name))
		}

		if forkPoint.Version <= 0 {
			return errors.New(fmt.Sprintf("The version of fork point %s is 0. "+
				"the `ForkPoints` config in genesis.json is not correct, you can remove the `ForkPoints` key in genesis.json then use the default config of `ForkPoints`", t.Field(k).Name))
		}

	}

	return nil
}

func GetLeafForkPoint() *ForkPointItem {
	leafForkPoint, ok := forkPointMap["LeafFork"]
	if !ok {
		panic("check leaf fork failed. LeafFork is not existed.")
	}

	return leafForkPoint
}

func IsActiveForkPoint(snapshotHeight uint64) bool {
	// assume that fork point list is sorted by height asc
	for i := len(forkPointList) - 1; i >= 0; i-- {
		forkPoint := forkPointList[i]
		if forkPoint.Height == snapshotHeight {
			return IsForkActive(*forkPoint)
		}

		if forkPoint.Height < snapshotHeight {
			break
		}
	}

	return false
}

func GetForkPoint(snapshotHeight uint64) *ForkPointItem {
	// assume that fork point list is sorted by height asc
	for i := len(forkPointList) - 1; i >= 0; i-- {
		forkPoint := forkPointList[i]
		if forkPoint.Height == snapshotHeight {
			return forkPoint
		}

		if forkPoint.Height < snapshotHeight {
			break
		}
	}

	return nil
}

func GetForkPoints() config.ForkPoints {
	return forkPoints
}

func GetForkPointList() ForkPointList {
	return forkPointList
}

func GetForkPointMap() ForkPointMap {
	return forkPointMap
}

func GetActiveForkPointList() ForkPointList {
	activeForkPointList := make(ForkPointList, 0, len(forkPointList))
	for _, forkPoint := range forkPointList {
		if IsForkActive(*forkPoint) {
			activeForkPointList = append(activeForkPointList, forkPoint)
		}
	}

	return activeForkPointList
}

func GetRecentActiveFork(blockHeight uint64) *ForkPointItem {
	for i := len(forkPointList) - 1; i >= 0; i-- {
		item := forkPointList[i]
		if item.Height <= blockHeight && IsForkActive(*item) {
			return item
		}
	}
	return nil
}

func GetLastForkPointVersion() uint32 {
	if len(forkPointList) == 0 {
		return 0
	}
	return forkPointList[forkPointList.Len()-1].Version
}

func IsForkActive(point ForkPointItem) bool {
	// TODO suppose all point is active.
	return true
	//return activeChecker.IsForkActive(point)
}
