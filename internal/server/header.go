package server

import (
	"errors"
	"runtime/debug"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/constant"
	"github.com/samber/oops"
)

const RequestIDHeader = "X-Request-ID"

func buildServerHeader() (string, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", oops.In("server").Owner("server custom header").Wrap(errors.New("could not read build info"))
	}
	version := info.Main.Version
	return collectionx.NewList(constant.ServerHeaderPrefix, version).Join("/"), nil
}
