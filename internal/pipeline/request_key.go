package pipeline

import (
	cxlist "github.com/arcgolabs/collectionx/list"
	"github.com/samber/oops"
	"strconv"
	"strings"
)

func buildRequestKey(assetPath string, encodings, formats *cxlist.List[string], widths *cxlist.List[int]) string {
	var builder strings.Builder
	writeBuilderString(&builder, assetPath)
	writeBuilderString(&builder, "|e=")
	writeStringList(&builder, encodings)
	writeBuilderString(&builder, "|f=")
	writeStringList(&builder, formats)
	writeBuilderString(&builder, "|w=")
	writeIntList(&builder, widths)
	return builder.String()
}

func writeStringList(builder *strings.Builder, values *cxlist.List[string]) {
	if values == nil {
		return
	}
	values.Range(func(index int, value string) bool {
		if index > 0 {
			writeBuilderByte(builder, ',')
		}
		writeBuilderString(builder, value)
		return true
	})
}

func writeIntList(builder *strings.Builder, values *cxlist.List[int]) {
	if values == nil {
		return
	}
	values.Range(func(index int, value int) bool {
		if index > 0 {
			writeBuilderByte(builder, ',')
		}
		writeBuilderString(builder, strconv.Itoa(value))
		return true
	})
}

func writeBuilderString(builder *strings.Builder, value string) {
	if _, err := builder.WriteString(value); err != nil {
		errors := oops.In("request key").Wrap(err)
		panic(errors)
	}
}

func writeBuilderByte(builder *strings.Builder, value byte) {
	if err := builder.WriteByte(value); err != nil {
		panic(err)
	}
}
