package fixtures

import "time"

type DictSubjWithPersonAndLegal struct {
	Subj   DictsSubjTableRowShort  `qqm:"primary"`
	Person DictsSubjPersonRowShort `qqm:"join=LEFT"`
	Legal  DictsSubjLegalRowShort  `qqm:"join=LEFT"`
}

type dictsSubjTableMarker struct{}

func (dictsSubjTableMarker) SQLName() string {
	return "subj_table"
}

type DictsSubjTableRowShort struct {
	dictsSubjTableMarker
	ID      SubjID      `qqm:"pk;auto"`
	Name    SubjName    `qqm:"order=1"`
	Address SubjAddress `qqm:"order=2"`
}

type dictsSubjPersonMarker struct{}

func (dictsSubjPersonMarker) SQLName() string {
	return "subj_person"
}

type DictsSubjPersonRowShort struct {
	dictsSubjPersonMarker
	SubjID SubjID `qqm:"pk;ref=subj_table.id"`
	Val    SomeVal
}

type dictsSubjLegalMarker struct{}

func (dictsSubjLegalMarker) SQLName() string {
	return "subj_legal"
}

type DictsSubjLegalRowShort struct {
	dictsSubjLegalMarker
	SubjID SubjID `qqm:"pk;ref=subj_table.id"`
	INN    SubjINN
}

type SubjID int64
type SubjName string
type SubjAddress string
type SubjBirthday time.Time
type SubjINN string
type SomeVal int
