package providers

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// FilesSection provides a files sidebar section
type FilesSection struct {
	files []FileItem
}

type FileItem struct {
	Name     string
	Path     string
	Modified time.Time
	Size     string
	Status   string
}

func NewFilesSection() *FilesSection {
	return &FilesSection{
		files: generateSampleFiles(),
	}
}

func (f *FilesSection) GetTitle() string {
	return "Files"
}

func (f *FilesSection) RenderItems(maxItems, width int) []SidebarItem {
	var items []SidebarItem
	
	count := 0
	for _, file := range f.files {
		if count >= maxItems {
			break
		}
		
		icon := f.getFileStatusIcon(file.Status)
		status := f.getFileStatusType(file.Status)
		
		// Format filename
		name := file.Name
		if len(name) > width-10 { // Reserve space for icon and size
			name = name[:width-13] + "..."
		}
		
		items = append(items, SidebarItem{
			Icon:   icon,
			Text:   name,
			Value:  file.Size,
			Status: status,
		})
		count++
	}
	
	return items
}

func (f *FilesSection) HandleSectionUpdate(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case sectionTickMsg:
		f.updateFiles()
	}
	return nil
}

func (f *FilesSection) InitSection() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return sectionTickMsg(t)
	})
}

func (f *FilesSection) RefreshSection() tea.Cmd {
	f.updateFiles()
	return nil
}

func (f *FilesSection) getFileStatusIcon(status string) string {
	switch status {
	case "modified":
		return "●"
	case "new":
		return "+"
	case "deleted":
		return "×"
	case "clean":
		return "○"
	default:
		return "○"
	}
}

func (f *FilesSection) getFileStatusType(status string) string {
	switch status {
	case "modified":
		return "warning"
	case "new":
		return "success"
	case "deleted":
		return "error"
	case "clean":
		return "muted"
	default:
		return "muted"
	}
}

func (f *FilesSection) updateFiles() {
	now := time.Now()
	// Simulate main.go being recently modified
	if len(f.files) > 0 {
		f.files[0].Modified = now.Add(-time.Duration(now.Second()) * time.Minute)
		f.files[0].Status = "modified"
	}
}

func generateSampleFiles() []FileItem {
	return []FileItem{
		{Name: "main.go", Path: "./main.go", Size: "2.1KB", Status: "modified", Modified: time.Now().Add(-5 * time.Minute)},
		{Name: "app.go", Path: "./ui/app.go", Size: "4.5KB", Status: "modified", Modified: time.Now().Add(-10 * time.Minute)},
		{Name: "sidebar.go", Path: "./ui/components/sidebar.go", Size: "8.2KB", Status: "new", Modified: time.Now().Add(-2 * time.Minute)},
		{Name: "theme.go", Path: "./ui/styles/theme.go", Size: "3.1KB", Status: "clean", Modified: time.Now().Add(-30 * time.Minute)},
		{Name: "README.md", Path: "./README.md", Size: "1.8KB", Status: "clean", Modified: time.Now().Add(-1 * time.Hour)},
		{Name: "go.mod", Path: "./go.mod", Size: "456B", Status: "clean", Modified: time.Now().Add(-2 * time.Hour)},
	}
}

// ServersSection provides a servers sidebar section
type ServersSection struct {
	servers []ServerItem
}

type ServerItem struct {
	Name   string
	Status string
	Load   string
	Uptime string
}

func NewServersSection() *ServersSection {
	return &ServersSection{
		servers: generateSampleServers(),
	}
}

func (s *ServersSection) GetTitle() string {
	return "Servers"
}

func (s *ServersSection) RenderItems(maxItems, width int) []SidebarItem {
	var items []SidebarItem
	
	count := 0
	for _, server := range s.servers {
		if count >= maxItems {
			break
		}
		
		icon := s.getServerStatusIcon(server.Status)
		status := s.getServerStatusType(server.Status)
		
		// Format server name
		name := server.Name
		if len(name) > width-10 {
			name = name[:width-13] + "..."
		}
		
		items = append(items, SidebarItem{
			Icon:   icon,
			Text:   name,
			Value:  server.Load,
			Status: status,
		})
		count++
	}
	
	return items
}

func (s *ServersSection) HandleSectionUpdate(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case sectionTickMsg:
		s.updateServers()
	}
	return nil
}

func (s *ServersSection) InitSection() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return sectionTickMsg(t)
	})
}

func (s *ServersSection) RefreshSection() tea.Cmd {
	s.updateServers()
	return nil
}

func (s *ServersSection) getServerStatusIcon(status string) string {
	switch status {
	case "online":
		return "●"
	case "offline":
		return "○"
	case "warning":
		return "⚠"
	case "error":
		return "×"
	default:
		return "○"
	}
}

func (s *ServersSection) getServerStatusType(status string) string {
	switch status {
	case "online":
		return "success"
	case "offline":
		return "error"
	case "warning":
		return "warning"
	case "error":
		return "error"
	default:
		return "muted"
	}
}

func (s *ServersSection) updateServers() {
	now := time.Now()
	// Update server loads
	for i := range s.servers {
		baseLoad := 30 + (now.Second() % 40)
		s.servers[i].Load = fmt.Sprintf("%d%%", baseLoad+i*10)
		
		if baseLoad+i*10 > 80 {
			s.servers[i].Status = "warning"
		} else {
			s.servers[i].Status = "online"
		}
	}
}

func generateSampleServers() []ServerItem {
	return []ServerItem{
		{Name: "prod-api-01", Status: "online", Load: "45%", Uptime: "23d"},
		{Name: "prod-api-02", Status: "online", Load: "62%", Uptime: "23d"},
		{Name: "staging-web", Status: "online", Load: "12%", Uptime: "5d"},
		{Name: "dev-db", Status: "warning", Load: "89%", Uptime: "12h"},
		{Name: "backup-srv", Status: "offline", Load: "0%", Uptime: "0m"},
	}
}

// StatusSection provides a status sidebar section
type StatusSection struct {
	status []StatusItem
}

type StatusItem struct {
	Name    string
	Status  string
	Message string
	Value   string
}

func NewStatusSection() *StatusSection {
	return &StatusSection{
		status: generateSampleStatus(),
	}
}

func (s *StatusSection) GetTitle() string {
	return "Status"
}

func (s *StatusSection) RenderItems(maxItems, width int) []SidebarItem {
	var items []SidebarItem
	
	count := 0
	for _, item := range s.status {
		if count >= maxItems {
			break
		}
		
		items = append(items, SidebarItem{
			Icon:   item.Value,
			Text:   item.Name,
			Value:  "",
			Status: item.Status,
		})
		count++
	}
	
	return items
}

func (s *StatusSection) HandleSectionUpdate(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case sectionTickMsg:
		s.updateStatus()
	}
	return nil
}

func (s *StatusSection) InitSection() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return sectionTickMsg(t)
	})
}

func (s *StatusSection) RefreshSection() tea.Cmd {
	s.updateStatus()
	return nil
}

func (s *StatusSection) updateStatus() {
	now := time.Now()
	// Update status values
	for i := range s.status {
		if s.status[i].Name == "Health" {
			if now.Second()%30 < 10 {
				s.status[i].Status = "warning"
				s.status[i].Message = "High memory usage"
				s.status[i].Value = "⚠"
			} else {
				s.status[i].Status = "success"
				s.status[i].Message = "All systems normal"
				s.status[i].Value = "✓"
			}
		}
	}
}

func generateSampleStatus() []StatusItem {
	return []StatusItem{
		{Name: "Build", Status: "success", Message: "All tests passed", Value: "✓"},
		{Name: "Deploy", Status: "success", Message: "v1.2.3 deployed", Value: "✓"},
		{Name: "Health", Status: "warning", Message: "High memory usage", Value: "⚠"},
		{Name: "Security", Status: "success", Message: "No vulnerabilities", Value: "✓"},
		{Name: "Backup", Status: "info", Message: "Last: 2h ago", Value: "●"},
	}
}

type sectionTickMsg time.Time