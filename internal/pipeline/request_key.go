package pipeline

import (
	"strconv"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func buildRequestKey(assetPath string, encodings, formats collectionx.List[string], widths collectionx.List[int]) string {
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

func writeStringList(builder *strings.Builder, values collectionx.List[string]) {
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

func writeIntList(builder *strings.Builder, values collectionx.List[int]) {
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
		panic(err)
	}
}

func writeBuilderByte(builder *strings.Builder, value byte) {
	if err := builder.WriteByte(value); err != nil {
		panic(err)
	}
}
