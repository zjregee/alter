package storage

type initStorageFunc func()

var initStorageFuncs = []initStorageFunc{
	initWorkspaceInfos,
}

func initStorage() {
	for _, f := range initStorageFuncs {
		f()
	}
}
