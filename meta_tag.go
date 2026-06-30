package qqm

import "github.com/mirrorru/qqm/meta"

// SetTagName устанавливает имя тега для парсинга метаданных.
// EN: SetTagName sets the tag name for metadata parsing.
func SetTagName(newTagVal string) string {
	if newTagVal != "" {
		meta.TagName = newTagVal
	}
	return meta.TagName
}
