package printer

type jsonOriginal struct {
	Path     string         `json:"path"`
	MIME     string         `json:"mime"`
	Size     int64          `json:"size"`
	Hash     string         `json:"hash"`
	Variants []*jsonVariant `json:"variants"`
	VarCount int            `json:"variant_count"`
}

type jsonVariant struct {
	Path        string `json:"path"`
	Ext         string `json:"ext"`
	VariantType string `json:"variant_type"`
	Size        int64  `json:"size"`
}

type jsonReport struct {
	Originals []*jsonOriginal            `json:"originals"`
	ByMIME    map[string][]*jsonOriginal `json:"by_mime"`
	Total     int                        `json:"total"`
}
