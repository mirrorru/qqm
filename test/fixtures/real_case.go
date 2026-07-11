package fixtures

type DictSubjWithPersonAndLegal struct {
	Subj   DictsSubjTableRowShort  `tbl:"from"`
	Person DictsSubjPersonRowShort `tbl:"join=left"`
	Legal  DictsSubjLegalRowShort  `tbl:"join=left"`
}

type dictsSubjTableMarker struct{}

func (dictsSubjTableMarker) SQLName() string {
	return "subj_table"
}

type DictsSubjTableRowShort struct {
	dictsSubjTableMarker
	ID      SubjID      `tbl:"pk;auto"`
	Name    SubjName    `tbl:"sort=1"`
	Address SubjAddress `tbl:"sort=2"`
}

type dictsSubjPersonMarker struct{}

func (dictsSubjPersonMarker) SQLName() string {
	return "subj_person"
}

type DictsSubjPersonRowShort struct {
	dictsSubjPersonMarker
	SubjID SubjID `tbl:"pk;ref=subj_table:id"`
	Val    SomeVal
}

type dictsSubjLegalMarker struct{}

func (dictsSubjLegalMarker) SQLName() string { return "subj_legal" }

type DictsSubjLegalRowShort struct {
	dictsSubjLegalMarker
	SubjID SubjID `tbl:"pk;ref=subj_table:id"`
	INN    SubjINN
}

type SubjID int64
type SubjName string
type SubjAddress string
type SubjINN string
type SomeVal int
