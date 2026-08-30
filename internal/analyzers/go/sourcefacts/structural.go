package sourcefacts

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"

	"github.com/buffo/arch-view/internal/analysis"
)

// structuralFacts contains only counts that can be observed directly from a
// Go syntax tree. The quality layer turns these counts into advisory signals;
// it does not infer design intent from names or paths.
type structuralFacts struct {
	memberCount             int
	methodCount             int
	dependencyCount         int
	concreteDependencyCount int
	interfaceMethodCount    int
	typeSwitchCount         int
	hierarchyDepth          int
	derivedTypeCount        int
	abstractionCount        int
}

type goTypeDefinition struct {
	name       string
	position   token.Pos
	kind       string
	structType *ast.StructType
	interfaceT *ast.InterfaceType
}

type goDependency struct {
	interfaceType bool
}

// collectStructuralFacts parses the same source bytes as the Tree-sitter
// extractor and indexes facts by the declaration identity used by
// declarationTargets. go/parser supplies typed syntax relationships (fields,
// receivers, interfaces, and type switches) without making the source-index
// core language-aware.
func collectStructuralFacts(path string, content []byte) map[string]structuralFacts {
	fileSet := token.NewFileSet()
	file, _ := parser.ParseFile(fileSet, path, content, parser.AllErrors)
	if file == nil {
		return map[string]structuralFacts{}
	}

	types := make(map[string]goTypeDefinition)
	interfaces := make(map[string]bool)
	typeEmbeds := make(map[string][]string)
	methodCounts := make(map[string]int)

	for _, declaration := range file.Decls {
		gen, ok := declaration.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name == nil {
				continue
			}
			definition := goTypeDefinition{
				name:     typeSpec.Name.Name,
				position: typeSpec.Name.Pos(),
				kind:     goASTTypeKind(typeSpec),
			}
			switch typeNode := typeSpec.Type.(type) {
			case *ast.StructType:
				definition.structType = typeNode
			case *ast.InterfaceType:
				definition.interfaceT = typeNode
			}
			types[definition.name] = definition
			interfaces[definition.name] = definition.interfaceT != nil
			if definition.structType != nil {
				typeEmbeds[definition.name] = embeddedTypeNames(definition.structType.Fields)
			} else if definition.interfaceT != nil {
				typeEmbeds[definition.name] = embeddedTypeNames(definition.interfaceT.Methods)
			}
		}
	}

	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil {
			continue
		}
		if receiver := receiverTypeName(function.Recv); receiver != "" {
			methodCounts[receiver]++
		}
	}

	result := make(map[string]structuralFacts)
	for _, declaration := range file.Decls {
		switch value := declaration.(type) {
		case *ast.FuncDecl:
			if value.Name == nil {
				continue
			}
			kind := "go:function"
			if value.Recv != nil {
				kind = "go:method"
			}
			result[astSymbolKey(fileSet, kind, value.Name.Name, value.Name.Pos())] = factsForFunction(value, interfaces)
		case *ast.GenDecl:
			for _, spec := range value.Specs {
				switch declarationSpec := spec.(type) {
				case *ast.TypeSpec:
					if declarationSpec.Name == nil {
						continue
					}
					definition := types[declarationSpec.Name.Name]
					facts := factsForType(definition, methodCounts, typeEmbeds, interfaces)
					result[astSymbolKey(fileSet, definition.kind, definition.name, definition.position)] = facts
				case *ast.ValueSpec:
					kind := "go:constant"
					if value.Tok == token.VAR {
						kind = "go:variable"
					}
					for _, name := range declarationSpec.Names {
						if name == nil || name.Name == "_" {
							continue
						}
						result[astSymbolKey(fileSet, kind, name.Name, name.Pos())] = factsForValue(declarationSpec, interfaces)
					}
				}
			}
		}
	}
	return result
}

func applyStructuralFacts(symbol *analysis.SymbolRecord, facts structuralFacts) {
	if symbol == nil {
		return
	}
	symbol.MemberCount = intPointer(facts.memberCount)
	symbol.MethodCount = intPointer(facts.methodCount)
	symbol.DependencyCount = intPointer(facts.dependencyCount)
	symbol.ConcreteDependencyCount = intPointer(facts.concreteDependencyCount)
	symbol.InterfaceMethodCount = intPointer(facts.interfaceMethodCount)
	symbol.TypeSwitchCount = intPointer(facts.typeSwitchCount)
	symbol.HierarchyDepth = intPointer(facts.hierarchyDepth)
	symbol.DerivedTypeCount = intPointer(facts.derivedTypeCount)
	symbol.AbstractionCount = intPointer(facts.abstractionCount)
}

func goASTTypeKind(spec *ast.TypeSpec) string {
	kind := "go:type"
	if spec.Assign.IsValid() {
		kind = "go:type-alias"
	}
	switch spec.Type.(type) {
	case *ast.StructType:
		return "go:struct"
	case *ast.InterfaceType:
		return "go:interface"
	default:
		return kind
	}
}

func factsForFunction(function *ast.FuncDecl, interfaces map[string]bool) structuralFacts {
	result := structuralFacts{}
	if function == nil {
		return result
	}
	if function.Type != nil {
		dependencies := make(map[string]goDependency)
		collectFunctionDependencies(function.Type, interfaces, dependencies)
		applyDependencies(&result, dependencies)
	}
	if function.Body != nil {
		ast.Inspect(function.Body, func(node ast.Node) bool {
			if _, ok := node.(*ast.TypeSwitchStmt); ok {
				result.typeSwitchCount++
			}
			return true
		})
	}
	return result
}

func factsForValue(value *ast.ValueSpec, interfaces map[string]bool) structuralFacts {
	result := structuralFacts{}
	if value == nil {
		return result
	}
	dependencies := make(map[string]goDependency)
	collectTypeDependencies(value.Type, interfaces, dependencies)
	for _, expression := range value.Values {
		collectExpressionDependencies(expression, interfaces, dependencies)
	}
	applyDependencies(&result, dependencies)
	return result
}

func factsForType(definition goTypeDefinition, methodCounts map[string]int, typeEmbeds map[string][]string, interfaces map[string]bool) structuralFacts {
	result := structuralFacts{methodCount: methodCounts[definition.name]}
	dependencies := make(map[string]goDependency)
	switch {
	case definition.structType != nil:
		result.memberCount = fieldCount(definition.structType.Fields) + result.methodCount
		collectFieldListDependencies(definition.structType.Fields, interfaces, dependencies)
	case definition.interfaceT != nil:
		result.memberCount = fieldCount(definition.interfaceT.Methods)
		result.interfaceMethodCount = fieldCount(definition.interfaceT.Methods)
		result.abstractionCount = 1
		collectFieldListDependencies(definition.interfaceT.Methods, interfaces, dependencies)
	}
	result.hierarchyDepth = typeHierarchyDepth(definition.name, typeEmbeds, map[string]bool{})
	for _, embedded := range typeEmbeds {
		for _, name := range embedded {
			if name == definition.name {
				result.derivedTypeCount++
			}
		}
	}
	applyDependencies(&result, dependencies)
	result.abstractionCount += interfaceDependencyCount(dependencies)
	return result
}

func collectFunctionDependencies(function *ast.FuncType, interfaces map[string]bool, dependencies map[string]goDependency) {
	if function == nil {
		return
	}
	collectFieldListDependencies(function.Params, interfaces, dependencies)
	collectFieldListDependencies(function.Results, interfaces, dependencies)
}

func collectFieldListDependencies(fields *ast.FieldList, interfaces map[string]bool, dependencies map[string]goDependency) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		if field != nil {
			collectTypeDependencies(field.Type, interfaces, dependencies)
		}
	}
}

func collectTypeDependencies(expression ast.Expr, interfaces map[string]bool, dependencies map[string]goDependency) {
	switch value := expression.(type) {
	case nil:
		return
	case *ast.Ident:
		if !goBuiltinType(value.Name) {
			dependencies[value.Name] = goDependency{interfaceType: interfaces[value.Name]}
		}
	case *ast.SelectorExpr:
		if value.Sel != nil && !goBuiltinType(value.Sel.Name) {
			dependencies[value.Sel.Name] = goDependency{interfaceType: interfaces[value.Sel.Name]}
		}
	case *ast.StarExpr:
		collectTypeDependencies(value.X, interfaces, dependencies)
	case *ast.ArrayType:
		collectTypeDependencies(value.Elt, interfaces, dependencies)
	case *ast.MapType:
		collectTypeDependencies(value.Key, interfaces, dependencies)
		collectTypeDependencies(value.Value, interfaces, dependencies)
	case *ast.ChanType:
		collectTypeDependencies(value.Value, interfaces, dependencies)
	case *ast.Ellipsis:
		collectTypeDependencies(value.Elt, interfaces, dependencies)
	case *ast.FuncType:
		collectFunctionDependencies(value, interfaces, dependencies)
	case *ast.StructType:
		collectFieldListDependencies(value.Fields, interfaces, dependencies)
	case *ast.InterfaceType:
		collectFieldListDependencies(value.Methods, interfaces, dependencies)
	case *ast.IndexExpr:
		collectTypeDependencies(value.X, interfaces, dependencies)
		collectTypeDependencies(value.Index, interfaces, dependencies)
	case *ast.IndexListExpr:
		collectTypeDependencies(value.X, interfaces, dependencies)
		for _, index := range value.Indices {
			collectTypeDependencies(index, interfaces, dependencies)
		}
	case *ast.ParenExpr:
		collectTypeDependencies(value.X, interfaces, dependencies)
	case *ast.UnaryExpr:
		collectTypeDependencies(value.X, interfaces, dependencies)
	case *ast.BinaryExpr:
		collectTypeDependencies(value.X, interfaces, dependencies)
		collectTypeDependencies(value.Y, interfaces, dependencies)
	}
}

func collectExpressionDependencies(expression ast.Expr, interfaces map[string]bool, dependencies map[string]goDependency) {
	ast.Inspect(expression, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.CompositeLit:
			collectTypeDependencies(value.Type, interfaces, dependencies)
			return false
		case *ast.CallExpr:
			collectTypeDependencies(value.Fun, interfaces, dependencies)
			return false
		}
		return true
	})
}

func applyDependencies(result *structuralFacts, dependencies map[string]goDependency) {
	result.dependencyCount = len(dependencies)
	for _, dependency := range dependencies {
		if !dependency.interfaceType {
			result.concreteDependencyCount++
		}
	}
}

func interfaceDependencyCount(dependencies map[string]goDependency) int {
	count := 0
	for _, dependency := range dependencies {
		if dependency.interfaceType {
			count++
		}
	}
	return count
}

func fieldCount(fields *ast.FieldList) int {
	if fields == nil {
		return 0
	}
	count := 0
	for _, field := range fields.List {
		if field == nil {
			continue
		}
		if len(field.Names) == 0 {
			count++
			continue
		}
		count += len(field.Names)
	}
	return count
}

func embeddedTypeNames(fields *ast.FieldList) []string {
	if fields == nil {
		return []string{}
	}
	result := make([]string, 0)
	for _, field := range fields.List {
		if field == nil || len(field.Names) != 0 {
			continue
		}
		if name := receiverExprName(field.Type); name != "" {
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result
}

func receiverTypeName(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 || fields.List[0] == nil {
		return ""
	}
	return receiverExprName(fields.List[0].Type)
}

func receiverExprName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return receiverExprName(value.X)
	case *ast.IndexExpr:
		return receiverExprName(value.X)
	case *ast.IndexListExpr:
		return receiverExprName(value.X)
	case *ast.ParenExpr:
		return receiverExprName(value.X)
	case *ast.SelectorExpr:
		if value.Sel != nil {
			return value.Sel.Name
		}
	}
	return ""
}

func typeHierarchyDepth(name string, embeds map[string][]string, visiting map[string]bool) int {
	if visiting[name] {
		return 0
	}
	parents := embeds[name]
	if len(parents) == 0 {
		return 0
	}
	visiting[name] = true
	defer delete(visiting, name)
	depth := 0
	for _, parent := range parents {
		candidate := 1
		if _, exists := embeds[parent]; exists {
			candidate += typeHierarchyDepth(parent, embeds, visiting)
		}
		if candidate > depth {
			depth = candidate
		}
	}
	return depth
}

func astSymbolKey(fileSet *token.FileSet, kind, name string, position token.Pos) string {
	return fmt.Sprintf("%s:%s:%d", kind, name, fileSet.PositionFor(position, false).Offset)
}

func goBuiltinType(name string) bool {
	switch name {
	case "any", "bool", "byte", "complex64", "complex128", "error", "float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "comparable":
		return true
	default:
		return false
	}
}
