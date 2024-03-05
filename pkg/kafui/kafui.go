package kafui

import (
	"com/emptystate/kafui/pkg/datasource/kafds"
	"com/emptystate/kafui/pkg/datasource/mock"
	"fmt"
)

func Init() {

	fmt.Printf("Init...")
	useMock := false
	var dataSource KafkaDataSource

	dataSource = mock.KafkaDataSourceMock{}
	if !useMock {
		dataSource = kafds.KafkaDataSourceKaf{}
	}

	OpenUI(dataSource)
}
