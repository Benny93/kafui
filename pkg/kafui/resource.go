package kafui

import (
	"github.com/Benny93/kafui/pkg/api"
	"github.com/rivo/tview"
)

type Resource interface {
	StartFetchingData()
	UpdateTable(table *tview.Table, dataSource api.KafkaDataSource, search string)
	StopFetching()
	GetName() string
}
