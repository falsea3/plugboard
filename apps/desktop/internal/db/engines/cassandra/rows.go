package cassandra

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	gocql "github.com/apache/cassandra-gocql-driver/v2"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

const chunk = 1000

func typeName(ti gocql.TypeInfo) string {
	switch ti.Type() {
	case gocql.TypeDate:
		return "date"
	case gocql.TypeTime:
		return "time"
	case gocql.TypeBlob:
		return "blob"
	case gocql.TypeInt, gocql.TypeBigInt, gocql.TypeSmallInt, gocql.TypeTinyInt, gocql.TypeVarint, gocql.TypeCounter, gocql.TypeFloat, gocql.TypeDouble, gocql.TypeDecimal:
		return "int"
	case gocql.TypeBoolean:
		return "boolean"
	case gocql.TypeTimestamp:
		return "timestamp"
	case gocql.TypeText, gocql.TypeVarchar, gocql.TypeAscii, gocql.TypeUUID, gocql.TypeTimeUUID, gocql.TypeInet:
		return "text"
	}
	return ""
}

type fetched struct {
	columns []model.ResultColumn
	types   []string
	rows    [][]any
	next    []byte
}

func fetch(ctx context.Context, gs *gocql.Session, stmt string, args []any, state []byte, want int) (fetched, error) {
	var out fetched
	for {
		size := min(want-len(out.rows), 5000)
		iter := gs.Query(stmt, args...).WithContext(ctx).PageSize(size).PageState(state).Iter()
		if out.columns == nil {
			for _, c := range iter.Columns() {
				t := typeName(c.TypeInfo)
				out.columns = append(out.columns, model.ResultColumn{Name: c.Name, Type: strings.ToLower(fmt.Sprint(c.TypeInfo)), Kind: kindOf(t)})
				out.types = append(out.types, t)
			}
		}
		if len(out.columns) > 0 {
			rd, err := iter.RowData()
			if err != nil {
				iter.Close()
				return out, err
			}
			for n := iter.NumRows(); n > 0 && iter.Scan(rd.Values...); n-- {
				row := make([]any, len(rd.Values))
				for i, v := range rd.Values {
					row[i] = normalize(v, out.types[i])
				}
				out.rows = append(out.rows, row)
			}
		}
		state = iter.PageState()
		if err := iter.Close(); err != nil {
			return out, err
		}
		out.next = state
		if len(state) == 0 || len(out.rows) >= want || len(out.columns) == 0 {
			return out, nil
		}
	}
}

var syntaxAt = regexp.MustCompile(`line (\d+):(\d+)`)

func errorPosition(err error, stmt string) int {
	var se *gocql.RequestErrSyntax
	if !errors.As(err, &se) {
		return -1
	}
	m := syntaxAt.FindStringSubmatch(se.Message())
	if m == nil {
		return -1
	}
	line, _ := strconv.Atoi(m[1])
	col, _ := strconv.Atoi(m[2])
	pos := 0
	for l := 1; l < line; l++ {
		i := strings.IndexByte(stmt[pos:], '\n')
		if i < 0 {
			return -1
		}
		pos += i + 1
	}
	if pos+col > len(stmt) {
		return -1
	}
	return pos + col
}
