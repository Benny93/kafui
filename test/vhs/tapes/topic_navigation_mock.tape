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

# Clear any existing output
Type "clear"
Enter
Sleep 500ms

# Start Kafui with mock data for reliable testing
Type "go run main.go --mock"
Enter

# Wait for application to initialize
Sleep 4s

# Main page should show resource list
# Navigate to Topics
Type "t"
Sleep 1s

# Down to Topics, Enter to select
Down
Down
Enter

# Wait for topics list to load
Sleep 2s

# Navigate to test.users topic
Type "jjj"
Sleep 500ms

# Use arrow keys as fallback
Down
Down
Down
Sleep 500ms

# Press Enter to open the topic
Enter

# Wait for topic page to load
Sleep 3s

# Topic page should show messages table
# Navigate through messages

# Navigate down through messages
Type "jjjjj"
Sleep 1s

# Navigate up
Type "kkk"
Sleep 1s

# Go to bottom
Type "G"
Sleep 1s

# Go to top
Type "gg"
Sleep 1s

# Refresh messages
Type "r"
Sleep 2s

# Press Escape to go back
Escape
Sleep 2s

# Press Escape again to return to main page
Escape
Sleep 2s

# Quit application
Type "q"
Sleep 2s

# Test completed
Sleep 1s
