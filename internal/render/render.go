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

	tooltip := ""
	if len(tbl.Comments) > 0 {
		tooltip = strings.Join(tbl.Comments, " ")
	}

	sb.WriteString(fmt.Sprintf("  %s [label=<\n", dotID(tbl.Name)))
	sb.WriteString("    <TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\" CELLPADDING=\"4\">\n")

	// header row
	headerTitle := htmlEscape(tbl.Name)
	if tooltip != "" {
		sb.WriteString(fmt.Sprintf("      <TR><TD BGCOLOR=\"#AED6F1\" COLSPAN=\"2\"><B>%s</B><BR/><FONT POINT-SIZE=\"9\">%s</FONT></TD></TR>\n",
			headerTitle, htmlEscape(tooltip)))
	} else {
		sb.WriteString(fmt.Sprintf("      <TR><TD BGCOLOR=\"#AED6F1\" COLSPAN=\"2\"><B>%s</B></TD></TR>\n", headerTitle))
	}

	// column rows
	for _, col := range tbl.Columns {
		typeStr := parser.FormatType(col)
		modStr := modifierSummary(col)
		if modStr != "" {
			typeStr = typeStr + " " + modStr
		}
		colComment := ""
		if len(col.Comments) > 0 {
			colComment = " — " + strings.Join(col.Comments, " ")
		}
		sb.WriteString(fmt.Sprintf("      <TR><TD ALIGN=\"LEFT\" PORT=\"%s\" BGCOLOR=\"white\">%s%s</TD><TD ALIGN=\"LEFT\" BGCOLOR=\"white\"><FONT COLOR=\"#555555\">%s</FONT></TD></TR>\n",
			htmlEscape(col.Name),
			htmlEscape(col.Name),
			htmlEscape(colComment),
			htmlEscape(typeStr),
		))
	}

	sb.WriteString("    </TABLE>\n")
	sb.WriteString("  >]\n")
	return sb.String()
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

func modifierSummary(col *ast.Column) string {
	var parts []string
	for _, mod := range col.Modifiers {
		switch mod.Kind {
		case ast.ModPrimaryKey:
			parts = append(parts, "PK")
		case ast.ModNullable:
			parts = append(parts, "nullable")
		case ast.ModNotNull:
			parts = append(parts, "not null")
		case ast.ModAutoIncrement:
			parts = append(parts, "auto")
		case ast.ModIndexed:
			parts = append(parts, "idx")
		case ast.ModDefault:
			parts = append(parts, fmt.Sprintf("default(%s)", mod.Value))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "[" + strings.Join(parts, ", ") + "]"
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
