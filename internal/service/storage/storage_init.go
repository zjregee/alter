package storage

type initStorageFunc func()

var initStorageFuncs = []initStorageFunc{
	initWorkspaceInfos,
	initFeedItems,
}

func initStorage() {
	for _, f := range initStorageFuncs {
		f()
	}
}
