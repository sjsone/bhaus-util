package ast

// ConnectionArrow represents the type of C4 connection.
type ConnectionArrow uint8

const (
	ArrowUnidirectional ConnectionArrow = iota // ->
	ArrowBidirectional                         // <->
	ArrowShorthand                             // => (source is implicit parent)
)

func (a ConnectionArrow) String() string {
	switch a {
	case ArrowUnidirectional:
		return "->"
	case ArrowBidirectional:
		return "<->"
	case ArrowShorthand:
		return "=>"
	}
	return "?"
}

// SystemDecl is a SYSTEM declaration in the C4 model.
//
//	SYSTEM MailSystem "Mail Backend":
//	    CONTAINER MTA "Mail Transfer Agent":
//	        COMPONENT SmtpServer:
//	    CONTAINER Database:
type SystemDecl struct {
	Base
	Name        *Ident
	Description string // optional quoted description, without quotes
	Containers  []*ContainerDecl
	Connections []*ConnectionShorthand // implicit-source connections
}

func (*SystemDecl) declNode() {}

// ContainerDecl is a CONTAINER declaration.
//
//	CONTAINER MTA "Mail Transfer Agent":
//	    COMPONENT SmtpServer:
//	    CONNECTION => MailSystem.Database
type ContainerDecl struct {
	Base
	Name        *Ident
	Description string
	Components  []*ComponentDecl
	Connections []*ConnectionShorthand
}

func (*ContainerDecl) declNode() {}

// ComponentDecl is a COMPONENT declaration.
//
//	COMPONENT AuthModule "Auth":
//	    CONNECTION => MailSystem.MTA
type ComponentDecl struct {
	Base
	Name        *Ident
	Description string
	Connections []*ConnectionShorthand
}

func (*ComponentDecl) declNode() {}

// ConnectionDecl is a top-level CONNECTION declaration.
//
//	CONNECTION MailSystem.MTA -> MailSystem.MDA
type ConnectionDecl struct {
	Base
	Source *C4Path
	Arrow  ConnectionArrow
	Target *C4Path
}

func (c *ConnectionDecl) String() string {
	return c.Source.String() + " " + c.Arrow.String() + " " + c.Target.String()
}

func (*ConnectionDecl) declNode() {}

// ConnectionShorthand is a nested CONNECTION => target. The source is the
// enclosing system, container or component (the semantic context).
//
//	CONNECTION => MailSystem.MTA
type ConnectionShorthand struct {
	Base
	Target *C4Path
}
