package cmd

import (
	"errors"
	"fmt"
	"github.com/dave/jennifer/jen"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
	"math"
	"os"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "go-enumerator",
	Short: "Generate enum-like code for Go constants",
	Long: `Generate enum-like code for Go constants. 

go-enumerator is designed to be called by go generate. 
See https://pkg.go.dev/github.com/ajjensen13/go-enumerator for usage examples.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		inputFileName, ok := resolveParameterValue(cmd.Flag("input"), "GOFILE")
		if !ok {
			return errors.New("failed to determine input file")
		}

		pkgName, ok := resolveParameterValue(cmd.Flag("pkg"), "GOPACKAGE")
		if !ok {
			return errors.New("failed to determine package name")
		}

		pkg, err := loadPackage(pkgName, inputFileName)
		if err != nil {
			return err
		}

		typeName, _ := resolveParameterValue(cmd.Flag("type"), "")
		lineStr, _ := resolveParameterValue(cmd.Flag("line"), "GOLINE")
		var line int
		_, err = fmt.Sscan(lineStr, &line)
		if err != nil {
			return err
		}

		tn, err := findTypeDecl(pkg.Fset, pkg.TypesInfo, typeName, inputFileName, line)
		if err != nil {
			return err
		}

		if typeName == "" && tn.Name() != "" {
			typeName = tn.Name()
		}

		receiver, _ := resolveParameterValue(cmd.Flag("receiver"), "")
		if receiver == "" {
			receiver = defaultReceiverName(tn)
		}
		receiver = safeVarName(receiver)

		vs, kind := findConstantsOfType(pkg.Fset, pkg.TypesInfo, tn)
		if len(vs) == 0 {
			return fmt.Errorf("no constants of type %q found", tn.Name())
		}

		f, err := generateEnums(pkgName, tn, vs, kind, receiver)
		if err != nil {
			return err
		}

		outputFileName, ok := resolveParameterValue(cmd.Flag("output"), "")
		if !ok {
			outputFileName = fmt.Sprintf("%s_enum.go", unexportName(typeName))
		}

		out, cleanup, err := openOutputFile(outputFileName)
		if err != nil {
			return err
		}
		defer cleanup()

		return f.Render(out)
	},
}

func init() {
	fs := rootCmd.Flags()
	fs.StringVarP(&flagInput, "input", "i", "", "input file to scan. If not specified, input defaults to the value of $GOFILE, which is set by go generate")
	fs.StringVarP(&flagOutput, "output", "o", "", "output file to create. If not specified, output defaults to the value of <type>_enum.go. As special cases, you can specify <STDOUT> or <STDERR> to output to standard output or standard error")
	fs.StringVarP(&flagPkg, "pkg", "p", "", "package name for the generated file. If not specified, pkg defaults to the value of $GOPACKAGE which is set by go generate")
	fs.StringVarP(&flagType, "type", "t", "", "type name to generate an enum definition for. If not specified, it attempts to find the type using $GOLINE and $GOFILE")
	fs.StringVarP(&flagReceiver, "receiver", "r", "", "receiver variable name of the generated methods. By default, the first letter of the type if used")
	fs.IntVarP(&flagLine, "line", "l", 0, "Use this parameter to specify the line to search for types from if a type name is not specified. If not specified, line defaults to the value of $GOLINE which is set by go generate.")
	_ = fs.MarkHidden("line")
}

var (
	flagInput    string
	flagOutput   string
	flagPkg      string
	flagType     string
	flagReceiver string
	flagLine     int
)

func resolveParameterValue(f *pflag.Flag, env string) (string, bool) {
	if f.Changed {
		return f.Value.String(), true
	}

	if env != "" {
		return os.LookupEnv(env)
	}

	return f.DefValue, false
}

func loadPackage(pkgName, inputFileName string) (*packages.Package, error) {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo}, fmt.Sprintf("file=%s", inputFileName))
	if err != nil {
		return nil, err
	}

	var ret *packages.Package
	for _, pkg := range pkgs {
		if pkg.Name != pkgName {
			continue
		}

		if ret != nil {
			return nil, fmt.Errorf("multiple packages found with name %s", pkgName)
		}

		ret = pkg
	}

	if ret == nil {
		return nil, fmt.Errorf("no packages found with name %s", pkgName)
	}

	return ret, nil
}

func findTypeDecl(fset *token.FileSet, info *types.Info, name, inputFileName string, line int) (*types.TypeName, error) {
	if name != "" {
		return findTypeDeclByName(info, name)
	}

	return findTypeDeclByPosition(fset, info, inputFileName, line)
}

func findTypeDeclByPosition(fset *token.FileSet, info *types.Info, inputFileName string, line int) (*types.TypeName, error) {
	var ret *types.TypeName
	var closestObject types.Object
	closest := math.MaxInt32
	for _, object := range info.Defs {
		if object == nil {
			continue
		}

		p := fset.Position(object.Pos())
		if !sameFile(p.Filename, inputFileName) {
			continue
		}

		if p.Line < line || closest < p.Line {
			continue
		}

		ret = nil // we found something closer than our current closest thing
		closestObject = object

		c, ok := object.(*types.TypeName)
		if !ok {
			continue
		}

		ret = c
		closest = p.Line
	}

	if ret == nil {
		if closestObject != nil {
			return nil, fmt.Errorf("failed to determine type: closest declaration is not a named type: %v", closestObject)
		}
		return nil, fmt.Errorf("failed to determine type")
	}

	return ret, nil
}

func findTypeDeclByName(info *types.Info, name string) (*types.TypeName, error) {
	for _, object := range info.Defs {
		if object == nil {
			continue
		}

		c, ok := object.(*types.TypeName)
		if !ok {
			continue
		}

		if c.Name() != name {
			continue
		}

		return c, nil
	}

	return nil, fmt.Errorf("type %q not found", name)
}

func findConstantsOfType(fset *token.FileSet, info *types.Info, obj types.Object) ([]*types.Const, constant.Kind) {
	var ret []*types.Const
	kind := constant.Unknown
	for _, object := range info.Defs {
		if object == nil {
			continue
		}

		c, ok := object.(*types.Const)
		if !ok {
			continue
		}

		t, ok := c.Type().(*types.Named)
		if !ok {
			continue
		}

		if c.Name() == "_" {
			continue
		}

		if t.Obj() != obj {
			continue
		}

		k := c.Val().Kind()
		if kind == constant.Unknown {
			kind = k
		}

		if kind != k {
			panic("multiple constant kinds found")
		}

		ret = append(ret, c)
	}

	if len(ret) == 0 {
		return nil, constant.Unknown
	}

	// Sort the items based on where they show up in source code.
	// This is mainly to avoid significant differences in version control overtime.
	sort.Slice(ret, func(i, j int) bool {
		ip := fset.Position(ret[i].Pos())
		jp := fset.Position(ret[j].Pos())

		return ip.Filename < jp.Filename ||
			ip.Filename == jp.Filename && ip.Offset < jp.Offset
	})

	return ret, kind
}

func sameFile(a, b string) bool {
	as, err := os.Stat(a)
	if err != nil {
		panic(err)
	}

	bs, err := os.Stat(b)
	if err != nil {
		panic(err)
	}

	return os.SameFile(as, bs)
}

func generateEnums(pkgName string, eType *types.TypeName, cs []*types.Const, kind constant.Kind, receiver string) (f *jen.File, err error) {
	defer func() {
		if r := recover(); r != nil {
			f = nil
			err = r.(error)
		}
	}()

	tokenVarName := safeVarName("token", receiver)
	stringVarName := safeVarName("str", receiver, tokenVarName)
	scanStateVarName := safeVarName("scanState", receiver, tokenVarName, stringVarName)
	verbVarName := safeVarName("verb", receiver, tokenVarName, stringVarName, scanStateVarName)
	xVarName := safeVarName("x", receiver, tokenVarName, stringVarName, scanStateVarName, verbVarName)

	f = jen.NewFile(pkgName)
	f.HeaderComment(fmt.Sprintf("Code generated by %q; DO NOT EDIT.", strings.Join(os.Args, " ")))

	f.Line()
	f.Commentf("String implements fmt.Stringer. If !%s.Defined(), then a generated string is returned based on %s's value.", receiver, receiver)
	switch kind {
	case constant.String:
		f.Func().Params(jen.Id(receiver).Id(eType.Name())).Id("String").Params().String().Block(
			jen.Return(jen.String().Parens(jen.Id(receiver))),
		)
	default:
		f.Func().Params(jen.Id(receiver).Id(eType.Name())).Id("String").Params().String().Block(
			jen.Switch(jen.Id(receiver)).BlockFunc(func(g *jen.Group) {
				for _, c := range cs {
					g.Case(jen.Id(c.Name())).Block(jen.Return(jen.Lit(c.Name())))
				}
			}),
			jen.Return(jen.Qual("fmt", "Sprintf").Call(jen.Lit(fmt.Sprintf("%s(%%d)", eType.Name())), jen.Id(receiver))),
		)
	}

	f.Line()
	f.Commentf("Defined returns true if %s holds a defined value.", receiver)
	f.Func().Params(jen.Id(receiver).Id(eType.Name())).Id("Defined").Params().Bool().Block(
		jen.Switch(jen.Id(receiver)).Block(
			jen.CaseFunc(func(g *jen.Group) {
				for _, c := range cs {
					g.Op(c.Val().ExactString())
				}
			}).Block(jen.Return(jen.True())),
			jen.Default().Block(jen.Return(jen.False())),
		),
	)

	f.Line()
	f.Commentf("Scan implements fmt.Scanner. Use fmt.Scan() to parse strings into %s values", eType.Name())
	f.Func().Params(jen.Id(receiver).Op("*").Id(eType.Name())).Id("Scan").Params(jen.Id(scanStateVarName).Qual("fmt", "ScanState"), jen.Id(verbVarName).Rune()).Error().Block(
		jen.List(jen.Id(tokenVarName), jen.Err()).Op(":=").Id(scanStateVarName).Dot("Token").Call(jen.True(), jen.Nil()),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Err()),
		),

		jen.Line(),
		jen.Switch(jen.String().Parens(jen.Id(tokenVarName))).BlockFunc(func(g *jen.Group) {
			for _, c := range cs {
				g.Case(jen.Lit(c.Name())).Block(
					jen.Op("*").Id(receiver).Op("=").Id(c.Name()),
				)
			}
			g.Default().Block(
				jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("unknown "+eType.Name()+" value: %s"), jen.Id(tokenVarName))),
			)
		}),

		jen.Return(jen.Nil()),
	)

	f.Line()
	f.Commentf("Next returns the next defined %s. If %s is not defined, then Next returns the first defined value.", eType.Name(), receiver)
	f.Commentf("The order that defined values are returned is undefined.")
	f.Commentf("The only guarantee is that all defined values will be returned before Next starts cycling through previous values again.")
	f.Commentf("The order will be consistent for a given program, but the order may change if the program is re-compiled.")
	f.Func().Params(jen.Id(receiver).Id(eType.Name())).Id("Next").Params().Id(eType.Name()).Block(
		jen.Switch(jen.Id(receiver)).BlockFunc(func(g *jen.Group) {
			for i, c := range cs {
				ni := (i + 1) % len(cs)
				g.Case(jen.Id(c.Name())).Block(jen.Return(jen.Id(cs[ni].Name())))
			}
			if len(cs) > 0 {
				g.Default().Block(jen.Return(jen.Id(cs[0].Name())))
			}
		}),
	)

	// f.Line()
	// f.Commentf("First returns the first defined %s. First is a convenience method for initializing Next loops.", eType.Name())
	// f.Func().Params(jen.Id(receiver).Id(eType.Name())).Id("First").Params().Id(eType.Name()).Block(
	// 	jen.Return(jen.Id(cs[0].Name())),
	// )
	//
	// f.Line()
	// f.Commentf("Last returns the last defined %s. Last is a convenience method for terminating Next loops.", eType.Name())
	// f.Func().Params(jen.Id(receiver).Id(eType.Name())).Id("Last").Params().Id(eType.Name()).Block(
	// 	jen.Return(jen.Id(cs[len(cs)-1].Name())),
	// )

	f.Line()
	f.Func().Id("_").Params().BlockFunc(func(g *jen.Group) {
		g.Var().Id(xVarName).Index(jen.Lit(1)).Struct()
		g.Comment(`An "invalid array index" compiler error signifies that the constant values have changed.`)
		g.Commentf(`Re-run the %s command to generate them again.`, os.Args[0])
		for _, c := range cs {
			switch kind {
			case constant.String:
				v := constant.StringVal(c.Val())
				g.Line()
				g.Commentf("Begin %q", v)
				for i, b := range []byte(v) {
					g.Id("_").Op("=").Id(xVarName).Index(jen.LitByte(b).Op("-").Id(c.Name()).Index(jen.Lit(i)))
				}
			default:
				// using jen.Op here is a bit of a hack, but it allows us to
				// insert the string verbatim without surrounding it with a
				// type cast (as Lit does)
				g.Id("_").Op("=").Id(xVarName).Index(jen.Id(c.Name()).Op("-").Op(c.Val().ExactString()))
			}
		}
	})

	f.Line()

	return f, nil
}

func defaultReceiverName(tn *types.TypeName) string {
	s, _ := utf8.DecodeRuneInString(tn.Name())
	return unexportName(string(s))
}

func safeVarName(want string, not ...string) string {
	if token.IsKeyword(want) {
		return safeVarName("_"+want, not...)
	}

	for _, s := range not {
		if want == s {
			return safeVarName("_"+want, not...)
		}
	}

	return want
}

func openOutputFile(name string) (*os.File, func(), error) {
	switch name {
	case "<STDOUT>":
		return os.Stdout, func() {}, nil
	case "<STDERR>":
		return os.Stderr, func() {}, nil
	default:
		ret, err := os.Create(name)
		if err != nil {
			return nil, nil, err
		}
		return ret, func() { _ = ret.Close() }, nil
	}
}

func unexportName(s string) string {
	if !ast.IsExported(s) {
		return s
	}

	start, size := utf8.DecodeRuneInString(s)
	if size == 0 {
		panic("s is empty")
	}

	start = unicode.ToLower(start)
	return string(start) + s[size:]
}
