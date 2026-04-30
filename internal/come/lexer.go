package come

import (
	"fmt"
	"strings"
	"unicode"
)

type TokenType int

const (
	TokEOF TokenType = iota
	TokIdent
	TokString
	TokInt
	TokFloat
	TokDuration
	TokDecorator
	TokLBrace
	TokRBrace
	TokLParen
	TokRParen
	TokLBrack
	TokRBrack
	TokColon
	TokDot
	TokComma
	TokQuestion
	TokEq
	TokNeq
	TokGte
	TokLte
	TokGt
	TokLt
	TokAssign
	TokArrow
)

var tokNames = map[TokenType]string{
	TokEOF:       "EOF",
	TokIdent:     "identifier",
	TokString:    "string",
	TokInt:       "int",
	TokFloat:     "float",
	TokDuration:  "duration",
	TokDecorator: "decorator",
	TokLBrace:    "{",
	TokRBrace:    "}",
	TokLParen:    "(",
	TokRParen:    ")",
	TokLBrack:    "[",
	TokRBrack:    "]",
	TokColon:     ":",
	TokDot:       ".",
	TokComma:     ",",
	TokQuestion:  "?",
	TokEq:        "==",
	TokNeq:       "!=",
	TokGte:       ">=",
	TokLte:       "<=",
	TokGt:        ">",
	TokLt:        "<",
	TokAssign:    "=",
	TokArrow:     "->",
}

func (t TokenType) String() string {
	if s, ok := tokNames[t]; ok {
		return s
	}
	return fmt.Sprintf("TokenType(%d)", int(t))
}

type Token struct {
	Type TokenType
	Val  string
	Line int
	Col  int
}

func (t Token) String() string {
	if t.Val != "" {
		return fmt.Sprintf("%s(%q)", t.Type, t.Val)
	}
	return t.Type.String()
}

type Lexer struct {
	src  string
	pos  int
	line int
	col  int
}

func NewLexer(src string) *Lexer {
	return &Lexer{src: src, pos: 0, line: 1, col: 1}
}

func (l *Lexer) Tokenize() ([]Token, error) {
	var tokens []Token
	for {
		tok, err := l.nextToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.Type == TokEOF {
			break
		}
	}
	return tokens, nil
}

func (l *Lexer) peek() byte {
	if l.pos >= len(l.src) {
		return 0
	}
	return l.src[l.pos]
}

func (l *Lexer) peekAt(offset int) byte {
	idx := l.pos + offset
	if idx >= len(l.src) {
		return 0
	}
	return l.src[idx]
}

func (l *Lexer) advance() byte {
	ch := l.src[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.src) && (l.src[l.pos] == ' ' || l.src[l.pos] == '\t' || l.src[l.pos] == '\r' || l.src[l.pos] == '\n') {
		l.advance()
	}
}

func (l *Lexer) skipComment() {
	for l.pos < len(l.src) && l.src[l.pos] != '\n' {
		l.advance()
	}
}

func (l *Lexer) nextToken() (Token, error) {
	l.skipWhitespace()
	if l.pos >= len(l.src) {
		return Token{Type: TokEOF, Line: l.line, Col: l.col}, nil
	}
	line, col := l.line, l.col
	ch := l.peek()
	if ch == '#' {
		l.skipComment()
		return l.nextToken()
	}
	if ch == '"' {
		return l.readString(line, col)
	}
	if ch == '@' {
		l.advance()
		start := l.pos
		for l.pos < len(l.src) && (unicode.IsLetter(rune(l.peek())) || unicode.IsDigit(rune(l.peek())) || l.peek() == '_') {
			l.advance()
		}
		if l.pos == start {
			return Token{}, fmt.Errorf("expected decorator name after '@' at line %d col %d", line, col)
		}
		return Token{Type: TokDecorator, Val: l.src[start:l.pos], Line: line, Col: col}, nil
	}
	if unicode.IsDigit(rune(ch)) {
		return l.readNumber(line, col)
	}
	if unicode.IsLetter(rune(ch)) || ch == '_' {
		return l.readIdent(line, col)
	}
	switch ch {
	case '{':
		l.advance()
		return Token{Type: TokLBrace, Val: "{", Line: line, Col: col}, nil
	case '}':
		l.advance()
		return Token{Type: TokRBrace, Val: "}", Line: line, Col: col}, nil
	case '(':
		l.advance()
		return Token{Type: TokLParen, Val: "(", Line: line, Col: col}, nil
	case ')':
		l.advance()
		return Token{Type: TokRParen, Val: ")", Line: line, Col: col}, nil
	case '[':
		l.advance()
		return Token{Type: TokLBrack, Val: "[", Line: line, Col: col}, nil
	case ']':
		l.advance()
		return Token{Type: TokRBrack, Val: "]", Line: line, Col: col}, nil
	case ':':
		l.advance()
		return Token{Type: TokColon, Val: ":", Line: line, Col: col}, nil
	case '.':
		l.advance()
		return Token{Type: TokDot, Val: ".", Line: line, Col: col}, nil
	case ',':
		l.advance()
		return Token{Type: TokComma, Val: ",", Line: line, Col: col}, nil
	case '?':
		l.advance()
		return Token{Type: TokQuestion, Val: "?", Line: line, Col: col}, nil
	case '=':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return Token{Type: TokEq, Val: "==", Line: line, Col: col}, nil
		}
		return Token{Type: TokAssign, Val: "=", Line: line, Col: col}, nil
	case '!':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return Token{Type: TokNeq, Val: "!=", Line: line, Col: col}, nil
		}
		return Token{}, fmt.Errorf("unexpected character '!' at line %d col %d", line, col)
	case '-':
		l.advance()
		if l.peek() == '>' {
			l.advance()
			return Token{Type: TokArrow, Val: "->", Line: line, Col: col}, nil
		}
		return Token{}, fmt.Errorf("unexpected character '-' at line %d col %d", line, col)
	case '>':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return Token{Type: TokGte, Val: ">=", Line: line, Col: col}, nil
		}
		return Token{Type: TokGt, Val: ">", Line: line, Col: col}, nil
	case '<':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return Token{Type: TokLte, Val: "<=", Line: line, Col: col}, nil
		}
		return Token{Type: TokLt, Val: "<", Line: line, Col: col}, nil
	}
	return Token{}, fmt.Errorf("unexpected character %q at line %d col %d", ch, line, col)
}

func (l *Lexer) readString(line, col int) (Token, error) {
	l.advance()
	var sb strings.Builder
	for l.pos < len(l.src) {
		ch := l.advance()
		if ch == '\\' {
			if l.pos < len(l.src) {
				esc := l.advance()
				switch esc {
				case 'n':
					sb.WriteByte('\n')
				case 't':
					sb.WriteByte('\t')
				case 'r':
					sb.WriteByte('\r')
				case '"':
					sb.WriteByte('"')
				case '\\':
					sb.WriteByte('\\')
				default:
					sb.WriteByte('\\')
					sb.WriteByte(esc)
				}
			}
			continue
		}
		if ch == '"' {
			return Token{Type: TokString, Val: sb.String(), Line: line, Col: col}, nil
		}
		sb.WriteByte(ch)
	}
	return Token{}, fmt.Errorf("unterminated string at line %d col %d", line, col)
}

func (l *Lexer) readNumber(line, col int) (Token, error) {
	start := l.pos
	isFloat := false
	for l.pos < len(l.src) && unicode.IsDigit(rune(l.peek())) {
		l.advance()
	}
	if l.pos < len(l.src) && l.peek() == '.' {
		if l.peekAt(1) != 0 && unicode.IsDigit(rune(l.peekAt(1))) {
			isFloat = true
			l.advance()
			for l.pos < len(l.src) && unicode.IsDigit(rune(l.peek())) {
				l.advance()
			}
		}
	}
	durSuffixes := []string{"ns", "us", "ms", "s", "m", "h"}
	for _, suffix := range durSuffixes {
		end := l.pos + len(suffix)
		if end > len(l.src) {
			continue
		}
		if l.src[l.pos:end] != suffix {
			continue
		}
		if end < len(l.src) && (unicode.IsLetter(rune(l.src[end])) || unicode.IsDigit(rune(l.src[end])) || l.src[end] == '_') {
			continue
		}
		for i := 0; i < len(suffix); i++ {
			l.advance()
		}
		return Token{Type: TokDuration, Val: l.src[start:l.pos], Line: line, Col: col}, nil
	}
	if isFloat {
		return Token{Type: TokFloat, Val: l.src[start:l.pos], Line: line, Col: col}, nil
	}
	return Token{Type: TokInt, Val: l.src[start:l.pos], Line: line, Col: col}, nil
}

func (l *Lexer) readIdent(line, col int) (Token, error) {
	start := l.pos
	for l.pos < len(l.src) && (unicode.IsLetter(rune(l.peek())) || unicode.IsDigit(rune(l.peek())) || l.peek() == '_') {
		l.advance()
	}
	return Token{Type: TokIdent, Val: l.src[start:l.pos], Line: line, Col: col}, nil
}

func (l *Lexer) ReadRawBlock() (string, error) {
	l.skipWhitespace()
	if l.peek() != '{' {
		return "", fmt.Errorf("expected '{' after rawgo at line %d", l.line)
	}
	l.advance()
	start := l.pos
	depth := 1
	for l.pos < len(l.src) && depth > 0 {
		ch := l.src[l.pos]
		switch ch {
		case '{':
			depth++
			l.advance()
		case '}':
			depth--
			if depth == 0 {
				content := l.src[start:l.pos]
				l.advance()
				return content, nil
			}
			l.advance()
		case '"':
			l.advance()
			for l.pos < len(l.src) && l.src[l.pos] != '"' {
				if l.src[l.pos] == '\\' {
					l.advance()
				}
				if l.pos < len(l.src) {
					l.advance()
				}
			}
			if l.pos < len(l.src) {
				l.advance()
			}
		case '`':
			l.advance()
			for l.pos < len(l.src) && l.src[l.pos] != '`' {
				l.advance()
			}
			if l.pos < len(l.src) {
				l.advance()
			}
		default:
			l.advance()
		}
	}
	return "", fmt.Errorf("unterminated rawgo block at line %d", l.line)
}
