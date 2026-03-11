package gui

import "shutdown_automan/config"

var translations = map[string]map[string]string{
	"en": {
		"Start Service":          "Start Service",
		"Stop Service":           "Stop Service",
		"Settings":               "Settings",
		"Language":               "Language",
		"Run on Startup":         "Run on Startup",
		"Exit":                   "Exit",
		"Fatal Error":            "Fatal Error",
		"Service Started":        "Service Started",
		"Service Stopped":        "Service Stopped",
		"Remote Restart Service": "Remote Restart Service",

		// Settings Dialog
		"General Settings":               "General Settings",
		"HTTP Port:":                     "HTTP Port:",
		"Secret Key (Optional):":         "Secret Key (Optional):",
		"Enable Monitor:":                "Enable Monitor:",
		"Check Interval (s):":            "Check Interval (s):",
		"Remote Restart Link":            "Remote Restart Link",
		"URL:":                           "URL:",
		"Copy Link":                      "Copy Link",
		"Process List (Execution Order)": "Process List (Execution Order)",
		"Process Name":                   "Process Name",
		"Delay (s)":                      "Delay (s)",
		"Add...":                         "Add...",
		"Edit...":                        "Edit...",
		"Remove":                         "Remove",
		"Note: Processes will be terminated in the order listed above.": "Note: Processes will be terminated in the order listed above.",
		"Save Configuration":        "Save Configuration",
		"Cancel":                    "Cancel",
		"Error":                     "Error",
		"Success":                   "Success",
		"Link copied to clipboard!": "Link copied to clipboard!",

		// Sub Dialog
		"Add Process":                  "Add Process",
		"Edit Process":                 "Edit Process",
		"Process Name:":                "Process Name:",
		"OK":                           "OK",
		"Process name cannot be empty": "Process name cannot be empty",
		"Not Started":                  "Not Started",
		"Running":                      "Running",
		"Not Responding":               "Not Responding",
		"Checking...":                  "Checking...",
		"Process Status":               "Status",
	},
	"zh": {
		"Start Service":          "启动服务",
		"Stop Service":           "停止服务",
		"Settings":               "设置",
		"Language":               "语言 (Language)",
		"Run on Startup":         "开机自动启动",
		"Exit":                   "退出",
		"Fatal Error":            "严重错误",
		"Service Started":        "服务已启动",
		"Service Stopped":        "服务已停止",
		"Remote Restart Service": "远程重启服务",

		// Settings Dialog
		"General Settings":               "常规设置",
		"HTTP Port:":                     "HTTP 端口:",
		"Secret Key (Optional):":         "访问密钥 (可选):",
		"Enable Monitor:":                "启用进程监控:",
		"Check Interval (s):":            "监控间隔 (秒):",
		"Remote Restart Link":            "远程控制链接",
		"URL:":                           "地址:",
		"Copy Link":                      "复制链接",
		"Process List (Execution Order)": "进程列表 (按序执行)",
		"Process Name":                   "进程名",
		"Delay (s)":                      "延迟 (秒)",
		"Add...":                         "添加...",
		"Edit...":                        "编辑...",
		"Remove":                         "删除",
		"Note: Processes will be terminated in the order listed above.": "注意：进程将按照上述列表顺序依次被终止。",
		"Save Configuration":        "保存配置",
		"Cancel":                    "取消",
		"Error":                     "错误",
		"Success":                   "成功",
		"Link copied to clipboard!": "链接已复制到剪贴板！",

		// Sub Dialog
		"Add Process":                  "添加进程",
		"Edit Process":                 "编辑进程",
		"Process Name:":                "进程名:",
		"OK":                           "确定",
		"Process name cannot be empty": "进程名不能为空",
	},
}

func tr(cfg *config.Config, key string) string {
	lang := cfg.Get().Language
	if lang == "" {
		lang = "zh" // Default to Chinese as user requested
	}

	if mapping, ok := translations[lang]; ok {
		if val, ok := mapping[key]; ok {
			return val
		}
	}

	// Fallback to key itself or English if needed, but key itself is usually English source
	return key
}
