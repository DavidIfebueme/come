package come

import (
	"fmt"
	"strings"
)

type Parser struct {
	tokens []Token
	pos    int
	src    string
}

func NewParser(tokens []Token, src string) *Parser {
	return &Parser{tokens: tokens, pos: 0, src: src}
}

func (p *Parser) Parse() (*File, error) {
	var decls []Node
	for !p.atEnd() {
		decl, err := p.parseDeclaration()
		if err != nil {
			return nil, err
		}
		if decl != nil {
			decls = append(decls, decl)
		}
	}
	return &File{Declarations: decls}, nil
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) atEnd() bool {
	return p.peek().Type == TokEOF
}

func (p *Parser) expectType(tt TokenType) (Token, error) {
	tok := p.advance()
	if tok.Type != tt {
		return tok, fmt.Errorf("expected %s, got %s at line %d", tt, tok, tok.Line)
	}
	return tok, nil
}

func (p *Parser) expectIdent(val string) error {
	tok := p.advance()
	if tok.Type != TokIdent || tok.Val != val {
		return fmt.Errorf("expected identifier %q, got %s at line %d", val, tok, tok.Line)
	}
	return nil
}

func (p *Parser) matchType(tt TokenType) bool {
	if p.peek().Type == tt {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) parseDeclaration() (Node, error) {
	tok := p.peek()
	if tok.Type != TokIdent {
		return nil, fmt.Errorf("expected declaration, got %s at line %d", tok, tok.Line)
	}
	switch tok.Val {
	case "nogo":
		return p.parseApp()
	case "pileup":
		return p.parseDB()
	case "aura":
		return p.parseAura()
	case "unblockthehomies":
		return p.parseCORS()
	case "bouncer":
		return p.parseBouncerConfig()
	case "borrow":
		return p.parseBorrow()
	case "manifest":
		return p.parseManifest()
	case "pick":
		return p.parseEnum()
	case "yeet":
		return p.parseRoute()
	case "spawnchaos":
		return p.parseSpawnChaos()
	case "vibes":
		return p.parseVibes()
	case "rawgo":
		return p.parseRawGo()
	case "reshape":
		return p.parseReshape()
	case "babble":
		return p.parseBabble()
	default:
		return nil, fmt.Errorf("unexpected keyword %q at line %d", tok.Val, tok.Line)
	}
}

func (p *Parser) parseApp() (Node, error) {
	p.advance()
	tok, err := p.expectType(TokString)
	if err != nil {
		return nil, err
	}
	return AppDecl{Name: tok.Val}, nil
}

func (p *Parser) parseDB() (Node, error) {
	p.advance()
	driverTok, err := p.expectType(TokIdent)
	if err != nil {
		return nil, err
	}
	connTok, err := p.expectType(TokString)
	if err != nil {
		return nil, err
	}
	decl := DBDecl{
		Driver:     driverTok.Val,
		Connection: connTok.Val,
	}
	if p.peek().Type == TokIdent && p.peek().Val == "env" {
		p.advance()
		if _, err := p.expectType(TokAssign); err != nil {
			return nil, err
		}
		envTok, err := p.expectType(TokIdent)
		if err != nil {
			return nil, err
		}
		decl.EnvTag = envTok.Val
	}
	return decl, nil
}

func (p *Parser) parseAura() (Node, error) {
	p.advance()
	if _, err := p.expectType(TokLBrace); err != nil {
		return nil, err
	}
	decl := AuraDecl{Port: 8080}
	for p.peek().Type != TokRBrace && !p.atEnd() {
		keyTok, err := p.expectType(TokIdent)
		if err != nil {
			return nil, err
		}
		switch keyTok.Val {
		case "port":
			valTok, err := p.expectType(TokInt)
			if err != nil {
				return nil, err
			}
			fmt.Sscanf(valTok.Val, "%d", &decl.Port)
		case "read_timeout":
			valTok, err := p.expectType(TokDuration)
			if err != nil {
				return nil, err
			}
			decl.ReadTimeout = valTok.Val
		case "write_timeout":
			valTok, err := p.expectType(TokDuration)
			if err != nil {
				return nil, err
			}
			decl.WriteTimeout = valTok.Val
		case "idle_timeout":
			valTok, err := p.expectType(TokDuration)
			if err != nil {
				return nil, err
			}
			decl.IdleTimeout = valTok.Val
		default:
			return nil, fmt.Errorf("unknown aura key %q at line %d", keyTok.Val, keyTok.Line)
		}
	}
	if _, err := p.expectType(TokRBrace); err != nil {
		return nil, err
	}
	return decl, nil
}

func (p *Parser) parseCORS() (Node, error) {
	p.advance()
	tok, err := p.expectType(TokString)
	if err != nil {
		return nil, err
	}
	return CORSDecl{Origin: tok.Val}, nil
}

func (p *Parser) parseBouncerConfig() (Node, error) {
	p.advance()
	_, err := p.expectType(TokIdent)
	if err != nil {
		return nil, err
	}
	if _, err := p.expectType(TokLBrace); err != nil {
		return nil, err
	}
	decl := BouncerConfigDecl{Algorithm: "hs256", Expire: "24h"}
	for p.peek().Type != TokRBrace && !p.atEnd() {
		keyTok, err := p.expectType(TokIdent)
		if err != nil {
			return nil, err
		}
		switch keyTok.Val {
		case "secret":
			src, err := p.parseValueSource()
			if err != nil {
				return nil, err
			}
			decl.Secret = src.Value
			if src.Kind == SourceEnv {
				decl.Secret = "env." + src.Value
			}
		case "expire":
			valTok, err := p.expectType(TokDuration)
			if err != nil {
				return nil, err
			}
			decl.Expire = valTok.Val
		case "algorithm":
			valTok, err := p.expectType(TokIdent)
			if err != nil {
				return nil, err
			}
			decl.Algorithm = valTok.Val
		default:
			return nil, fmt.Errorf("unknown bouncer key %q at line %d", keyTok.Val, keyTok.Line)
		}
	}
	if _, err := p.expectType(TokRBrace); err != nil {
		return nil, err
	}
	return decl, nil
}

func (p *Parser) parseBorrow() (Node, error) {
	p.advance()
	tok, err := p.expectType(TokString)
	if err != nil {
		return nil, err
	}
	return BorrowDecl{Path: tok.Val}, nil
}

func (p *Parser) parseManifest() (Node, error) {
	p.advance()
	nameTok, err := p.expectType(TokIdent)
	if err != nil {
		return nil, err
	}
	if _, err := p.expectType(TokLBrace); err != nil {
		return nil, err
	}
	decl := ManifestDecl{Name: nameTok.Val}
	for p.peek().Type != TokRBrace && !p.atEnd() {
		if p.peek().Type == TokIdent && p.peek().Val == "spotlight" {
			p.advance()
			idx, err := p.parseIndexDecl()
			if err != nil {
				return nil, err
			}
			decl.Indexes = append(decl.Indexes, idx)
			continue
		}
		field, err := p.parseFieldDecl()
		if err != nil {
			return nil, err
		}
		decl.Fields = append(decl.Fields, field)
	}
	if _, err := p.expectType(TokRBrace); err != nil {
		return nil, err
	}
	return decl, nil
}

func (p *Parser) parseIndexDecl() (IndexDecl, error) {
	var idx IndexDecl
	if p.peek().Type == TokLParen {
		p.advance()
		idx.Unique = true
	}
	first, err := p.expectType(TokIdent)
	if err != nil {
		return idx, err
	}
	idx.Fields = append(idx.Fields, first.Val)
	for p.peek().Type == TokComma {
		p.advance()
		f, err := p.expectType(TokIdent)
		if err != nil {
			return idx, err
		}
		idx.Fields = append(idx.Fields, f.Val)
	}
	if idx.Unique {
		if _, err := p.expectType(TokRParen); err != nil {
			return idx, err
		}
	}
	return idx, nil
}

func (p *Parser) parseFieldDecl() (FieldDecl, error) {
	nameTok, err := p.expectType(TokIdent)
	if err != nil {
		return FieldDecl{}, err
	}
	ft, err := p.parseFieldType()
	if err != nil {
		return FieldDecl{}, err
	}
	nullable := false
	if p.peek().Type == TokQuestion {
		p.advance()
		nullable = true
	}
	var decs []Decorator
	for p.peek().Type == TokDecorator {
		dec, err := p.parseDecorator()
		if err != nil {
			return FieldDecl{}, err
		}
		decs = append(decs, dec)
	}
	return FieldDecl{Name: nameTok.Val, Type: ft, Nullable: nullable, Decorators: decs}, nil
}

func (p *Parser) parseFieldType() (FieldType, error) {
	tok, err := p.expectType(TokIdent)
	if err != nil {
		return FieldType{}, err
	}
	switch tok.Val {
	case "string":
		return FieldType{Kind: FieldString}, nil
	case "int":
		return FieldType{Kind: FieldInt}, nil
	case "float":
		return FieldType{Kind: FieldFloat}, nil
	case "bool":
		return FieldType{Kind: FieldBool}, nil
	case "timestamp":
		return FieldType{Kind: FieldTimestamp}, nil
	case "uuid":
		return FieldType{Kind: FieldUUID}, nil
	case "json":
		return FieldType{Kind: FieldJSON}, nil
	case "bytes":
		return FieldType{Kind: FieldBytes}, nil
	case "pick":
		refTok, err := p.expectType(TokIdent)
		if err != nil {
			return FieldType{}, err
		}
		return FieldType{Kind: FieldEnum, Ref: refTok.Val}, nil
	case "[]":
		if p.peek().Type != TokIdent {
			return FieldType{}, fmt.Errorf("expected type after [] at line %d", p.peek().Line)
		}
		inner, err := p.parseFieldType()
		if err != nil {
			return FieldType{}, err
		}
		return FieldType{Kind: FieldArray, Items: &inner}, nil
	default:
		return FieldType{}, fmt.Errorf("unknown type %q at line %d", tok.Val, tok.Line)
	}
}

func (p *Parser) parseDecorator() (Decorator, error) {
	tok, err := p.expectType(TokDecorator)
	if err != nil {
		return Decorator{}, err
	}
	dec := Decorator{Name: tok.Val}
	if p.peek().Type == TokLParen {
		p.advance()
		for p.peek().Type != TokRParen && !p.atEnd() {
			arg := p.advance()
			dec.Args = append(dec.Args, arg.Val)
			if p.peek().Type == TokComma {
				p.advance()
			}
		}
		if _, err := p.expectType(TokRParen); err != nil {
			return Decorator{}, err
		}
	}
	return dec, nil
}

func (p *Parser) parseEnum() (Node, error) {
	p.advance()
	nameTok, err := p.expectType(TokIdent)
	if err != nil {
		return nil, err
	}
	if _, err := p.expectType(TokLBrace); err != nil {
		return nil, err
	}
	decl := EnumDecl{Name: nameTok.Val}
	for p.peek().Type != TokRBrace && !p.atEnd() {
		valTok, err := p.expectType(TokIdent)
		if err != nil {
			return nil, err
		}
		decl.Values = append(decl.Values, valTok.Val)
	}
	if _, err := p.expectType(TokRBrace); err != nil {
		return nil, err
	}
	return decl, nil
}

func (p *Parser) parseRoute() (Node, error) {
	p.advance()
	methodTok, err := p.expectType(TokIdent)
	if err != nil {
		return nil, err
	}
	pathTok, err := p.expectType(TokString)
	if err != nil {
		return nil, err
	}
	handlerTok, err := p.expectType(TokIdent)
	if err != nil {
		return nil, err
	}
	decl := RouteDecl{
		Method:  methodTok.Val,
		Path:    pathTok.Val,
		Handler: handlerTok.Val,
	}
	if p.peek().Type != TokLBrace {
		return decl, nil
	}
	p.advance()
	for p.peek().Type != TokRBrace && !p.atEnd() {
		tok := p.peek()
		if tok.Type != TokIdent {
			return nil, fmt.Errorf("expected ward/vouch/grabit/bouncer/hurl in route block, got %s at line %d", tok, tok.Line)
		}
		switch tok.Val {
		case "ward":
			w, err := p.parseWard()
			if err != nil {
				return nil, err
			}
			decl.Wards = append(decl.Wards, w)
		case "vouch":
			v, err := p.parseVouch()
			if err != nil {
				return nil, err
			}
			decl.Vouch = &v
		case "grabit":
			g, err := p.parseGrabit()
			if err != nil {
				return nil, err
			}
			decl.Grabit = &g
		case "bouncer":
			b, err := p.parseBouncerBlock()
			if err != nil {
				return nil, err
			}
			decl.Bouncer = &b
		case "hurl":
			h, err := p.parseHurl()
			if err != nil {
				return nil, err
			}
			decl.Hurls = append(decl.Hurls, h)
		case "rawgo":
			r, err := p.parseRawGo()
			if err != nil {
				return nil, err
			}
			_ = r
		default:
			return nil, fmt.Errorf("unexpected keyword %q in route block at line %d", tok.Val, tok.Line)
		}
	}
	if _, err := p.expectType(TokRBrace); err != nil {
		return nil, err
	}
	return decl, nil
}

func (p *Parser) parseWard() (WardDecl, error) {
	p.advance()
	nameTok, err := p.expectType(TokIdent)
	if err != nil {
		return WardDecl{}, err
	}
	ward := WardDecl{Name: nameTok.Val}
	for p.peek().Type == TokDecorator {
		dec, err := p.parseDecorator()
		if err != nil {
			return WardDecl{}, err
		}
		ward.Args = append(ward.Args, dec.Name)
		if len(dec.Args) > 0 {
			ward.Args = append(ward.Args, dec.Args...)
		}
	}
	return ward, nil
}

func (p *Parser) parseVouch() (VouchDecl, error) {
	p.advance()
	if _, err := p.expectType(TokLBrace); err != nil {
		return VouchDecl{}, err
	}
	decl := VouchDecl{}
	for p.peek().Type != TokRBrace && !p.atEnd() {
		nameTok, err := p.expectType(TokIdent)
		if err != nil {
			return VouchDecl{}, err
		}
		var decs []Decorator
		for p.peek().Type == TokDecorator {
			dec, err := p.parseDecorator()
			if err != nil {
				return VouchDecl{}, err
			}
			decs = append(decs, dec)
		}
		decl.Fields = append(decl.Fields, VouchField{Name: nameTok.Val, Decorators: decs})
	}
	if _, err := p.expectType(TokRBrace); err != nil {
		return VouchDecl{}, err
	}
	return decl, nil
}

func (p *Parser) parseGrabit() (GrabitDecl, error) {
	p.advance()
	if _, err := p.expectType(TokLBrace); err != nil {
		return GrabitDecl{}, err
	}
	decl := GrabitDecl{}
	for p.peek().Type != TokRBrace && !p.atEnd() {
		tok := p.peek()
		if tok.Type != TokIdent {
			return GrabitDecl{}, fmt.Errorf("expected grabit keyword, got %s at line %d", tok, tok.Line)
		}
		switch tok.Val {
		case "from":
			p.advance()
			modelTok, err := p.expectType(TokIdent)
			if err != nil {
				return GrabitDecl{}, err
			}
			decl.Operation = GrabitSelect
			decl.Model = modelTok.Val
		case "insert":
			p.advance()
			modelTok, err := p.expectType(TokIdent)
			if err != nil {
				return GrabitDecl{}, err
			}
			decl.Operation = GrabitInsert
			decl.Model = modelTok.Val
			src, err := p.parseValueSource()
			if err != nil {
				return GrabitDecl{}, err
			}
			if src.Kind == SourceLiteral {
				decl.SetSource = src.Value
			}
		case "update":
			p.advance()
			modelTok, err := p.expectType(TokIdent)
			if err != nil {
				return GrabitDecl{}, err
			}
			decl.Operation = GrabitUpdate
			decl.Model = modelTok.Val
		case "delete":
			p.advance()
			p.advance()
			modelTok, err := p.expectType(TokIdent)
			if err != nil {
				return GrabitDecl{}, err
			}
			decl.Operation = GrabitDelete
			decl.Model = modelTok.Val
		case "where":
			p.advance()
			w, err := p.parseWhereClause()
			if err != nil {
				return GrabitDecl{}, err
			}
			decl.Wheres = append(decl.Wheres, w)
		case "set":
			p.advance()
			src, err := p.parseValueSource()
			if err != nil {
				return GrabitDecl{}, err
			}
			if src.Kind == SourceLiteral {
				decl.SetSource = src.Value
			}
		case "order_by":
			p.advance()
			src, err := p.parseValueSource()
			if err != nil {
				return GrabitDecl{}, err
			}
			decl.OrderBy = &src
			for p.peek().Type == TokDecorator {
				dec, err := p.parseDecorator()
				if err != nil {
					return GrabitDecl{}, err
				}
				if dec.Name == "default" && len(dec.Args) > 0 {
					decl.OrderByDefault = dec.Args[0]
				}
				if dec.Name == "oneof" {
					decl.OrderByAllowed = dec.Args
				}
			}
		case "order_dir":
			p.advance()
			src, err := p.parseValueSource()
			if err != nil {
				return GrabitDecl{}, err
			}
			decl.OrderDir = &src
			for p.peek().Type == TokDecorator {
				dec, err := p.parseDecorator()
				if err != nil {
					return GrabitDecl{}, err
				}
				if dec.Name == "default" && len(dec.Args) > 0 {
					decl.OrderDirDefault = dec.Args[0]
				}
			}
		case "limit":
			p.advance()
			src, err := p.parseValueSource()
			if err != nil {
				return GrabitDecl{}, err
			}
			decl.Limit = &src
			for p.peek().Type == TokDecorator {
				dec, err := p.parseDecorator()
				if err != nil {
					return GrabitDecl{}, err
				}
				if dec.Name == "default" && len(dec.Args) > 0 {
					fmt.Sscanf(dec.Args[0], "%d", &decl.LimitDefault)
				}
				if dec.Name == "max" && len(dec.Args) > 0 {
					fmt.Sscanf(dec.Args[0], "%d", &decl.LimitMax)
				}
			}
		case "offset":
			p.advance()
			src, err := p.parseValueSource()
			if err != nil {
				return GrabitDecl{}, err
			}
			decl.Offset = &src
			for p.peek().Type == TokDecorator {
				dec, err := p.parseDecorator()
				if err != nil {
					return GrabitDecl{}, err
				}
				if dec.Name == "default" && len(dec.Args) > 0 {
					fmt.Sscanf(dec.Args[0], "%d", &decl.OffsetDefault)
				}
			}
		case "page":
			p.advance()
			src, err := p.parseValueSource()
			if err != nil {
				return GrabitDecl{}, err
			}
			decl.Page = &src
			for p.peek().Type == TokDecorator {
				dec, err := p.parseDecorator()
				if err != nil {
					return GrabitDecl{}, err
				}
				if dec.Name == "default" && len(dec.Args) > 0 {
					fmt.Sscanf(dec.Args[0], "%d", &decl.PageDefault)
				}
			}
		case "one":
			p.advance()
			decl.One = true
		default:
			return GrabitDecl{}, fmt.Errorf("unexpected grabit keyword %q at line %d", tok.Val, tok.Line)
		}
	}
	if _, err := p.expectType(TokRBrace); err != nil {
		return GrabitDecl{}, err
	}
	return decl, nil
}

func (p *Parser) parseWhereClause() (WhereClause, error) {
	fieldTok, err := p.expectType(TokIdent)
	if err != nil {
		return WhereClause{}, err
	}
	op, err := p.parseOperator()
	if err != nil {
		return WhereClause{}, err
	}
	src, err := p.parseValueSource()
	if err != nil {
		return WhereClause{}, err
	}
	return WhereClause{Field: fieldTok.Val, Op: op, Source: src}, nil
}

func (p *Parser) parseOperator() (string, error) {
	tok := p.advance()
	switch tok.Type {
	case TokEq:
		return "==", nil
	case TokNeq:
		return "!=", nil
	case TokGte:
		return ">=", nil
	case TokLte:
		return "<=", nil
	case TokGt:
		return ">", nil
	case TokLt:
		return "<", nil
	case TokIdent:
		if tok.Val == "ilike" {
			return "ilike", nil
		}
		return "", fmt.Errorf("expected operator, got %q at line %d", tok.Val, tok.Line)
	default:
		return "", fmt.Errorf("expected operator, got %s at line %d", tok, tok.Line)
	}
}

func (p *Parser) parseValueSource() (ValueSource, error) {
	tok := p.peek()
	switch tok.Type {
	case TokInt:
		p.advance()
		return ValueSource{Kind: SourceLiteral, Value: tok.Val}, nil
	case TokFloat:
		p.advance()
		return ValueSource{Kind: SourceLiteral, Value: tok.Val}, nil
	case TokString:
		p.advance()
		return ValueSource{Kind: SourceLiteral, Value: tok.Val}, nil
	case TokDuration:
		p.advance()
		return ValueSource{Kind: SourceLiteral, Value: tok.Val}, nil
	case TokIdent:
		p.advance()
		if p.peek().Type == TokDot {
			root := tok.Val
			p.advance()
			fieldTok, err := p.expectType(TokIdent)
			if err != nil {
				return ValueSource{}, err
			}
			switch root {
			case "query":
				return ValueSource{Kind: SourceQuery, Value: fieldTok.Val}, nil
			case "param":
				return ValueSource{Kind: SourceParam, Value: fieldTok.Val}, nil
			case "body":
				return ValueSource{Kind: SourceBody, Value: fieldTok.Val}, nil
			case "header":
				return ValueSource{Kind: SourceHeader, Value: fieldTok.Val}, nil
			case "env":
				return ValueSource{Kind: SourceEnv, Value: fieldTok.Val}, nil
			case "result":
				return ValueSource{Kind: SourceResult, Value: fieldTok.Val}, nil
			default:
				return ValueSource{}, fmt.Errorf("unknown source prefix %q at line %d", root, tok.Line)
			}
		}
		switch tok.Val {
		case "body":
			return ValueSource{Kind: SourceBody, Value: ""}, nil
		case "query":
			return ValueSource{Kind: SourceQuery, Value: ""}, nil
		case "param":
			return ValueSource{Kind: SourceParam, Value: ""}, nil
		case "result":
			return ValueSource{Kind: SourceResult, Value: ""}, nil
		default:
			return ValueSource{Kind: SourceLiteral, Value: tok.Val}, nil
		}
	default:
		return ValueSource{}, fmt.Errorf("expected value source, got %s at line %d", tok, tok.Line)
	}
}

func (p *Parser) parseBouncerBlock() (BouncerBlockDecl, error) {
	p.advance()
	actionTok, err := p.expectType(TokIdent)
	if err != nil {
		return BouncerBlockDecl{}, err
	}
	var kind BouncerActionKind
	switch actionTok.Val {
	case "sign":
		kind = BouncerSign
	case "verify":
		kind = BouncerVerify
	case "invalidate":
		action := BouncerAction{Kind: BouncerInvalidate, Fields: map[string]ValueSource{}}
		return BouncerBlockDecl{Actions: []BouncerAction{action}}, nil
	default:
		return BouncerBlockDecl{}, fmt.Errorf("expected sign/verify/invalidate, got %q at line %d", actionTok.Val, actionTok.Line)
	}
	if p.peek().Type != TokLBrace {
		return BouncerBlockDecl{}, fmt.Errorf("expected '{' after bouncer %s at line %d", actionTok.Val, actionTok.Line)
	}
	p.advance()
	fields := map[string]ValueSource{}
	for p.peek().Type != TokRBrace && !p.atEnd() {
		keyTok, err := p.expectType(TokIdent)
		if err != nil {
			return BouncerBlockDecl{}, err
		}
		src, err := p.parseValueSource()
		if err != nil {
			return BouncerBlockDecl{}, err
		}
		fields[keyTok.Val] = src
	}
	if _, err := p.expectType(TokRBrace); err != nil {
		return BouncerBlockDecl{}, err
	}
	return BouncerBlockDecl{Actions: []BouncerAction{{Kind: kind, Fields: fields}}}, nil
}

func (p *Parser) parseHurl() (HurlDecl, error) {
	p.advance()
	codeTok, err := p.expectType(TokInt)
	if err != nil {
		return HurlDecl{}, err
	}
	var code int
	fmt.Sscanf(codeTok.Val, "%d", &code)
	decl := HurlDecl{StatusCode: code}
	tok := p.peek()
	switch tok.Type {
	case TokIdent:
		switch tok.Val {
		case "result":
			p.advance()
			decl.Kind = HurlResult
		case "validation":
			p.advance()
			decl.Kind = HurlValidation
		default:
			decl.Kind = HurlCustom
		}
	case TokLBrace:
		p.advance()
		decl.Kind = HurlCustom
		obj, err := p.parseHurlObject()
		if err != nil {
			return HurlDecl{}, err
		}
		decl.CustomObj = obj
	default:
		decl.Kind = HurlNoContent
	}
	return decl, nil
}

func (p *Parser) parseHurlObject() (map[string]string, error) {
	obj := map[string]string{}
	for p.peek().Type != TokRBrace && !p.atEnd() {
		keyTok, err := p.expectType(TokIdent)
		if err != nil {
			return nil, err
		}
		if _, err := p.expectType(TokColon); err != nil {
			return nil, err
		}
		valTok, err := p.expectType(TokString)
		if err != nil {
			return nil, err
		}
		obj[keyTok.Val] = valTok.Val
		if p.peek().Type == TokComma {
			p.advance()
		}
	}
	if _, err := p.expectType(TokRBrace); err != nil {
		return nil, err
	}
	return obj, nil
}

func (p *Parser) parseSpawnChaos() (Node, error) {
	p.advance()
	pathTok, err := p.expectType(TokString)
	if err != nil {
		return nil, err
	}
	if err := p.expectIdent("root"); err != nil {
		return nil, err
	}
	rootTok, err := p.expectType(TokString)
	if err != nil {
		return nil, err
	}
	if err := p.expectIdent("unique"); err != nil {
		return nil, err
	}
	uniqueTok, err := p.expectType(TokString)
	if err != nil {
		return nil, err
	}
	decl := SpawnChaosDecl{Path: pathTok.Val, Root: rootTok.Val, Unique: uniqueTok.Val}
	if p.peek().Type == TokIdent && p.peek().Val == "key" {
		p.advance()
		keyTok, err := p.expectType(TokString)
		if err != nil {
			return nil, err
		}
		decl.Key = keyTok.Val
	}
	return decl, nil
}

func (p *Parser) parseVibes() (Node, error) {
	p.advance()
	if _, err := p.expectType(TokLBrace); err != nil {
		return nil, err
	}
	decl := VibesDecl{}
	for p.peek().Type != TokRBrace && !p.atEnd() {
		nameTok, err := p.expectType(TokIdent)
		if err != nil {
			return nil, err
		}
		src, err := p.parseValueSource()
		if err != nil {
			return nil, err
		}
		ev := EnvVarDecl{Name: nameTok.Val, Source: src.Value}
		if src.Kind == SourceEnv {
			ev.Source = "env." + src.Value
		}
		for p.peek().Type == TokDecorator {
			dec, err := p.parseDecorator()
			if err != nil {
				return nil, err
			}
			switch dec.Name {
			case "required":
				ev.Req = true
			case "default":
				if len(dec.Args) > 0 {
					ev.Default = dec.Args[0]
				}
			}
		}
		decl.Vars = append(decl.Vars, ev)
	}
	if _, err := p.expectType(TokRBrace); err != nil {
		return nil, err
	}
	return decl, nil
}

func (p *Parser) parseRawGo() (Node, error) {
	p.advance()
	lexer := NewLexer("")
	lexer.src = p.src
	lexer.pos = p.pos
	lexer.line = p.tokens[p.pos-1].Line
	content, err := lexer.ReadRawBlock()
	if err != nil {
		return nil, err
	}
	p.pos = lexer.pos
	return RawGoDecl{Code: content}, nil
}

func (p *Parser) parseReshape() (Node, error) {
	p.advance()
	if err := p.expectIdent("up"); err != nil {
		return nil, err
	}
	upTok, err := p.expectType(TokString)
	if err != nil {
		return nil, err
	}
	if err := p.expectIdent("down"); err != nil {
		return nil, err
	}
	downTok, err := p.expectType(TokString)
	if err != nil {
		return nil, err
	}
	return ReshapeDecl{Up: upTok.Val, Down: downTok.Val}, nil
}

func (p *Parser) parseBabble() (Node, error) {
	p.advance()
	modelTok, err := p.expectType(TokIdent)
	if err != nil {
		return nil, err
	}
	if _, err := p.expectType(TokLBrace); err != nil {
		return nil, err
	}
	decl := BabbleDecl{Model: modelTok.Val}
	for p.peek().Type != TokRBrace && !p.atEnd() {
		tok := p.peek()
		if tok.Type != TokIdent {
			return nil, fmt.Errorf("expected keyword/prefix/ignore in babble block, got %s at line %d", tok, tok.Line)
		}
		switch tok.Val {
		case "keyword":
			p.advance()
			rule, err := p.parseBabbleKeyword()
			if err != nil {
				return nil, err
			}
			decl.Rules = append(decl.Rules, rule)
		case "prefix":
			p.advance()
			rule, err := p.parseBabblePrefix()
			if err != nil {
				return nil, err
			}
			decl.Rules = append(decl.Rules, rule)
		case "ignore":
			p.advance()
			words, err := p.parseBabbleWords()
			if err != nil {
				return nil, err
			}
			decl.Ignore = append(decl.Ignore, words...)
		default:
			return nil, fmt.Errorf("unexpected keyword %q in babble block at line %d", tok.Val, tok.Line)
		}
	}
	if _, err := p.expectType(TokRBrace); err != nil {
		return nil, err
	}
	return decl, nil
}

func (p *Parser) parseBabbleWords() ([]string, error) {
	tok, err := p.expectType(TokString)
	if err != nil {
		return nil, err
	}
	return strings.Fields(tok.Val), nil
}

func (p *Parser) parseBabbleKeyword() (BabbleRule, error) {
	words, err := p.parseBabbleWords()
	if err != nil {
		return BabbleRule{}, err
	}
	if _, err := p.expectType(TokArrow); err != nil {
		return BabbleRule{}, err
	}
	filters, err := p.parseBabbleFilters()
	if err != nil {
		return BabbleRule{}, err
	}
	return BabbleRule{Kind: BabbleKeyword, Words: words, Filters: filters}, nil
}

func (p *Parser) parseBabblePrefix() (BabbleRule, error) {
	words, err := p.parseBabbleWords()
	if err != nil {
		return BabbleRule{}, err
	}
	if _, err := p.expectType(TokArrow); err != nil {
		return BabbleRule{}, err
	}
	fieldTok, err := p.expectType(TokIdent)
	if err != nil {
		return BabbleRule{}, err
	}
	if _, err := p.expectType(TokAssign); err != nil {
		return BabbleRule{}, err
	}
	nextAsTok, err := p.expectType(TokIdent)
	if err != nil {
		return BabbleRule{}, err
	}
	return BabbleRule{Kind: BabblePrefix, Words: words, Filters: map[string]string{fieldTok.Val: nextAsTok.Val}}, nil
}

func (p *Parser) parseBabbleFilters() (map[string]string, error) {
	filters := map[string]string{}
	for {
		if p.peek().Type != TokIdent {
			break
		}
		if p.peek().Val == "keyword" || p.peek().Val == "prefix" || p.peek().Val == "ignore" {
			break
		}
		fieldTok, err := p.expectType(TokIdent)
		if err != nil {
			return nil, err
		}
		if _, err := p.expectType(TokAssign); err != nil {
			return nil, err
		}
		valTok := p.advance()
		filters[fieldTok.Val] = valTok.Val
		if p.peek().Type == TokComma {
			p.advance()
		}
	}
	return filters, nil
}
