package qqm_test

type complexRow struct {
	mustIgnorePrivateInt int      `qqm:"pk"`
	IntPKLvl1            int64    `qqm:"pk;auto"`
	StringLvl1           string   `qqm:"col=lvl1_string"`
	embedded             Embedded `qqm:"prefix=embd_;embed"`
}

type Embedded struct {
	Anonimous
	EmbdInt    int    `qqm:"pk"`
	EmbdString string `qqm:"col=string_embedded"`
}

type Anonimous struct {
	ID int `qqm:"auto"`
}

type Group struct {
	ID   int `qqm:"auto"`
	Flag bool
}
