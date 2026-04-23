package server

import (
	"github.com/arcgolabs/collectionx"
	"github.com/arcgolabs/dix"
	"github.com/daiyuang/spack/internal/constant"
)

const RequestIDHeader = "X-Request-ID"

func buildServerHeader(meta dix.AppMeta) string {
	return collectionx.NewList[string](constant.ServerHeaderPrefix, meta.Version).Join("/")
}
