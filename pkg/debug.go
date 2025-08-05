package pkg

import (
	"runtime/debug"
	"strings"
)

func GetVersionFromBuildInfo() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	// info.Main.Version 里就是主模块版本号，比如 "v1.0.3" 或 "(devel)"
	ver := info.Main.Version
	if strings.HasPrefix(ver, "v") {
		return ver
	}
	// 如果是 "(devel)" 或空，自己定义兜底
	return "dev"
}
