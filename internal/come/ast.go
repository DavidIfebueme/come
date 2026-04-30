package come

type File struct {
	Path         string
	Declarations []Node
}

type Node interface {
	node()
}

type AppDecl struct {
	Name string
}

func (AppDecl) node() {}

type DBDecl struct {
	Driver     string
	Connection string
	EnvTag     string
}

func (DBDecl) node() {}

type AuraDecl struct {
	Port         int
	ReadTimeout  string
	WriteTimeout string
	IdleTimeout  string
}

func (AuraDecl) node() {}

type CORSDecl struct {
	Origin string
}

func (CORSDecl) node() {}

type BouncerConfigDecl struct {
	Algorithm string
	Secret    string
	Expire    string
}

func (BouncerConfigDecl) node() {}

type BorrowDecl struct {
	Path string
}

func (BorrowDecl) node() {}

type ManifestDecl struct {
	Name    string
	Fields  []FieldDecl
	Indexes []IndexDecl
}

func (ManifestDecl) node() {}

type FieldKind int

const (
	FieldString FieldKind = iota
	FieldInt
	FieldFloat
	FieldBool
	FieldTimestamp
	FieldUUID
	FieldJSON
	FieldBytes
	FieldEnum
	FieldArray
)

type FieldType struct {
	Kind   FieldKind
	Ref    string
	Items  *FieldType
}

type FieldDecl struct {
	Name       string
	Type       FieldType
	Nullable   bool
	Decorators []Decorator
}

type Decorator struct {
	Name string
	Args []string
}

type IndexDecl struct {
	Fields []string
	Unique bool
}

func (IndexDecl) node() {}

type EnumDecl struct {
	Name   string
	Values []string
}

func (EnumDecl) node() {}

type RouteDecl struct {
	Method  string
	Path    string
	Handler string
	Wards   []WardDecl
	Vouch   *VouchDecl
	Grabit  *GrabitDecl
	Bouncer *BouncerBlockDecl
	Hurls   []HurlDecl
}

func (RouteDecl) node() {}

type WardDecl struct {
	Name string
	Args []string
}

type VouchDecl struct {
	Fields []VouchField
}

type VouchField struct {
	Name       string
	Decorators []Decorator
}

type GrabitOp int

const (
	GrabitSelect GrabitOp = iota
	GrabitInsert
	GrabitUpdate
	GrabitDelete
)

type GrabitDecl struct {
	Operation      GrabitOp
	Model          string
	Wheres         []WhereClause
	OrderBy        *ValueSource
	OrderByDefault string
	OrderByAllowed []string
	OrderDir       *ValueSource
	OrderDirDefault string
	Limit          *ValueSource
	LimitDefault   int
	LimitMax       int
	Page           *ValueSource
	PageDefault    int
	Offset         *ValueSource
	OffsetDefault  int
	One            bool
	SetSource      string
}

type WhereClause struct {
	Field  string
	Op     string
	Source ValueSource
}

type SourceKind int

const (
	SourceQuery SourceKind = iota
	SourceParam
	SourceBody
	SourceHeader
	SourceEnv
	SourceResult
	SourceLiteral
)

type ValueSource struct {
	Kind  SourceKind
	Value string
}

type BouncerBlockDecl struct {
	Actions []BouncerAction
}

type BouncerActionKind int

const (
	BouncerSign BouncerActionKind = iota
	BouncerVerify
	BouncerInvalidate
)

type BouncerAction struct {
	Kind   BouncerActionKind
	Fields map[string]ValueSource
}

type HurlKind int

const (
	HurlResult HurlKind = iota
	HurlValidation
	HurlNoContent
	HurlCustom
)

type HurlDecl struct {
	StatusCode int
	Kind       HurlKind
	CustomObj  map[string]string
}

type VibesDecl struct {
	Vars []EnvVarDecl
}

func (VibesDecl) node() {}

type EnvVarDecl struct {
	Name    string
	Source  string
	Req     bool
	Default string
}

type SpawnChaosDecl struct {
	Path   string
	Root   string
	Unique string
	Key    string
}

func (SpawnChaosDecl) node() {}

type RawGoDecl struct {
	Code string
}

func (RawGoDecl) node() {}

type ReshapeDecl struct {
	Up   string
	Down string
}

func (ReshapeDecl) node() {}

type BabbleDecl struct {
	Model  string
	Rules  []BabbleRule
	Ignore []string
}

func (BabbleDecl) node() {}

type BabbleRuleKind int

const (
	BabbleKeyword BabbleRuleKind = iota
	BabblePrefix
)

type BabbleRule struct {
	Kind    BabbleRuleKind
	Words   []string
	Filters map[string]string
	NextAs  string
}

type Project struct {
	AppName  string
	DBs      []DBDecl
	Aura     AuraDecl
	CORS     CORSDecl
	Bouncer  *BouncerConfigDecl
	Vibes    *VibesDecl
	Features []Feature
	Driver   string
}

type Feature struct {
	Name      string
	Manifests []ManifestDecl
	Enums     []EnumDecl
	Routes    []RouteDecl
	Seeds     []SpawnChaosDecl
	RawGo     []RawGoDecl
	Reshapes  []ReshapeDecl
	Babbles   []BabbleDecl
}
