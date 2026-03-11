package gui

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"shutdown_automan/config"
	"shutdown_automan/service"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type GUI struct {
	mainWindow *walk.MainWindow
	ni         *walk.NotifyIcon
	cfg        *config.Config

	serviceCtx    context.Context
	serviceCancel context.CancelFunc
	serviceMu     sync.Mutex

	startAction *walk.Action
	stopAction  *walk.Action

	updateMenuTexts func() // Function to update menu text dynamicall
}

func NewGUI(cfg *config.Config) *GUI {
	return &GUI{
		cfg: cfg,
	}
}

func (g *GUI) Run() {
	mw, err := walk.NewMainWindow()
	if err != nil {
		walk.MsgBox(nil, "Fatal Error", "Failed to create main window: "+err.Error(), walk.MsgBoxOK|walk.MsgBoxIconError)
		return
	}
	g.mainWindow = mw

	// Try to load icon from file "app.ico"
	icon, err := walk.NewIconFromFile("app.ico")
	if err == nil {
		mw.SetIcon(icon)
	} else {
		// Fallback or just ignore
	}

	// Icon for Tray - Retry loop for auto-start scenarios where Shell isn't ready
	var ni *walk.NotifyIcon
	for i := 0; i < 10; i++ {
		ni, err = walk.NewNotifyIcon(mw)
		if err == nil {
			break
		}
		log.Printf("Failed to create NotifyIcon (attempt %d/10): %v", i+1, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		walk.MsgBox(mw, "Fatal Error", "Failed to create Notify Icon after multiple attempts: "+err.Error(), walk.MsgBoxOK|walk.MsgBoxIconError)
		return
	}
	g.ni = ni

	if icon != nil {
		ni.SetIcon(icon)
	} else {
		ni.SetIcon(walk.IconInformation())
	}

	// Helper for translation
	t := func(key string) string {
		return tr(g.cfg, key)
	}

	ni.SetToolTip(t("Remote Restart Service"))

	// Prevent MainWindow from closing (which would exit the app)
	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		*canceled = true
		mw.SetVisible(false)
	})

	// Actions
	g.startAction = walk.NewAction()
	g.startAction.Triggered().Attach(g.startService)

	g.stopAction = walk.NewAction()
	g.stopAction.SetEnabled(false)
	g.stopAction.Triggered().Attach(g.stopService)

	settingsAction := walk.NewAction()
	settingsAction.Triggered().Attach(g.openSettings)

	// Language Actions
	// We need langAction to be accessible for text updates, but it creates via AddMenu
	var langAction *walk.Action

	zhAction := walk.NewAction()
	zhAction.SetText("中文")
	zhAction.Triggered().Attach(func() {
		newCfg := g.cfg.Get()
		newCfg.Language = "zh"
		g.cfg.Update(newCfg)
		g.cfg.Save()
		g.updateMenuTexts()
	})

	enAction := walk.NewAction()
	enAction.SetText("English")
	enAction.Triggered().Attach(func() {
		newCfg := g.cfg.Get()
		newCfg.Language = "en"
		g.cfg.Update(newCfg)
		g.cfg.Save()
		g.updateMenuTexts()
	})

	autoStartAction := walk.NewAction()
	autoStartAction.SetCheckable(true)
	if g.checkAutoStart() {
		autoStartAction.SetChecked(true)
	}
	autoStartAction.Triggered().Attach(func() {
		// New state is already set based on toggle
		newState := autoStartAction.Checked()
		if err := g.toggleAutoStart(newState); err != nil {
			walk.MsgBox(mw, t("Error"), "Failed to update registry: "+err.Error(), walk.MsgBoxOK|walk.MsgBoxIconError)
			autoStartAction.SetChecked(!newState) // Revert
		}
	})

	exitAction := walk.NewAction()
	exitAction.Triggered().Attach(func() {
		g.stopService()
		g.ni.Dispose()
		g.mainWindow.Dispose() // Explicit dispose to allow mw.Run() to exit
		walk.App().Exit(0)
	})

	// Menu
	if err := ni.ContextMenu().Actions().Add(g.startAction); err != nil {
		log.Fatal(err)
	}
	if err := ni.ContextMenu().Actions().Add(g.stopAction); err != nil {
		log.Fatal(err)
	}
	ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())
	ni.ContextMenu().Actions().Add(settingsAction)

	// Language Submenu
	ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())

	langSubMenu, _ := walk.NewMenu()
	langSubMenu.Actions().Add(zhAction)
	langSubMenu.Actions().Add(enAction)

	// AddMenu returns the Action associated with the menu
	langAction, _ = ni.ContextMenu().Actions().AddMenu(langSubMenu)

	ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())
	ni.ContextMenu().Actions().Add(autoStartAction)

	ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())
	ni.ContextMenu().Actions().Add(exitAction)

	// Initial text update
	g.updateMenuTexts = func() {
		g.startAction.SetText(t("Start Service"))
		g.stopAction.SetText(t("Stop Service"))
		settingsAction.SetText(t("Settings"))
		langAction.SetText(t("Language"))
		autoStartAction.SetText(t("Run on Startup"))
		exitAction.SetText(t("Exit"))
		ni.SetToolTip(t("Remote Restart Service"))
	}
	g.updateMenuTexts()

	// Events
	ni.MouseUp().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			g.openSettings()
		}
	})

	ni.SetVisible(true)

	// Auto-start Service
	g.startService()

	// Run Loop
	mw.Run()
}

// updateMenuText verifies and updates texts (assigned dynamically)
func (g *GUI) updateMenuText() {
	// Logic moved inside Run closure for simplicity with Action variables,
	// or we can refactor.
	// For now, let's keep the closure in Run and just define the struct field if needed,
	// BUT since Run is one massive function, I can't easily call it from `Action` triggered
	// unless `updateMenuText` is a member of `GUI`... wait.
	//
	// To make this clean, I'll add `updateMenuText` as a field to GUI struct which is a function.
}

func (g *GUI) startService() {
	t := func(key string) string { return tr(g.cfg, key) }

	g.serviceMu.Lock()
	defer g.serviceMu.Unlock()

	if g.serviceCtx != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	g.serviceCtx = ctx
	g.serviceCancel = cancel

	go func() {
		if err := service.StartHTTPServer(ctx, g.cfg); err != nil {
			log.Printf("HTTP Server error: %v", err)
		}
	}()

	go service.StartMonitor(ctx, g.cfg)

	g.startAction.SetEnabled(false)
	g.stopAction.SetEnabled(true)
	g.ni.ShowCustom(t("Service Started"), t("Remote Restart Service")+" "+t("SubjectStarted"), nil) // Simplified msg
}

func (g *GUI) stopService() {
	g.serviceMu.Lock()
	defer g.serviceMu.Unlock()

	if g.serviceCtx == nil {
		return
	}

	g.serviceCancel()
	g.serviceCtx = nil // Reset
	g.serviceCancel = nil

	// t := func(key string) string { return tr(g.cfg, key) }

	g.startAction.SetEnabled(true)
	g.stopAction.SetEnabled(false)
	// g.ni.ShowCustom(t("Service Stopped"), ... ) // Skip for brevity or add if needed
}

func (g *GUI) restartService() {
	g.serviceMu.Lock()
	running := g.serviceCtx != nil
	g.serviceMu.Unlock()

	if !running {
		return
	}

	go func() {
		g.mainWindow.Synchronize(func() {
			g.stopService()
		})
		time.Sleep(1 * time.Second)
		g.mainWindow.Synchronize(func() {
			g.startService()
		})
	}()
}

func (g *GUI) openSettings() {
	t := func(key string) string { return tr(g.cfg, key) }

	var dlg *walk.Dialog
	var portLE *walk.LineEdit
	var monitorChk *walk.CheckBox
	var intervalNE *walk.NumberEdit
	var acceptPB, cancelPB *walk.PushButton
	var tv *walk.TableView
	var secretKeyLE *walk.LineEdit
	var urlLE *walk.LineEdit
	var copyLinkPB *walk.PushButton

	currentCfg := g.cfg.Get()

	// 1. Prepare Data Model for Process List
	processList := make([]config.ProcessItem, len(currentCfg.ProcessList))
	copy(processList, currentCfg.ProcessList)
	model := NewProcessModel(processList)

	// Simple Data Binding
	port := currentCfg.Port
	secretKey := currentCfg.SecretKey
	monitorEnabled := currentCfg.MonitorEnabled
	interval := float64(currentCfg.MonitorInterval)

	// Helper to update URL display
	updateUrl := func() {
		if urlLE == nil {
			return
		}
		p := portLE.Text()
		if p == "" {
			p = "8080"
		}
		k := secretKeyLE.Text()

		u := fmt.Sprintf("http://localhost:%s/restart", p)
		if k != "" {
			u += fmt.Sprintf("?key=%s", k)
		}
		urlLE.SetText(u)
	}

	// Layout
	err := Dialog{
		AssignTo: &dlg,
		Title:    t("Settings"),
		MinSize:  Size{500, 500},
		Layout:   VBox{},
		Children: []Widget{
			// Group 1: General Settings
			GroupBox{
				Title:  t("General Settings"),
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{Text: t("HTTP Port:")},
					LineEdit{
						AssignTo:      &portLE,
						Text:          port,
						OnTextChanged: updateUrl,
					},

					Label{Text: t("Secret Key (Optional):")},
					LineEdit{
						AssignTo:      &secretKeyLE,
						Text:          secretKey,
						OnTextChanged: updateUrl,
					},

					Label{Text: t("Enable Monitor:")},
					CheckBox{AssignTo: &monitorChk, Checked: monitorEnabled},

					Label{Text: t("Check Interval (s):")},
					NumberEdit{AssignTo: &intervalNE, Value: interval, Decimals: 0},
				},
			},

			// Group 2: Response URL
			GroupBox{
				Title:  t("Remote Restart Link"),
				Layout: Grid{Columns: 3},
				Children: []Widget{
					Label{Text: t("URL:"), ColumnSpan: 1},
					LineEdit{
						AssignTo:   &urlLE,
						ReadOnly:   true,
						ColumnSpan: 2,
					},

					// Row 2: Copy Button
					HSpacer{}, // Empty col 1
					PushButton{
						AssignTo: &copyLinkPB,
						Text:     t("Copy Link"),
						OnClicked: func() {
							if walk.Clipboard().SetText(urlLE.Text()) != nil {
								walk.MsgBox(dlg, t("Error"), "Failed to copy: clipboard error", walk.MsgBoxOK|walk.MsgBoxIconError)
							} else {
								walk.MsgBox(dlg, t("Success"), t("Link copied to clipboard!"), walk.MsgBoxOK|walk.MsgBoxIconInformation)
							}
						},
					},
					HSpacer{}, // Empty col 3
				},
			},

			// Group 2: Process List
			GroupBox{
				Title:  t("Process List (Execution Order)"),
				Layout: VBox{},
				Children: []Widget{
					TableView{
						AssignTo:         &tv,
						AlternatingRowBG: true,
						Columns: []TableViewColumn{
							{Title: t("Process Name"), Width: 180},
							{Title: t("Delay (s)"), Width: 60},
							{Title: t("Process Status"), Width: 120},
						},
						Model:      model,
						CellStyler: NewProcessStyler(model),
					},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							PushButton{
								Text: t("Add..."),
								OnClicked: func() {
									if newItem, ok := g.runProcessDialog(nil, nil); ok {
										model.items = append(model.items, &ProcessRow{ProcessItem: newItem, Status: t("Checking...")})
										model.PublishRowsReset()
									}
								},
							},
							PushButton{
								Text: t("Edit..."),
								OnClicked: func() {
									idx := tv.CurrentIndex()
									if idx < 0 || idx >= len(model.items) {
										return
									}
									if newItem, ok := g.runProcessDialog(nil, &model.items[idx].ProcessItem); ok {
										model.items[idx].ProcessItem = newItem
										model.PublishRowsReset()
									}
								},
							},
							PushButton{
								Text: t("Remove"),
								OnClicked: func() {
									idx := tv.CurrentIndex()
									if idx < 0 || idx >= len(model.items) {
										return
									}
									// Delete item
									model.items = append(model.items[:idx], model.items[idx+1:]...)
									model.PublishRowsReset()
								},
							},
						},
					},
					Label{Text: t("Note: Processes will be terminated in the order listed above.")},
				},
			},

			// Buttons
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &acceptPB,
						Text:     t("Save Configuration"),
						OnClicked: func() {
							// Update Config object
							newCfg := g.cfg.Get()
							newCfg.Port = portLE.Text()
							newCfg.SecretKey = secretKeyLE.Text() // Fix: Save Secret Key
							newCfg.MonitorEnabled = monitorChk.Checked()
							newCfg.MonitorInterval = int(intervalNE.Value())

							// Important: Save the modified process list
							// Convert []*ProcessRow back to []config.ProcessItem
							newItems := make([]config.ProcessItem, len(model.items))
							for i, row := range model.items {
								newItems[i] = row.ProcessItem
							}
							newCfg.ProcessList = newItems

							g.cfg.Update(newCfg)

							if err := g.cfg.Save(); err != nil {
								walk.MsgBox(dlg, t("Error"), "Failed to save config: "+err.Error(), walk.MsgBoxOK|walk.MsgBoxIconError)
							} else {
								dlg.Accept()
								// Reload service if configuration changed
								g.restartService()
							}
						},
					},
					PushButton{
						AssignTo:  &cancelPB,
						Text:      t("Cancel"),
						OnClicked: func() { dlg.Cancel() },
					},
				},
			},
		},
	}.Create(nil) // Changed from g.mainWindow to nil

	if err != nil {
		log.Println("Failed to create settings dialog:", err)
		return
	}

	// Initialize URL field
	updateUrl()

	// Start background status checker
	ctx, cancel := context.WithCancel(context.Background())
	// Ensure we cancel context when dialog closes to stop ticker
	// We can hook into dlg.Closing? But dlg.Run() blocks.
	// So we can defer cancel() after dlg.Run() returns.

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		// Initial check
		g.checkProcessStatuses(model)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				g.checkProcessStatuses(model)
			}
		}
	}()

	dlg.Run()
	cancel() // Stop checker when dialog closes
}

// Sub-window for adding/editing a process
func (g *GUI) runProcessDialog(owner walk.Form, item *config.ProcessItem) (config.ProcessItem, bool) {
	t := func(key string) string { return tr(g.cfg, key) }

	var dlg *walk.Dialog
	var nameLE *walk.LineEdit
	var delayNE *walk.NumberEdit
	var acceptPB, cancelPB *walk.PushButton

	accepted := false
	resultItem := config.ProcessItem{Delay: 5} // Default defaults

	title := t("Add Process")
	if item != nil {
		title = t("Edit Process")
		resultItem = *item
	}

	err := Dialog{
		AssignTo: &dlg,
		Title:    title,
		MinSize:  Size{300, 150},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{Text: t("Process Name:")},
					LineEdit{AssignTo: &nameLE, Text: resultItem.Name},
					Label{Text: t("Delay (s):")},
					NumberEdit{AssignTo: &delayNE, Value: float64(resultItem.Delay), Decimals: 0},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &acceptPB,
						Text:     t("OK"),
						OnClicked: func() {
							if nameLE.Text() == "" {
								walk.MsgBox(dlg, t("Error"), t("Process name cannot be empty"), walk.MsgBoxOK|walk.MsgBoxIconError)
								return
							}
							accepted = true
							resultItem.Name = nameLE.Text()
							resultItem.Delay = int(delayNE.Value())
							dlg.Accept()
						},
					},
					PushButton{
						AssignTo:  &cancelPB,
						Text:      t("Cancel"),
						OnClicked: func() { dlg.Cancel() },
					},
				},
			},
		},
	}.Create(nil) // Use nil parent to be safe on ARM

	if err != nil {
		log.Printf("Failed to create sub-dialog: %v", err)
		return resultItem, false
	}

	dlg.Run()
	return resultItem, accepted
}

func (g *GUI) checkAutoStart() bool {
	// Use Task Scheduler for more robust auto-start (especially if Admin rights are needed)
	// Check if task exists
	cmd := exec.Command("schtasks", "/Query", "/TN", "RemoteRestartService")
	// Hide window logic skipped for cross-platform compatibility
	return cmd.Run() == nil
}

func (g *GUI) toggleAutoStart(enable bool) error {
	taskName := "RemoteRestartService"
	regKey := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	regVal := "RemoteRestartService"

	if enable {
		exePath, err := os.Executable()
		if err != nil {
			return err
		}

		// Create Scheduled Task with highest privileges
		// /SC ONLOGON: Trigger at user logon
		// /RL HIGHEST: Run with highest privileges (Bypass UAC for admin apps)
		// /F: Force create
		// Note: We wrap exePath in quotes to handle spaces
		cmd := exec.Command("schtasks", "/Create", "/TN", taskName, "/TR", "\""+exePath+"\"", "/SC", "ONLOGON", "/RL", "HIGHEST", "/F")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create scheduled task: %v", err)
		}

		// Clean up old registry key if exists (hybrid approach transition)
		exec.Command("reg", "delete", regKey, "/v", regVal, "/f").Run()

		return nil
	} else {
		// Remove Scheduled Task
		cmd := exec.Command("schtasks", "/Delete", "/TN", taskName, "/F")
		err := cmd.Run()

		// Also ensure registry key is gone
		exec.Command("reg", "delete", regKey, "/v", regVal, "/f").Run()

		return err
	}
}

// --- Table Model ---

// --- Table Model ---

type ProcessRow struct {
	config.ProcessItem
	Status string
}

type ProcessModel struct {
	walk.TableModelBase
	items []*ProcessRow
}

func NewProcessModel(items []config.ProcessItem) *ProcessModel {
	rows := make([]*ProcessRow, len(items))
	// t := func(key string) string { return ... } // Use placeholder and update later
	for i, item := range items {
		rows[i] = &ProcessRow{ProcessItem: item, Status: "Checking..."}
	}
	return &ProcessModel{items: rows}
}

func (m *ProcessModel) RowCount() int {
	return len(m.items)
}

func (m *ProcessModel) Value(row, col int) interface{} {
	if row < 0 || row >= len(m.items) {
		return nil
	}
	item := m.items[row]
	switch col {
	case 0:
		return item.Name
	case 1:
		return item.Delay
	case 2:
		return item.Status
	}
	return nil
}

// ProcessStyler handles cell coloring
type ProcessStyler struct {
	model *ProcessModel
}

func NewProcessStyler(m *ProcessModel) *ProcessStyler {
	return &ProcessStyler{model: m}
}

func (s *ProcessStyler) StyleCell(style *walk.CellStyle) {
	if style.Row() < 0 || style.Row() >= len(s.model.items) {
		return
	}
	if style.Col() == 2 {
		status := s.model.items[style.Row()].Status
		// Simple substring check or hardcoded strings.
		// "Running" (Green), "Not Responding" (Red), "Not Started" (Gray)
		// Since we will set these strings in checkProcessStatuses, we match them here.
		// We use a broader match to handle both English and localized versions if needed,
		// or relies on checkProcessStatuses producing specific keys.

		// Note: Walk colors are 0xBBGGRR format usually, but walk.RGB(r,g,b) handles it.
		if strings.Contains(status, "Running") || strings.Contains(status, "运行") {
			style.TextColor = walk.RGB(0, 150, 0)
		} else if strings.Contains(status, "Not Responding") || strings.Contains(status, "无响应") {
			style.TextColor = walk.RGB(255, 0, 0)
		} else if strings.Contains(status, "Not Started") || strings.Contains(status, "未启动") {
			style.TextColor = walk.RGB(128, 128, 128)
		} else {
			style.TextColor = walk.RGB(0, 0, 0)
		}
	}
}

func (g *GUI) checkProcessStatuses(model *ProcessModel) {
	statusMap, err := service.GetRunningProcesses()
	if err != nil {
		// log.Println("Failed to check process status:", err)
		return
	}

	t := func(key string) string { return tr(g.cfg, key) }

	changed := false
	for _, row := range model.items {
		// Case insensitive status lookup
		lowerName := strings.ToLower(row.Name)
		procInfo, exists := statusMap[lowerName]

		newStatus := ""
		if !exists {
			newStatus = t("Not Started")
		} else {
			stat := procInfo.Status
			if strings.Contains(stat, "Not Responding") { // Localized tasklist output might differ?
				// tasklist status: "Running", "Suspended", "Not Responding", "Unknown"
				// "Unknown" usually means running.
				newStatus = t("Not Responding")
			} else if strings.Contains(stat, "Unknown") || strings.Contains(stat, "Running") {
				newStatus = t("Running")
			} else {
				// Fallback to whatever status is, or assume running
				newStatus = t("Running")
			}
		}

		if row.Status != newStatus {
			row.Status = newStatus
			changed = true
		}
	}

	if changed {
		g.mainWindow.Synchronize(func() {
			model.PublishRowsReset()
		})
	}
}
