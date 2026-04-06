package render

import (
	"fmt"
	"strings"

	"github.com/headercat/erdn-lang/internal/ast"
	"github.com/headercat/erdn-lang/internal/parser"
)

// GenerateDOT converts an AST Program into a Graphviz DOT string.
func GenerateDOT(prog *ast.Program) string {
	var sb strings.Builder

	sb.WriteString("digraph erdn {\n")
	sb.WriteString("  graph [fontname=\"Helvetica\" rankdir=LR]\n")
	sb.WriteString("  node [fontname=\"Helvetica\" shape=plain]\n")
	sb.WriteString("  edge [fontname=\"Helvetica\"]\n\n")

	for _, tbl := range prog.Tables {
		sb.WriteString(renderTable(tbl))
		sb.WriteString("\n")
	}

	for _, lnk := range prog.Links {
		sb.WriteString(renderLink(lnk))
		sb.WriteString("\n")
	}

	sb.WriteString("}\n")
	return sb.String()
}

func renderTable(tbl *ast.Table) string {
	var sb strings.Builder

	const colspan = "5"

	sb.WriteString(fmt.Sprintf("  %s [label=<\n", dotID(tbl.Name)))
	sb.WriteString("    <TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\" CELLPADDING=\"5\">\n")

	// header row – dark background, white bold title
	sb.WriteString(fmt.Sprintf("      <TR><TD BGCOLOR=\"#2C3E50\" COLSPAN=\"%s\" ALIGN=\"CENTER\"><FONT COLOR=\"white\"><B>%s</B></FONT></TD></TR>\n",
		colspan, htmlEscape(tbl.Name)))

	// optional comment subtitle row
	if len(tbl.Comments) > 0 {
		tooltip := htmlEscape(strings.Join(tbl.Comments, " "))
		sb.WriteString(fmt.Sprintf("      <TR><TD BGCOLOR=\"#34495E\" COLSPAN=\"%s\" ALIGN=\"CENTER\"><FONT COLOR=\"#BDC3C7\" POINT-SIZE=\"9\"><I>%s</I></FONT></TD></TR>\n",
			colspan, tooltip))
	}

	// sub-header row labelling each column
	sb.WriteString("      <TR>")
	for _, h := range []string{"column", "type", "key", "null", "default"} {
		sb.WriteString(fmt.Sprintf("<TD BGCOLOR=\"#ECF0F1\" ALIGN=\"CENTER\"><FONT POINT-SIZE=\"9\" COLOR=\"#7F8C8D\"><B>%s</B></FONT></TD>", h))
	}
	sb.WriteString("</TR>\n")

	// one row per column
	for i, col := range tbl.Columns {
		rowBG := "white"
		if i%2 == 1 {
			rowBG = "#F8F9FA"
		}

		// name cell – with optional comment on a second line
		nameCell := htmlEscape(col.Name)
		if len(col.Comments) > 0 {
			comment := htmlEscape(strings.Join(col.Comments, " "))
			nameCell += fmt.Sprintf("<BR/><FONT POINT-SIZE=\"8\" COLOR=\"#95A5A6\"><I>%s</I></FONT>", comment)
		}

		// type cell
		typeCell := fmt.Sprintf("<FONT COLOR=\"#555555\">%s</FONT>", htmlEscape(parser.FormatType(col)))

		// key cell – badges: PK (red), AI (blue), IDX (green)
		keyCell := keyBadges(col)

		// null cell
		nullCell := nullBadge(col)

		// default cell
		defaultCell := defaultValue(col)

		sb.WriteString(fmt.Sprintf(
			"      <TR>"+
				"<TD ALIGN=\"LEFT\" PORT=\"%s\" BGCOLOR=\"%s\">%s</TD>"+
				"<TD ALIGN=\"LEFT\" BGCOLOR=\"%s\">%s</TD>"+
				"<TD ALIGN=\"CENTER\" BGCOLOR=\"%s\">%s</TD>"+
				"<TD ALIGN=\"CENTER\" BGCOLOR=\"%s\">%s</TD>"+
				"<TD ALIGN=\"LEFT\" BGCOLOR=\"%s\">%s</TD>"+
				"</TR>\n",
			htmlEscape(col.Name), rowBG, nameCell,
			rowBG, typeCell,
			rowBG, keyCell,
			rowBG, nullCell,
			rowBG, defaultCell,
		))
	}

	sb.WriteString("    </TABLE>\n")
	sb.WriteString("  >]\n")
	return sb.String()
}

// keyBadges returns coloured PK / AI / IDX badge markup for the key column.
func keyBadges(col *ast.Column) string {
	var parts []string
	for _, mod := range col.Modifiers {
		switch mod.Kind {
		case ast.ModPrimaryKey:
			parts = append(parts, "<FONT COLOR=\"#C0392B\"><B>PK</B></FONT>")
		case ast.ModAutoIncrement:
			parts = append(parts, "<FONT COLOR=\"#2980B9\"><B>AI</B></FONT>")
		case ast.ModIndexed:
			parts = append(parts, "<FONT COLOR=\"#27AE60\"><B>IDX</B></FONT>")
		}
	}
	return strings.Join(parts, " ")
}

// nullBadge returns a nullability indicator for the null column.
func nullBadge(col *ast.Column) string {
	for _, mod := range col.Modifiers {
		switch mod.Kind {
		case ast.ModNotNull:
			return "<FONT COLOR=\"#E67E22\"><B>NN</B></FONT>"
		case ast.ModNullable:
			return "<FONT COLOR=\"#95A5A6\">NULL</FONT>"
		}
	}
	return ""
}

// defaultValue returns the default value text for the default column.
func defaultValue(col *ast.Column) string {
	for _, mod := range col.Modifiers {
		if mod.Kind == ast.ModDefault {
			return fmt.Sprintf("<FONT COLOR=\"#8E44AD\" POINT-SIZE=\"9\">%s</FONT>", htmlEscape(mod.Value))
		}
	}
	return ""
}

func renderLink(lnk *ast.Link) string {
	fromLabel := cardLabel(lnk.FromCardinality)
	toLabel := cardLabel(lnk.ToCardinality)

	tooltip := ""
	if len(lnk.Comments) > 0 {
		tooltip = strings.Join(lnk.Comments, " ")
	}

	attrs := fmt.Sprintf("taillabel=%q headlabel=%q arrowhead=vee", fromLabel, toLabel)
	if tooltip != "" {
		attrs += fmt.Sprintf(" tooltip=%q", tooltip)
	}

	return fmt.Sprintf("  %s:%s -> %s:%s [%s]\n",
		dotID(lnk.FromTable), dotID(lnk.FromColumn),
		dotID(lnk.ToTable), dotID(lnk.ToColumn),
		attrs,
	)
}

func cardLabel(c ast.Cardinality) string {
	if c == ast.CardMany {
		return "N"
	}
	return "1"
}

// dotID wraps a string in double quotes for safe use as a DOT identifier.
func dotID(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
