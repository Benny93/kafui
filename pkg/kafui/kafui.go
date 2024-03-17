package kafui

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/kafds"
	"github.com/Benny93/kafui/pkg/datasource/mock"
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
