package ast

// Program is the root AST node.
type Program struct {
	Tables []*Table
	Links  []*Link
}

// Table represents a table definition.
type Table struct {
	Name     string
	Comments []string
	Columns  []*Column
	Line     int
}

// Column represents a single column in a table.
type Column struct {
	Name       string
	Type       string
	TypeParams []string
	Modifiers  []Modifier
	Comments   []string
	Line       int
}

// Modifier represents a column modifier.
type Modifier struct {
	Kind  ModifierKind
	Value string // used for default()
}

// ModifierKind identifies a column modifier type.
type ModifierKind int

const (
	ModPrimaryKey    ModifierKind = iota
	ModNullable
	ModNotNull
	ModAutoIncrement
	ModIndexed
	ModDefault
)

// Link represents a relationship between two table columns.
type Link struct {
	FromTable       string
	FromColumn      string
	ToTable         string
	ToColumn        string
	FromCardinality Cardinality
	ToCardinality   Cardinality
	Comments        []string
	Line            int
}

// Cardinality represents the cardinality side of a relationship.
type Cardinality int

const (
	CardOne  Cardinality = iota
	CardMany
)
