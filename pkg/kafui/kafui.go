package kafui

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/kafds"
	"github.com/Benny93/kafui/pkg/datasource/mock"
)

func Init(cfgOption string, useMock bool) {

	fmt.Println("Init...")
	var dataSource api.KafkaDataSource

	dataSource = mock.KafkaDataSourceMock{}
	if !useMock {
		dataSource = &kafds.KafkaDataSourceKaf{}
	}
	dataSource.Init(cfgOption)
	OpenUI(dataSource)
}
