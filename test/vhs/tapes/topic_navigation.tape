# VHS Integration Test for Kafui Topic Navigation
# Uses a large terminal for better visibility

# Set up a 1200x600 terminal with 46px font
Set FontSize 12
Set Width 1200
Set Height 600
Set Padding 2
Set Framerate 60
Set Theme "Dracula"
Set TypingSpeed 50ms
Set LoopOffset 20

# Start the test
Type "go run main.go --mock"
Enter

# Wait for the application to load
Sleep 3s

# The main page should show resource list
# Wait for initial data load
Sleep 2s

# Navigate down to topics
Down
Down

# Press Enter to select Topics resource type
Enter

# Wait for topics list to load
Sleep 1s

# Navigate to test.users topic
Down
Down
Down

# Press Enter to open test.users topic
Enter

# Wait for topic page to load with messages
Sleep 2s

# Navigate down through messages
Down
Down
Down
Down
Down

# Navigate up through messages
Up
Up
Up

# Navigate to bottom of message list
End

# Wait a moment
Sleep 500ms

# Navigate to top
Home

# Wait a moment
Sleep 500ms

# Page down through messages
PageDown

# Wait
Sleep 500ms

# Page up back to top
PageUp

# Wait
Sleep 500ms

# Press Escape to go back to topics list
Escape

# Wait for navigation back
Sleep 1s

# Press Escape again to go back to main page
Escape

# Wait for navigation back
Sleep 1s

# Quit the application
Type "q"

# Wait for application to close
Sleep 1s

# Verify we're back at the terminal prompt
Sleep 500ms
