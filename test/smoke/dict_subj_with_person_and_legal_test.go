//go:build smoke

package smoke

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/mirrorru/qqm"
	"github.com/mirrorru/qqm/txproc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/test/fixtures"
	_ "modernc.org/sqlite"
)

func TestSmoke_MultiQuery_DictSubjWithPersonAndLegal(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = db.Exec(`
		CREATE TABLE subj_table (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			address TEXT NOT NULL
		);
		CREATE TABLE subj_person (
			subj_id INTEGER NOT NULL REFERENCES subj_table(id),
			val INTEGER NOT NULL,
			birthday TEXT NOT NULL DEFAULT '',
			gender TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE subj_legal (
			subj_id INTEGER NOT NULL REFERENCES subj_table(id),
			inn TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	ex := txproc.NewDBAdapterVal(db)
	ctx := context.Background()

	subjTbl := qqm.NewTable[fixtures.DictsSubjTableRowShort](dialect.SQLiteDialect{})

	subj1, _, err := subjTbl.Ins(ctx, ex, &fixtures.DictsSubjTableRowShort{
		Name:    fixtures.SubjName("Subject 1"),
		Address: fixtures.SubjAddress("Address 1"),
	})
	require.NoError(t, err)

	subj2, _, err := subjTbl.Ins(ctx, ex, &fixtures.DictsSubjTableRowShort{
		Name:    fixtures.SubjName("Subject 2"),
		Address: fixtures.SubjAddress("Address 2"),
	})
	require.NoError(t, err)

	subj3, _, err := subjTbl.Ins(ctx, ex, &fixtures.DictsSubjTableRowShort{
		Name:    fixtures.SubjName("Subject 3"),
		Address: fixtures.SubjAddress("Address 3"),
	})
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO subj_person (subj_id, val, birthday, gender) VALUES (?, ?, ?, ?)`, subj1.ID, 100, "2000-01-15", "male")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO subj_legal (subj_id, inn) VALUES (?, ?)`, subj1.ID, "INN-001")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO subj_person (subj_id, val, birthday, gender) VALUES (?, ?, ?, ?)`, subj2.ID, 200, "1995-06-20", "female")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO subj_legal (subj_id, inn) VALUES (?, ?)`, subj3.ID, "INN-003")
	require.NoError(t, err)

	bday1 := time.Date(2000, 1, 15, 0, 0, 0, 0, time.UTC)
	bday2 := time.Date(1995, 6, 20, 0, 0, 0, 0, time.UTC)

	q := qqm.NewQuery[fixtures.DictSubjWithPersonAndLegal](dialect.SQLiteDialect{})

	t.Run("Many returns all subjects with their joined data", func(t *testing.T) {
		results, err := q.Many(ctx, ex, nil)
		require.NoError(t, err)
		assert.Len(t, results, 3)

		byName := make(map[fixtures.SubjName]fixtures.DictSubjWithPersonAndLegal)
		for _, r := range results {
			byName[r.Subj.Name] = *r
		}

		subj1Row := byName[fixtures.SubjName("Subject 1")]
		assert.Equal(t, subj1.ID, subj1Row.Subj.ID)
		assert.NotZero(t, subj1Row.Person.SubjID, "Person should not be zero when record exists")
		assert.Equal(t, fixtures.SomeVal(100), subj1Row.Person.Val)
		assert.NotZero(t, subj1Row.Legal.SubjID, "Legal should not be zero when record exists")
		assert.Equal(t, fixtures.SubjINN("INN-001"), subj1Row.Legal.INN)
		assert.True(t, bday1.Equal(time.Time(subj1Row.Person.Birthday.Date)), "Birthday should match")
		assert.Equal(t, fixtures.GenderTypeMale, subj1Row.Person.Gender)

		subj2Row := byName[fixtures.SubjName("Subject 2")]
		assert.Equal(t, subj2.ID, subj2Row.Subj.ID)
		assert.NotZero(t, subj2Row.Person.SubjID, "Person should not be zero when record exists")
		assert.Equal(t, fixtures.SomeVal(200), subj2Row.Person.Val)
		assert.True(t, bday2.Equal(time.Time(subj2Row.Person.Birthday.Date)), "Birthday should match")
		assert.Equal(t, fixtures.GenderTypeFemale, subj2Row.Person.Gender)
		assert.Zero(t, subj2Row.Legal.SubjID, "Legal should be zero when no record exists")

		subj3Row := byName[fixtures.SubjName("Subject 3")]
		assert.Equal(t, subj3.ID, subj3Row.Subj.ID)
		assert.Zero(t, subj3Row.Person.SubjID, "Person should be zero when no record exists")
		assert.NotZero(t, subj3Row.Legal.SubjID, "Legal should not be zero when record exists")
		assert.Equal(t, fixtures.SubjINN("INN-003"), subj3Row.Legal.INN)
		assert.True(t, time.Time(subj3Row.Person.Birthday.Date).IsZero(), "Birthday should be zero when no person record exists")
		assert.Equal(t, fixtures.GenderTypeUnknown, subj3Row.Person.Gender, "Gender should be unknown when no person record exists")
	})

	t.Run("One returns subject with both Person and Legal", func(t *testing.T) {
		row, err := q.One(ctx, ex, subj1.ID)
		require.NoError(t, err)
		require.NotNil(t, row)
		assert.Equal(t, subj1.ID, row.Subj.ID)
		assert.Equal(t, fixtures.SubjName("Subject 1"), row.Subj.Name)
		assert.NotZero(t, row.Person.SubjID, "Person should not be zero")
		assert.Equal(t, fixtures.SomeVal(100), row.Person.Val)
		assert.NotZero(t, row.Legal.SubjID, "Legal should not be zero")
		assert.Equal(t, fixtures.SubjINN("INN-001"), row.Legal.INN)
		assert.True(t, bday1.Equal(time.Time(row.Person.Birthday.Date)), "Birthday should match")
		assert.Equal(t, fixtures.GenderTypeMale, row.Person.Gender)
	})

	t.Run("One returns subject with Person only", func(t *testing.T) {
		row, err := q.One(ctx, ex, subj2.ID)
		require.NoError(t, err)
		require.NotNil(t, row)
		assert.Equal(t, subj2.ID, row.Subj.ID)
		assert.NotZero(t, row.Person.SubjID, "Person should not be zero")
		assert.Equal(t, fixtures.SomeVal(200), row.Person.Val)
		assert.True(t, bday2.Equal(time.Time(row.Person.Birthday.Date)), "Birthday should match")
		assert.Equal(t, fixtures.GenderTypeFemale, row.Person.Gender)
		assert.Zero(t, row.Legal.SubjID, "Legal should be zero")
	})

	t.Run("One returns subject with Legal only", func(t *testing.T) {
		row, err := q.One(ctx, ex, subj3.ID)
		require.NoError(t, err)
		require.NotNil(t, row)
		assert.Equal(t, subj3.ID, row.Subj.ID)
		assert.Zero(t, row.Person.SubjID, "Person should be zero")
		assert.True(t, time.Time(row.Person.Birthday.Date).IsZero(), "Birthday should be zero when no person record exists")
		assert.Equal(t, fixtures.GenderTypeUnknown, row.Person.Gender, "Gender should be unknown when no person record exists")
		assert.NotZero(t, row.Legal.SubjID, "Legal should not be zero")
		assert.Equal(t, fixtures.SubjINN("INN-003"), row.Legal.INN)
	})
}
