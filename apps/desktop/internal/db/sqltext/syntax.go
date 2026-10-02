package sqltext

type Syntax struct {
	HashComments          bool
	DashCommentNeedsSpace bool
	BackslashEscapes      bool
	EscapeStrings         bool
	DoubleQuotedStrings   bool
	DollarQuotes          bool
	ExecutableComments    bool
}
