// Package control 控制插件的启用与优先级等
package control

import (
	"os"
	"strconv"
	"sync"

	. "github.com/FloatTech/ZeroBot-Plugin/data"
	"github.com/sirupsen/logrus"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/extension"
	"github.com/wdvxdr1123/ZeroBot/message"
)

var (
	db = &Sqlite{DBPath: "data/control/plugins.db"}
	// managers 每个插件对应的管理
	managers = map[string]*Control{}
	mu       = sync.RWMutex{}
)

// Control is to control the plugins.
type Control struct {
	sync.RWMutex
	service string
	options Options
}

// New returns Manager with settings.
func New(service string, o *Options) *Control {
	m := &Control{service: service,
		options: func() Options {
			if o == nil {
				return Options{}
			}
			return *o
		}(),
	}
	mu.Lock()
	defer mu.Unlock()
	managers[service] = m
	err := db.Create(service, &grpcfg{})
	if err != nil {
		panic(err)
	}
	return m
}

// Enable enables a group to pass the Manager.
func (m *Control) Enable(groupID int64) {
	m.Lock()
	err := db.Insert(m.service, &grpcfg{groupID, 0})
	if err != nil {
		logrus.Errorf("[control] %v", err)
	}
	m.Unlock()
}

// Disable disables a group to pass the Manager.
func (m *Control) Disable(groupID int64) {
	m.Lock()
	err := db.Insert(m.service, &grpcfg{groupID, 1})
	if err != nil {
		logrus.Errorf("[control] %v", err)
	}
	m.Unlock()
}

// Handler 返回 预处理器
func (m *Control) Handler() zero.Rule {
	return func(ctx *zero.Ctx) bool {
		m.RLock()
		ctx.State["manager"] = m
		var c grpcfg
		err := db.Find(m.service, &c, "WHERE gid = "+strconv.FormatInt(ctx.Event.GroupID, 10))
		if err == nil {
			m.RUnlock()
			return c.Disable == 0
		} else {
			logrus.Errorf("[control] %v", err)
		}
		m.RUnlock()
		if m.options.DisableOnDefault {
			m.Disable(ctx.Event.GroupID)
		} else {
			m.Enable(ctx.Event.GroupID)
		}
		return !m.options.DisableOnDefault
	}
}

// Lookup returns a Manager by the service name, if
// not exist, it will returns nil.
func Lookup(service string) (*Control, bool) {
	mu.RLock()
	defer mu.RUnlock()
	m, ok := managers[service]
	return m, ok
}

// ForEach iterates through managers.
func ForEach(iterator func(key string, manager *Control) bool) {
	mu.RLock()
	m := copyMap(managers)
	mu.RUnlock()
	for k, v := range m {
		if !iterator(k, v) {
			return
		}
	}
}

func copyMap(m map[string]*Control) map[string]*Control {
	ret := make(map[string]*Control, len(m))
	for k, v := range m {
		ret[k] = v
	}
	return ret
}

func Init() {
	err := os.MkdirAll("data/control", 0755)
	if err != nil {
		panic(err)
	}
	zero.OnCommandGroup([]string{"启用", "enable"}, zero.AdminPermission, zero.OnlyGroup).
		Handle(func(ctx *zero.Ctx) {
			model := extension.CommandModel{}
			_ = ctx.Parse(&model)
			service, ok := Lookup(model.Args)
			if !ok {
				ctx.Send("没有找到指定服务!")
			}
			service.Enable(ctx.Event.GroupID)
			ctx.Send(message.Text("已启用服务: " + model.Args))
		})

	zero.OnCommandGroup([]string{"禁用", "disable"}, zero.AdminPermission, zero.OnlyGroup).
		Handle(func(ctx *zero.Ctx) {
			model := extension.CommandModel{}
			_ = ctx.Parse(&model)
			service, ok := Lookup(model.Args)
			if !ok {
				ctx.Send("没有找到指定服务!")
			}
			service.Disable(ctx.Event.GroupID)
			ctx.Send(message.Text("已关闭服务: " + model.Args))
		})

	zero.OnCommandGroup([]string{"用法", "usage"}, zero.AdminPermission, zero.OnlyGroup).
		Handle(func(ctx *zero.Ctx) {
			model := extension.CommandModel{}
			_ = ctx.Parse(&model)
			service, ok := Lookup(model.Args)
			if !ok {
				ctx.Send("没有找到指定服务!")
			}
			if service.options.Help != "" {
				ctx.Send(service.options.Help)
			} else {
				ctx.Send("该服务无帮助!")
			}
		})

	zero.OnCommandGroup([]string{"服务列表", "service_list"}, zero.AdminPermission, zero.OnlyGroup).
		Handle(func(ctx *zero.Ctx) {
			msg := `---服务列表---`
			i := 0
			ForEach(func(key string, manager *Control) bool {
				i++
				msg += "\n" + strconv.Itoa(i) + `: ` + key
				return true
			})
			ctx.Send(message.Text(msg))
		})
}
