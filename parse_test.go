package main

import (
	"testing"
)

func itemsCmp(items1 tagItems, items2 tagItems) bool  {
	if len(items1) != len(items2) {
		return false
	}

	m1 := map[string]string{}
	m2 := map[string]string{}

	for _, v := range items1 {
		m1[v.key] = v.value
	}

	for _, v := range items2 {
		m2[v.key] = v.value
	}

	if len(m1) != len(m2) {
		return false
	}

	for k, v := range m1 {
		if _, ok := m2[k]; !ok || v != m2[k] {
			return false
		}
	}

	return true
}

func TestParseTag(t *testing.T) {
	tags := []string{
		"valid:\"ip\" yaml:\"ip\"",
		`valid:"i\"p"`,
		`valid:"i\"p"  `,
		`valid:"i\"p"  x`,
		`valid:"i\"p"  x:`,
	}

	ans := []tagItems {
		tagItems{
			tagItem{"valid", `"ip"`},
			tagItem{"yaml", `"ip"`},
		},
		tagItems{
			tagItem{"valid", `"i\"p"`},
		},
		tagItems{
			tagItem{"valid", `"i\"p"`},
		},
		tagItems{
			tagItem{"valid", `"i\"p"`},
		},
		tagItems{
			tagItem{"valid", `"i\"p"`},
		},
	}

	for i, v := range tags {
		items := parseTag(v)
		if !itemsCmp(items, ans[i]) {
			t.Error("parsetag error", items, ans[i])
			break
		}

	}
}
