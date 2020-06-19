package pgxscan_test

import (
	"context"
	"flag"
	"os"
	"reflect"
	"testing"

	"github.com/georgysavva/dbscan/pgxscan"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/georgysavva/dbscan/internal/testutil"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	testDB *pgxpool.Pool
	ctx    = context.Background()
)

type testDst struct {
	Foo string
	Bar string
}

func TestQueryAll(t *testing.T) {
	t.Parallel()
	sqlText := `
		SELECT *
		FROM (
			VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
		) AS t (foo, bar)
	`
	expected := []*testDst{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}

	var got []*testDst
	err := pgxscan.QueryAll(ctx, testDB, &got, sqlText)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestQueryOne(t *testing.T) {
	t.Parallel()
	sqlText := `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	expected := testDst{Foo: "foo val", Bar: "bar val"}

	var got testDst
	err := pgxscan.QueryOne(ctx, testDB, &got, sqlText)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanAll(t *testing.T) {
	t.Parallel()
	sqlText := `
		SELECT *
		FROM (
			VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
		) AS t (foo, bar)
	`
	expected := []*testDst{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}
	rows, err := testDB.Query(ctx, sqlText)
	require.NoError(t, err)

	var got []*testDst
	err = pgxscan.ScanAll(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanOne(t *testing.T) {
	t.Parallel()
	sqlText := `
		SELECT 'foo val' AS foo, 'bar val' AS bar
	`
	expected := testDst{Foo: "foo val", Bar: "bar val"}
	rows, err := testDB.Query(ctx, sqlText)
	require.NoError(t, err)

	var got testDst
	err = pgxscan.ScanOne(&got, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanOne_noRows_returnsNotFoundErr(t *testing.T) {
	t.Parallel()
	sqlText := `
		SELECT NULL AS foo, NULL AS bar LIMIT 0;
	`
	rows, err := testDB.Query(ctx, sqlText)
	require.NoError(t, err)

	var got testDst
	err = pgxscan.ScanOne(&got, rows)

	assert.True(t, pgxscan.NotFound(err))
}

func TestScanRow(t *testing.T) {
	t.Parallel()
	sqlText := `
		SELECT *
		FROM (
			VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
		) AS t (foo, bar)
	`
	expected := []*testDst{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}

	var got []*testDst
	rows, err := testDB.Query(ctx, sqlText)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		dst := &testDst{}
		err := pgxscan.ScanRow(dst, rows)
		require.NoError(t, err)
		got = append(got, dst)
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, expected, got)
}

func TestRowScanner(t *testing.T) {
	t.Parallel()
	sqlText := `
		SELECT *
		FROM (
			VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
		) AS t (foo, bar)
	`
	expected := []*testDst{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}

	var got []*testDst
	rows, err := testDB.Query(ctx, sqlText)
	require.NoError(t, err)
	defer rows.Close()
	rs := pgxscan.NewRowScanner(rows)
	for rows.Next() {
		dst := &testDst{}
		err := rs.Scan(dst)
		require.NoError(t, err)
		got = append(got, dst)
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, expected, got)
}

func TestRowsAdapter_Scan(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		d1   interface{}
		d2   interface{}
		d3   interface{}
	}{
		{
			name: "all destinations are *interface{}",
			d1:   new(interface{}),
			d2:   new(interface{}),
			d3:   new(interface{}),
		},
		{
			name: "none of destinations are *interface{}",
			d1:   new(string),
			d2:   new(string),
			d3:   new(string),
		},
		{
			name: "mix of *interface{} and non *interface{} destinations",
			d1:   new(interface{}),
			d2:   new(string),
			d3:   new(interface{}),
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows, err := testDB.Query(ctx, `select '1', '2', '3'`)
			require.NoError(t, err)
			rows.Next()
			defer rows.Close()
			ra := pgxscan.NewRowsAdapter(rows)
			err = ra.Scan(tc.d1, tc.d2, tc.d3)
			require.NoError(t, err)
			require.NoError(t, rows.Err())

			assert.Equal(t, "1", reflect.ValueOf(tc.d1).Elem().Interface())
			assert.Equal(t, "2", reflect.ValueOf(tc.d2).Elem().Interface())
			assert.Equal(t, "3", reflect.ValueOf(tc.d3).Elem().Interface())
		})
	}
}

func TestMain(m *testing.M) {
	exitCode := func() int {
		flag.Parse()
		ts, err := testutil.StartCrdbServer()
		if err != nil {
			panic(err)
		}
		defer ts.Stop()
		testDB, err = pgxpool.Connect(ctx, ts.PGURL().String())
		if err != nil {
			panic(err)
		}
		defer testDB.Close()
		return m.Run()
	}()
	os.Exit(exitCode)
}
