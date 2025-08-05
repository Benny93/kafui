package kafui

import (
	"fmt"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/kafds"
	"github.com/Benny93/kafui/pkg/datasource/mock"
)

// openUIFunc is a variable that holds the OpenUI function, allowing it to be mocked in tests
var openUIFunc = OpenUI

func Init(cfgOption string, useMock bool) {

	fmt.Println("Init...")
	var dataSource api.KafkaDataSource

	dataSource = mock.KafkaDataSourceMock{}
	if !useMock {
		dataSource = kafds.NewKafkaDataSourceKaf()
	}
	dataSource.Init(cfgOption)
	openUIFunc(dataSource)
}
