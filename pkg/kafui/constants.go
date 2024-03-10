package kafui

type UIEvent string

const (
	OnModalClose  UIEvent = "ModalClose"
	OnFocusSearch UIEvent = "FocusSearch"
	OnPageChange  UIEvent = "PageChange"
)

type ResouceName []string // array because it can have multiple names

var (
	Context       ResouceName = []string{"context", "ctx", "kafka", "broker"}
	Topic         ResouceName = []string{"topics", "ts"}
	ConsumerGroup ResouceName = []string{"consumergroups", "groups", "consumers", "cgs"}
)
