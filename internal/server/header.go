package server

import (
	"errors"
	"runtime/debug"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/constant"
	"github.com/gofrs/uuid/v5"
	uuid2 "github.com/google/uuid"
	"github.com/samber/oops"
)

const RequestIDHeader = "X-Request-ID"

func requestIDGenerator() string {
	id, err := uuid.NewV7()
	if err == nil {
		return id.String()
	}
	return uuid2.NewString()
}

func buildServerHeader() (string, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", oops.In("server").Owner("server custom header").Wrap(errors.New("could not read build info"))
	}
	version := info.Main.Version
	return collectionx.NewList(constant.ServerHeaderPrefix, version).Join("/"), nil
}
