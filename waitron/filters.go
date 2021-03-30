package waitron

import (
	"regexp"

	"github.com/flosch/pongo2"
	"gopkg.in/yaml.v2"
)

func FilterFromYaml(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {

	s := in.String()

	out := make(map[interface{}]interface{})

	if err := yaml.Unmarshal([]byte(s), out); err != nil {
		return nil, &pongo2.Error{Sender: "filter:from_yaml", OrigError: err}
	}

	return pongo2.AsSafeValue(out), nil
}

type tagRegexReplaceNode struct {
	position *pongo2.Token
	args     []pongo2.IEvaluator
}

func (node tagRegexReplaceNode) Execute(ctx *pongo2.ExecutionContext, writer pongo2.TemplateWriter) *pongo2.Error {

	s, perr := node.args[0].Evaluate(ctx)
	if perr != nil {
		return perr
	}

	rgx, perr := node.args[1].Evaluate(ctx)
	if perr != nil {
		return perr
	}

	rpl, perr := node.args[2].Evaluate(ctx)
	if perr != nil {
		return perr
	}

	re, err := regexp.Compile(rgx.String())

	if err != nil {
		return &pongo2.Error{Sender: "tag:regex_replace", OrigError: err}
	}

	writer.WriteString(pongo2.AsValue(re.ReplaceAllString(s.String(), rpl.String())).String())

	return nil
}

func TagRegexReplace(doc *pongo2.Parser, start *pongo2.Token, arguments *pongo2.Parser) (pongo2.INodeTag, *pongo2.Error) {
	regexReplaceNode := &tagRegexReplaceNode{
		position: start,
	}

	for arguments.Remaining() > 0 {
		node, err := arguments.ParseExpression()
		if err != nil {
			return nil, err
		}
		regexReplaceNode.args = append(regexReplaceNode.args, node)
	}

	return regexReplaceNode, nil
}

func init() {
	pongo2.RegisterFilter("from_yaml", FilterFromYaml)
	pongo2.RegisterTag("regex_replace", TagRegexReplace)
}
