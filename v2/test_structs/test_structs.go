package test_structs

type SimpleRow struct {
	ID        int64  `tbl:"pk;auto"`
	InsFld    int    `tbl:"ins;auto"`
	ReadFld   int    `tbl:"ro;sort=2,desc"`
	UpdFld    int    `tbl:"upd;sort=1"`
	SecretFld int    `tbl:"rskip"`
	Omits     string `tbl:"omit"`
	FreeFld   int
}

type ComplexRow struct {
	mustIgnorePrivateInt int      `tbl:"pk"`
	IntPKLvl1            int64    `tbl:"pk;auto"`
	StringLvl1           string   `tbl:"col=lvl1_string"`
	Embedded             Embedded `tbl:"prefix=embd_;embed"`
	Group
}

type Embedded struct {
	Anonimous
	EmbdInt    int    `tbl:"pk"`
	EmbdString string `tbl:"col=string_embedded"`
}

type Anonimous struct {
	ID int `tbl:"auto"`
}

type Group struct {
	ID   int `tbl:"auto"`
	Flag bool
}
