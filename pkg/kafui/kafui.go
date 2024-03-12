package kafui

import (
	"com/emptystate/kafui/pkg/api"
	"com/emptystate/kafui/pkg/datasource/kafds"
	"com/emptystate/kafui/pkg/datasource/mock"
	"fmt"
)

func Init(useMock bool) {

	fmt.Printf("Init...")
	var dataSource api.KafkaDataSource

	dataSource = mock.KafkaDataSourceMock{}
	if !useMock {
		dataSource = kafds.KafkaDataSourceKaf{}
	}
	dataSource.Init()
	OpenUI(dataSource)
}
