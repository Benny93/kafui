package kafui

import (
	"context"
	"testing"
	"time"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/Benny93/kafui/pkg/datasource/mock"
)

// BenchmarkInit tests the performance of the initialization flow
func BenchmarkInit(b *testing.B) {
	tmpConfig := createTempConfigForBench(b)
	defer func() {
		// Clean up after benchmark
		// Note: We can't easily benchmark the full Init() due to UI dependencies
	}()

	b.Run("MockDataSourceInit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			dataSource := mock.KafkaDataSourceMock{}
			dataSource.Init(tmpConfig)
		}
	})
}

// BenchmarkDataSourceOperations tests the performance of core data source operations
func BenchmarkDataSourceOperations(b *testing.B) {
	dataSource := mock.KafkaDataSourceMock{}
	dataSource.Init("")

	b.Run("GetTopics", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := dataSource.GetTopics()
			if err != nil {
				b.Fatalf("GetTopics failed: %v", err)
			}
		}
	})

	b.Run("GetConsumerGroups", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := dataSource.GetConsumerGroups()
			if err != nil {
				b.Fatalf("GetConsumerGroups failed: %v", err)
			}
		}
	})

	b.Run("GetContexts", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := dataSource.GetContexts()
			if err != nil {
				b.Fatalf("GetContexts failed: %v", err)
			}
		}
	})

	b.Run("ContextSwitching", func(b *testing.B) {
		contexts, err := dataSource.GetContexts()
		if err != nil || len(contexts) == 0 {
			b.Skip("No contexts available for benchmarking")
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			contextIndex := i % len(contexts)
			err := dataSource.SetContext(contexts[contextIndex])
			if err != nil {
				b.Fatalf("SetContext failed: %v", err)
			}
		}
	})
}

// BenchmarkMessageConsumption tests the performance of message consumption
func BenchmarkMessageConsumption(b *testing.B) {
	dataSource := mock.KafkaDataSourceMock{}
	dataSource.Init("")

	b.Run("ConsumeMessages", func(b *testing.B) {
		messageCount := 0
		handleMessage := func(msg api.Message) {
			messageCount++
		}

		onError := func(err any) {
			b.Errorf("Unexpected error: %v", err)
		}

		flags := api.DefaultConsumeFlags()
		flags.Tail = int32(b.N)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		b.ResetTimer()
		
		// Start consumption
		go func() {
			err := dataSource.ConsumeTopic(ctx, "test-topic", flags, handleMessage, onError)
			if err != nil && err != context.DeadlineExceeded {
				b.Logf("ConsumeTopic ended with: %v", err)
			}
		}()

		// Wait for consumption to complete or timeout
		startTime := time.Now()
		for messageCount < b.N && time.Since(startTime) < 25*time.Second {
			time.Sleep(10 * time.Millisecond)
		}

		b.StopTimer()
		b.Logf("Consumed %d messages in benchmark", messageCount)
	})
}

// BenchmarkUIComponents tests the performance of UI component creation
func BenchmarkUIComponents(b *testing.B) {
	dataSource := mock.KafkaDataSourceMock{}
	dataSource.Init("")

	b.Run("MainPageCreation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			mainPage := NewMainPage()
			if mainPage == nil {
				b.Fatal("Failed to create MainPage")
			}
		}
	})

	b.Run("PropertyInfoCreation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			CreatePropertyInfo("TestProperty", "TestValue")
		}
	})

	b.Run("RunInfoCreation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			CreateRunInfo("TestRune", "TestInfo")
		}
	})
}

// BenchmarkMemoryUsage tests memory efficiency of core operations
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("DataSourceMemoryUsage", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dataSource := mock.KafkaDataSourceMock{}
			dataSource.Init("")
			
			// Perform typical operations
			dataSource.GetTopics()
			dataSource.GetConsumerGroups()
			dataSource.GetContexts()
		}
	})

	b.Run("UIComponentMemoryUsage", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			mainPage := NewMainPage()
			CreatePropertyInfo("Property", "Value")
			CreateRunInfo("Rune", "Info")
			_ = mainPage // Use the variable to prevent optimization
		}
	})
}

// Helper function for benchmark config creation
func createTempConfigForBench(b *testing.B) string {
	content := `current-cluster: bench
clusters:
- name: bench
  brokers:
  - localhost:9092
`
	// For benchmarks, we'll use a simple in-memory config
	// In a real scenario, you might want to create actual temp files
	return content
}