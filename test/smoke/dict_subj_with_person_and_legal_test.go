//go:build smoke

package smoke

import (
	"context"
	"database/sql"
	"testing"

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
			val INTEGER NOT NULL
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

	subj1, err := subjTbl.Insert(ctx, ex, &fixtures.DictsSubjTableRowShort{
		Name:    "Subject 1",
		Address: "Address 1",
	})
	require.NoError(t, err)

	subj2, err := subjTbl.Insert(ctx, ex, &fixtures.DictsSubjTableRowShort{
		Name:    "Subject 2",
		Address: "Address 2",
	})
	require.NoError(t, err)

	subj3, err := subjTbl.Insert(ctx, ex, &fixtures.DictsSubjTableRowShort{
		Name:    "Subject 3",
		Address: "Address 3",
	})
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO subj_person (subj_id, val) VALUES (?, ?)`, subj1.ID, 100)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO subj_legal (subj_id, inn) VALUES (?, ?)`, subj1.ID, "INN-001")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO subj_person (subj_id, val) VALUES (?, ?)`, subj2.ID, 200)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO subj_legal (subj_id, inn) VALUES (?, ?)`, subj3.ID, "INN-003")
	require.NoError(t, err)

	q, err := qqm.NewQuery[fixtures.DictSubjWithPersonAndLegal](dialect.SQLiteDialect{})
	require.NoError(t, err)

	t.Run("List returns all subjects with their joined data", func(t *testing.T) {
		results, err := q.List(ctx, ex)
		require.NoError(t, err)
		assert.Len(t, results, 3)

		byName := make(map[string]fixtures.DictSubjWithPersonAndLegal)
		for _, r := range results {
			byName[string(r.Subj.Name)] = *r
		}

		subj1Row := byName["Subject 1"]
		assert.Equal(t, subj1.ID, subj1Row.Subj.ID)
		require.NotNil(t, subj1Row.Person, "Person should not be nil when record exists")
		assert.Equal(t, fixtures.SomeVal(100), subj1Row.Person.Val)
		require.NotNil(t, subj1Row.Legal, "Legal should not be nil when record exists")
		assert.Equal(t, fixtures.SubjINN("INN-001"), subj1Row.Legal.INN)

		subj2Row := byName["Subject 2"]
		assert.Equal(t, subj2.ID, subj2Row.Subj.ID)
		require.NotNil(t, subj2Row.Person, "Person should not be nil when record exists")
		assert.Equal(t, fixtures.SomeVal(200), subj2Row.Person.Val)
		assert.Nil(t, subj2Row.Legal, "Legal should be nil when no record exists")

		subj3Row := byName["Subject 3"]
		assert.Equal(t, subj3.ID, subj3Row.Subj.ID)
		assert.Nil(t, subj3Row.Person, "Person should be nil when no record exists")
		require.NotNil(t, subj3Row.Legal, "Legal should not be nil when record exists")
		assert.Equal(t, fixtures.SubjINN("INN-003"), subj3Row.Legal.INN)
	})

	t.Run("One returns subject with both Person and Legal", func(t *testing.T) {
		row, err := q.One(ctx, ex, subj1.ID)
		require.NoError(t, err)
		require.NotNil(t, row)
		assert.Equal(t, subj1.ID, row.Subj.ID)
		assert.Equal(t, "Subject 1", string(row.Subj.Name))
		require.NotNil(t, row.Person, "Person should not be nil")
		assert.Equal(t, fixtures.SomeVal(100), row.Person.Val)
		require.NotNil(t, row.Legal, "Legal should not be nil")
		assert.Equal(t, fixtures.SubjINN("INN-001"), row.Legal.INN)
	})

	t.Run("One returns subject with Person only", func(t *testing.T) {
		row, err := q.One(ctx, ex, subj2.ID)
		require.NoError(t, err)
		require.NotNil(t, row)
		assert.Equal(t, subj2.ID, row.Subj.ID)
		require.NotNil(t, row.Person, "Person should not be nil")
		assert.Equal(t, fixtures.SomeVal(200), row.Person.Val)
		assert.Nil(t, row.Legal, "Legal should be nil")
	})

	t.Run("One returns subject with Legal only", func(t *testing.T) {
		row, err := q.One(ctx, ex, subj3.ID)
		require.NoError(t, err)
		require.NotNil(t, row)
		assert.Equal(t, subj3.ID, row.Subj.ID)
		assert.Nil(t, row.Person, "Person should be nil")
		require.NotNil(t, row.Legal, "Legal should not be nil")
		assert.Equal(t, fixtures.SubjINN("INN-003"), row.Legal.INN)
	})
}
