package myutils

import "github.com/1a1a11a/c2dnPrototype/src/myconst"

func CheckCodingPolicy(objSize int64) (shouldCode bool) {
	shouldCode = false

	if objSize > myconst.CodingObjSizeThreshold {
		shouldCode = true
	}

	return shouldCode
}

func CheckRamAdmitPolicy(ramCacheSize, objSize int64) (cacheInRam bool) {
	cacheInRam = false
	if ramCacheSize != 0 && objSize < myconst.RAMCacheAdmitThreshold && objSize*1024 < ramCacheSize {
		cacheInRam = true
	}
	return cacheInRam
}
