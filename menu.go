package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"runtime"
	"strings"
	"time"
)

//go:embed resource/built_in_menu.json
var built_in_menu []byte

type PlatForm struct {
	Id        string `json:"id"`
	Label     string `json:"label"`
	Url       string `json:"url"`
	Priority  int    `json:"priority" default:"0"`
	Separator bool   `json:"separator" default:"false"`
	Group     string `json:"group" default:"默认分组"`
}

type Menu struct {
	Id       string     `json:"id"`
	Title    string     `json:"title"`
	Priority int        `json:"priority" default:"0"`
	SubMenu  []PlatForm `json:"menu"`
}

func (app *App) ReadMenu() []PlatForm {
	filePath := ConfigPath("menu.json")

	platforms := []PlatForm{}

	if _, err := os.Stat(filePath); err == nil {
		fmt.Println("File exists")
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Println("Error reading file", err)
		}
		err = json.Unmarshal(content, &platforms)
		if err != nil {
			fmt.Println("Error unmarshalling json", err)
		}

	} else if os.IsNotExist(err) {
		fmt.Println("File does not exist")
		platforms = []PlatForm{
			{
				Id:        "1",
				Label:     "自定义Demo",
				Url:       "https://www.google.com",
				Priority:  0,
				Separator: false,
				Group:     "默认分组",
			},
		}
		content, err := json.Marshal(platforms)
		if err != nil {
			fmt.Println("Error marshalling json", err)
		}
		err = os.WriteFile(filePath, content, 0644)
		if err != nil {
			fmt.Println("Error writing file", err)
		}
		fmt.Println("File created and written")
	} else {
		fmt.Println("Error reading file", err)
	}

	return platforms

}

func (app *App) EditMenu(platorms []PlatForm) {
	filePath := ConfigPath("menu.json")
	content, err := json.Marshal(platorms)
	if err != nil {
		fmt.Println("Error marshalling json", err)
	}
	err = os.WriteFile(filePath, content, 0644)
	if err != nil {
		fmt.Println("Error writing file", err)
	}
	fmt.Println("Updated file")
	app.updateCustomMenu()
}

func (app *App) updateCustomMenu() {
	_menu := app.initMenu()
	wruntime.MenuSetApplicationMenu(app.ctx, _menu)
	wruntime.MenuUpdateApplicationMenu(app.ctx)
}

func (app *App) initMenu() *menu.Menu {
	trayMenu := menu.NewMenu()
	var menus []Menu
	if err := json.Unmarshal(built_in_menu, &menus); err != nil {
		fmt.Println("failed to unmarshal menus:", err)
	}

	if runtime.GOOS == "darwin" {
		trayMenu.Append(menu.AppMenu())
		trayMenu.Append(menu.EditMenu())
	}
	// 自动生成内置菜单
	notSupport := []string{"10600", "20200"}
	for _, _menu := range menus {
		tmp := _menu
		plt := trayMenu.AddSubmenu(tmp.Title)
		for _, _submenu := range _menu.SubMenu {
			subtmp := _submenu
			plt.AddText(subtmp.Label, nil, func(data *menu.CallbackData) {
				if strings.Contains(strings.Join(notSupport, ","), subtmp.Id) {
					wruntime.BrowserOpenURL(app.ctx, subtmp.Url)
				} else {
					wruntime.WindowExecJS(app.ctx, fmt.Sprintf("window.location.replace('%s');", subtmp.Url))
					app.WriteLastPage(subtmp.Url)
				}
			})
			if subtmp.Separator {
				plt.AddSeparator()
			}
		}
	}

	custom := trayMenu.AddSubmenu("自定义平台")
	customMenuData := app.ReadMenu()
	groups := make(map[string][]PlatForm)
	for _, p := range customMenuData {
		if p.Group != "" {
			groups[p.Group] = append(groups[p.Group], p)
			continue
		}
		groups["默认分组"] = append(groups["默认分组"], p)
	}

	for k, v := range groups {
		if k != "默认分组" {
			custom.AddSeparator()
		}
		g := custom.AddSubmenu(k)
		vv := v
		for _, p := range vv {
			// go的for循环陷阱
			temp := p
			g.Append(&menu.MenuItem{
				Label: temp.Label,
				Type:  menu.TextType,
				Click: func(cd *menu.CallbackData) {
					jscode := fmt.Sprintf("window.location.replace('%s');", temp.Url)
					wruntime.WindowExecJS(app.ctx, jscode)
					app.WriteLastPage(temp.Url)
				},
			})
		}
	}

	// 工具
	setting := trayMenu.AddSubmenu("设置")
	setting.AddText("打开设置", keys.CmdOrCtrl("o"), func(cd *menu.CallbackData) {
		home := ConfigPath("home.txt")
		url, err := os.ReadFile(home)
		if err != nil {
			fmt.Println("Error reading file", err)
			url = []byte("wails://wails/")
		}
		data := string(url)
		fmt.Println(data)
		wruntime.WindowExecJS(app.ctx, fmt.Sprintf("window.location.replace('%s');", data))
		wruntime.WindowReload(app.ctx)
	})
	setting.AddText("侧边栏模式", keys.CmdOrCtrl("s"), func(cd *menu.CallbackData) {
		app.SideMode()
	})
	setting.AddText("窗口模式", keys.CmdOrCtrl("w"), func(cd *menu.CallbackData) {
		app.WindowMode()
	})

	about := trayMenu.AddSubmenu("帮助")
	about.AddText("关于我们", nil, func(cd *menu.CallbackData) {
		wruntime.MessageDialog(app.ctx, wruntime.MessageDialogOptions{
			Title:   "关于我们",
			Message: "GPTFusion " + Version + "\n\n" + "作者：lpdswing\n\n" + "请关注微信公众号：Go学习日记",
		})
	})
	about.AddText("前往Github", nil, func(cd *menu.CallbackData) {
		wruntime.BrowserOpenURL(app.ctx, "https://github.com/lpdswing/chatgpt")
	})
	about.AddText("检查更新", nil, func(cd *menu.CallbackData) {
		// 检查更新
		app.updateDialog(true)
	})
	return trayMenu
}

func (app *App) ImportPlatfrom() {
	// 选择文件
	file, err := wruntime.OpenFileDialog(app.ctx, wruntime.OpenDialogOptions{
		Title: "选择文件",
	})
	if err != nil {
		fmt.Println("Error opening file", err)
		return
	}
	// 读取文件
	content, err := os.ReadFile(file)
	if err != nil {
		fmt.Println("Error reading file", err)
		return
	}
	// 解析json
	var platforms []PlatForm
	if err := json.Unmarshal(content, &platforms); err != nil {
		fmt.Println("failed to unmarshal platforms:", err)
		return
	}
	//fmt.Println(platforms)
	// 保存到文件
	app.EditMenu(platforms)
}

func (app *App) ExportPlatfrom() {
	// 选择文件
	platorms := app.ReadMenu()
	currentTime := time.Now().Format("20060102_150405")
	file, err := wruntime.SaveFileDialog(app.ctx, wruntime.SaveDialogOptions{
		Title:           "选择文件",
		DefaultFilename: "menu" + currentTime + ".json",
	})
	if err != nil {
		fmt.Println("Error opening file", err)
		return
	}
	// 写入文件
	content, err := json.Marshal(platorms)
	if err != nil {
		fmt.Println("failed to marshal platforms:", err)
		return
	}
	if err := os.WriteFile(file, content, 0644); err != nil {
		fmt.Println("Error writing file", err)
		return
	}
}
