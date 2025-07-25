package kafds

import (
	"crypto/sha256"
	"crypto/sha512"
	"testing"
)

// TestSHA256HashGenerator tests the SHA256 hash generator function
func TestSHA256HashGenerator(t *testing.T) {
	hash := SHA256()
	
	if hash == nil {
		t.Fatal("SHA256() returned nil hash")
	}
	
	// Test that it's actually SHA256
	expectedSize := sha256.Size
	hash.Write([]byte("test"))
	result := hash.Sum(nil)
	
	if len(result) != expectedSize {
		t.Errorf("SHA256 hash size = %d, want %d", len(result), expectedSize)
	}
}

// TestSHA512HashGenerator tests the SHA512 hash generator function
func TestSHA512HashGenerator(t *testing.T) {
	hash := SHA512()
	
	if hash == nil {
		t.Fatal("SHA512() returned nil hash")
	}
	
	// Test that it's actually SHA512
	expectedSize := sha512.Size
	hash.Write([]byte("test"))
	result := hash.Sum(nil)
	
	if len(result) != expectedSize {
		t.Errorf("SHA512 hash size = %d, want %d", len(result), expectedSize)
	}
}

// TestXDGSCRAMClient_Begin tests the Begin method
func TestXDGSCRAMClient_Begin(t *testing.T) {
	tests := []struct {
		name        string
		hashGen     func() *XDGSCRAMClient
		userName    string
		password    string
		authzID     string
		expectError bool
	}{
		{
			name: "SHA256 with valid credentials",
			hashGen: func() *XDGSCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA256}
			},
			userName:    "testuser",
			password:    "testpass",
			authzID:     "",
			expectError: false,
		},
		{
			name: "SHA512 with valid credentials",
			hashGen: func() *XDGSCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA512}
			},
			userName:    "testuser",
			password:    "testpass",
			authzID:     "",
			expectError: false,
		},
		{
			name: "SHA256 with empty username",
			hashGen: func() *XDGSCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA256}
			},
			userName:    "",
			password:    "testpass",
			authzID:     "",
			expectError: false, // SCRAM library might allow empty username
		},
		{
			name: "SHA256 with empty password",
			hashGen: func() *XDGSCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA256}
			},
			userName:    "testuser",
			password:    "",
			authzID:     "",
			expectError: false, // SCRAM library might allow empty password
		},
		{
			name: "SHA256 with authzID",
			hashGen: func() *XDGSCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA256}
			},
			userName:    "testuser",
			password:    "testpass",
			authzID:     "authzuser",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.hashGen()
			
			err := client.Begin(tt.userName, tt.password, tt.authzID)
			
			if tt.expectError && err == nil {
				t.Errorf("Begin() expected error, got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Begin() unexpected error: %v", err)
			}
			
			// If no error, verify client state
			if err == nil {
				if client.Client == nil {
					t.Error("Client should be initialized after successful Begin()")
				}
				
				if client.ClientConversation == nil {
					t.Error("ClientConversation should be initialized after successful Begin()")
				}
			}
		})
	}
}

// TestXDGSCRAMClient_Step tests the Step method
func TestXDGSCRAMClient_Step(t *testing.T) {
	client := &XDGSCRAMClient{HashGeneratorFcn: SHA256}
	
	// Initialize client
	err := client.Begin("testuser", "testpass", "")
	if err != nil {
		t.Fatalf("Begin() failed: %v", err)
	}
	
	// Test first step (client-first-message)
	response, err := client.Step("")
	if err != nil {
		t.Errorf("First Step() failed: %v", err)
	}
	
	if response == "" {
		t.Error("First Step() returned empty response")
	}
	
	// The response should contain client-first-message format
	// This is a basic check - in real SCRAM, this would be more complex
	if len(response) < 10 {
		t.Errorf("First Step() response too short: %s", response)
	}
}

// TestXDGSCRAMClient_Done tests the Done method
func TestXDGSCRAMClient_Done(t *testing.T) {
	client := &XDGSCRAMClient{HashGeneratorFcn: SHA256}
	
	// Initialize client
	err := client.Begin("testuser", "testpass", "")
	if err != nil {
		t.Fatalf("Begin() failed: %v", err)
	}
	
	// Initially should not be done
	if client.Done() {
		t.Error("Done() should return false initially")
	}
	
	// After first step, still should not be done
	_, err = client.Step("")
	if err != nil {
		t.Fatalf("Step() failed: %v", err)
	}
	
	if client.Done() {
		t.Error("Done() should return false after first step")
	}
}

// TestXDGSCRAMClient_FullWorkflow tests a complete SCRAM workflow
func TestXDGSCRAMClient_FullWorkflow(t *testing.T) {
	tests := []struct {
		name    string
		hashGen func() *XDGSCRAMClient
	}{
		{
			name: "SHA256 workflow",
			hashGen: func() *XDGSCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA256}
			},
		},
		{
			name: "SHA512 workflow",
			hashGen: func() *XDGSCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA512}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.hashGen()
			
			// Step 1: Begin
			err := client.Begin("testuser", "testpass", "")
			if err != nil {
				t.Fatalf("Begin() failed: %v", err)
			}
			
			// Step 2: First message
			clientFirst, err := client.Step("")
			if err != nil {
				t.Fatalf("First Step() failed: %v", err)
			}
			
			if clientFirst == "" {
				t.Error("Client first message is empty")
			}
			
			// Verify not done yet
			if client.Done() {
				t.Error("Should not be done after first step")
			}
			
			// Note: We can't easily test the full SCRAM handshake without a server
			// but we can verify the client is in the correct state
		})
	}
}

// TestXDGSCRAMClient_MultipleClients tests multiple client instances
func TestXDGSCRAMClient_MultipleClients(t *testing.T) {
	client1 := &XDGSCRAMClient{HashGeneratorFcn: SHA256}
	client2 := &XDGSCRAMClient{HashGeneratorFcn: SHA512}
	
	// Initialize both clients
	err1 := client1.Begin("user1", "pass1", "")
	err2 := client2.Begin("user2", "pass2", "")
	
	if err1 != nil {
		t.Errorf("Client1 Begin() failed: %v", err1)
	}
	
	if err2 != nil {
		t.Errorf("Client2 Begin() failed: %v", err2)
	}
	
	// Get first messages from both
	msg1, err1 := client1.Step("")
	msg2, err2 := client2.Step("")
	
	if err1 != nil {
		t.Errorf("Client1 Step() failed: %v", err1)
	}
	
	if err2 != nil {
		t.Errorf("Client2 Step() failed: %v", err2)
	}
	
	// Messages should be different (different users/passwords)
	if msg1 == msg2 {
		t.Error("Different clients produced identical messages")
	}
	
	// Both should not be done
	if client1.Done() || client2.Done() {
		t.Error("Clients should not be done after first step")
	}
}

// TestXDGSCRAMClient_ErrorHandling tests error scenarios
func TestXDGSCRAMClient_ErrorHandling(t *testing.T) {
	// Test Step without Begin - this will panic due to nil ClientConversation
	// We need to test this with proper error recovery
	client := &XDGSCRAMClient{HashGeneratorFcn: SHA256}
	
	// Test that calling Step without Begin causes expected behavior
	defer func() {
		if r := recover(); r != nil {
			// This is expected - Step() panics when ClientConversation is nil
			t.Logf("Step() without Begin() panicked as expected: %v", r)
		}
	}()
	
	// This will panic, which is the current behavior
	_, err := client.Step("")
	if err == nil {
		t.Error("Step() without Begin() should return error or panic")
	}
}

// Benchmark tests for SCRAM performance
func BenchmarkXDGSCRAMClient_Begin_SHA256(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := &XDGSCRAMClient{HashGeneratorFcn: SHA256}
		err := client.Begin("testuser", "testpass", "")
		if err != nil {
			b.Fatalf("Begin() failed: %v", err)
		}
	}
}

func BenchmarkXDGSCRAMClient_Begin_SHA512(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := &XDGSCRAMClient{HashGeneratorFcn: SHA512}
		err := client.Begin("testuser", "testpass", "")
		if err != nil {
			b.Fatalf("Begin() failed: %v", err)
		}
	}
}

func BenchmarkXDGSCRAMClient_Step_SHA256(b *testing.B) {
	client := &XDGSCRAMClient{HashGeneratorFcn: SHA256}
	err := client.Begin("testuser", "testpass", "")
	if err != nil {
		b.Fatalf("Begin() failed: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset client for each iteration
		client.Begin("testuser", "testpass", "")
		_, err := client.Step("")
		if err != nil {
			b.Fatalf("Step() failed: %v", err)
		}
	}
}

func BenchmarkSHA256HashGenerator(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := SHA256()
		hash.Write([]byte("benchmark test data"))
		_ = hash.Sum(nil)
	}
}

func BenchmarkSHA512HashGenerator(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := SHA512()
		hash.Write([]byte("benchmark test data"))
		_ = hash.Sum(nil)
	}
}