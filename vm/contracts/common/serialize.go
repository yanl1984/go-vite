package common

import "github.com/vitelabs/go-vite/vm_db"

type Serializable interface {
	Serialize() ([]byte, error)
	DeSerialize([]byte) error
}

func DeserializeFromDb(db vm_db.VmDb, key []byte, serializable Serializable) bool {
	if data := GetValueFromDb(db, key); len(data) > 0 {
		if err := serializable.DeSerialize(data); err != nil {
			panic(err)
		}
		return true
	} else {
		return false
	}
}

func SerializeToDb(db vm_db.VmDb, key []byte, serializable Serializable) {
	if data, err := serializable.Serialize(); err != nil {
		panic(err)
	} else {
		SetValueToDb(db, key, data)
	}
}


func GetValueFromDb(db vm_db.VmDb, key []byte) []byte {
	if data, err := db.GetValue(key); err != nil {
		panic(err)
	} else {
		return data
	}
}

func SetValueToDb(db vm_db.VmDb, key, value []byte) {
	if err := db.SetValue(key, value); err != nil {
		panic(err)
	}
}