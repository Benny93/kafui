package kafui

type Resource interface {
	StartFetchingData()
	StopFetching()
	GetName() string
}
