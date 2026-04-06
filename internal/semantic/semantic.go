package semantic

import (
	"fmt"

	"github.com/headercat/erdn-lang/internal/ast"
)

// Validate checks an AST Program for semantic errors.
func Validate(prog *ast.Program) []error {
	var errs []error

	tableNames := map[string]*ast.Table{}
	for _, tbl := range prog.Tables {
		if _, exists := tableNames[tbl.Name]; exists {
			errs = append(errs, fmt.Errorf("line %d: duplicate table name %q", tbl.Line, tbl.Name))
		} else {
			tableNames[tbl.Name] = tbl
		}

		colNames := map[string]bool{}
		pkCount := 0
		for _, col := range tbl.Columns {
			if colNames[col.Name] {
				errs = append(errs, fmt.Errorf("line %d: duplicate column name %q in table %q", col.Line, col.Name, tbl.Name))
			}
			colNames[col.Name] = true

			hasNullable := false
			hasNotNull := false
			for _, mod := range col.Modifiers {
				switch mod.Kind {
				case ast.ModPrimaryKey:
					pkCount++
				case ast.ModNullable:
					hasNullable = true
				case ast.ModNotNull:
					hasNotNull = true
				}
			}
			if hasNullable && hasNotNull {
				errs = append(errs, fmt.Errorf("line %d: column %q in table %q has both nullable and not-null", col.Line, col.Name, tbl.Name))
			}
		}
		if pkCount > 1 {
			errs = append(errs, fmt.Errorf("line %d: table %q has more than one primary key", tbl.Line, tbl.Name))
		}
	}

	for _, lnk := range prog.Links {
		fromTbl, ok := tableNames[lnk.FromTable]
		if !ok {
			errs = append(errs, fmt.Errorf("line %d: link references unknown table %q", lnk.Line, lnk.FromTable))
		} else if !hasColumn(fromTbl, lnk.FromColumn) {
			errs = append(errs, fmt.Errorf("line %d: link references unknown column %q in table %q", lnk.Line, lnk.FromColumn, lnk.FromTable))
		}

		toTbl, ok := tableNames[lnk.ToTable]
		if !ok {
			errs = append(errs, fmt.Errorf("line %d: link references unknown table %q", lnk.Line, lnk.ToTable))
		} else if !hasColumn(toTbl, lnk.ToColumn) {
			errs = append(errs, fmt.Errorf("line %d: link references unknown column %q in table %q", lnk.Line, lnk.ToColumn, lnk.ToTable))
		}
	}

	return errs
}

func hasColumn(tbl *ast.Table, colName string) bool {
	for _, col := range tbl.Columns {
		if col.Name == colName {
			return true
		}
	}
	return false
}
