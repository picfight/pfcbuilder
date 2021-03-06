package builder

import (
	"fmt"
	"github.com/jfixby/pin"
	"github.com/jfixby/pin/fileops"
	"github.com/jfixby/pin/lang"
	"github.com/picfight/pfcbuilder/deps"
	"github.com/picfight/pfcbuilder/ut"
	"github.com/stevenle/topsort"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

//func SortPackages(gomodresolver *deps.GoModResolver, rootd string, gomodfilepath *deps.GoModPath) []deps.Dependency {
//	result := &[]deps.Dependency{}
//	deps := map[string]*[]deps.Dependency{}
//	CollectImport(gomodresolver, deps, rootd, gomodfilepath)
//
//	for k, v := range deps {
//		pin.D(k, v)
//	}
//
//	return *result
//}

func LoadAllGoMods(root *deps.GitTag) {
	cache := &deps.UrlCache{
		LocalFolder: "GitCache",
	}
	cache.Load()
	graph := topsort.NewGraph()

	ds := &DepCollector{
		//depsList: []deps.Dependency{},
		//depsSet:  map[deps.Dependency]int{},
		graph:    graph,
		depsTree: map[string]map[string]bool{},
	}

	LoadGoMods(root, root, cache, ds)

	r, e := graph.TopSort(root.ToString()) // => [C, B, A]
	lang.CheckErr(e)
	pin.D("toposort", r)

	handler := &Handler{
		DepCollector: ds,
	}
	for _, handler.i = range r {
		processDep(handler)
	}

	//gomod := deps.ReadGoMod(root, cache)
	//pin.D("gomod", gomod)
	//
	//for _, d := range gomod.Dependencies {
	//	t := ResolveTarget(root, d)
	//	LoadGoMods(root, t, cache)
	//}

}

func processDep(handler *Handler) {

	tag := handler.DepCollector.GetTag(handler.i)
	kids := handler.DepCollector.ListDepsFor(handler.i)
	pin.D(fmt.Sprintf("%v", tag), kids)
	//dep := ParseDep(handler.i)

}

type DepCollector struct {
	//depsList []deps.Dependency
	tags     map[string]*deps.GitTag
	graph    *topsort.Graph
	depsTree map[string]map[string]bool
}

func (C *DepCollector) Append(target, next *deps.GitTag) {
	owner := target.ToString()
	child := next.ToString()

	if C.tags == nil {
		C.tags = map[string]*deps.GitTag{}
	}

	C.tags[owner] = target
	C.tags[child] = next

	C.graph.AddEdge(owner, child)

	if C.depsTree[owner] == nil {
		C.depsTree[owner] = map[string]bool{}
	}
	C.depsTree[owner][child] = true

}

func (c *DepCollector) ListDepsFor(owner string) []string {
	r := []string{}
	for k, _ := range c.depsTree[owner] {
		//r = append(r, fmt.Sprintf("%v -> %v", k, v))
		r = append(r, fmt.Sprintf("%v", k))
	}
	return r
}

func (C *DepCollector) GetTag(tag string) *deps.GitTag {
	return C.tags[tag]
}

//func (c *DepCollector) ListDepsFor(owner string) map[deps.Dependency]bool {
//	return c.depsTree[owner]
//}

func LoadGoMods(root *deps.GitTag, target *deps.GitTag, cache *deps.UrlCache, ds *DepCollector) {

	CollectImport(ds, root, target, cache)

	//for _, v := range ds.depsList {
	//	pin.D(fmt.Sprintf("%v", v), ds.depsSet[v])
	//	//pin.D("", *v)
	//}

}

func CollectImport(ds *DepCollector, rootTag, targetTag *deps.GitTag, cache *deps.UrlCache) *deps.GoModHandler {
	gomod := deps.ReadGoMod(targetTag, cache)

	for _, dep := range gomod.Dependencies {
		if strings.HasPrefix(dep.Import, rootTag.Package()) {
			//pin.D("", dep)
			nextTag := ResolveTarget(rootTag, dep)
			//nextGomod :=
			CollectImport(ds, rootTag, nextTag, cache)
			ds.Append(targetTag, nextTag)

			//ds.Append(gomod.Name, targetTag.DepVersion, nextGomod.Name, nextTag.DepVersion)
		}
		//pin.D("", dep)
	}

	return gomod

}

func ResolveTarget(root *deps.GitTag, dep deps.Dependency) *deps.GitTag {

	subtag := strings.ReplaceAll(dep.Import, root.Package(), "")[1:]
	//packagetag := path.Base(subtag)

	target := &deps.GitTag{
		GitOrg:     root.GitOrg,
		GitRepo:    root.GitRepo,
		SubPackage: subtag,
		ReleaseTag: subtag + "/" + dep.Version,
		DepVersion: dep.Version,
		//DepTag:     dep.Name + " " + dep.Version,
	}

	//dcrjson%2Fv3.1.0

	return target
}

func Swap(sorted []string, x int, y int) {
	sorted[x], sorted[y] = sorted[y], sorted[x]
}

func Relatives(root string, subfiles map[string]bool) map[string]string {
	result := map[string]string{}
	for e, _ := range subfiles {
		key := e[len(root)+1 : len(e)]
		result[key] = e
	}
	return result
}

func IsBigger(x string, y string, graph deps.DepsGraph) bool {
	if len(graph.ListChildrenForVertex(x)) == len(graph.ListChildrenForVertex(y)) {
		return x > y
	}
	return len(graph.ListChildrenForVertex(x)) > len(graph.ListChildrenForVertex(y))
}

func Resort(sorted []string, graph deps.DepsGraph) []string {

	N := len(sorted)
	swap := true
	for {
		for i := 0; i < N-1; i++ {
			if IsBigger(sorted[i], sorted[i+1], graph) {
				Swap(sorted, i, i+1)
				swap = true
			}
		}
		if !swap {
			break
		}
		swap = false
	}

	return sorted
}

const ALL_CHILDREN = true
const DIRECT_CHILDREN = !ALL_CHILDREN

func ListFiles(
	target string,
	IgnoredFiles map[string]bool,
	children bool,
	filter ut.FileFilter) map[string]bool {
	if fileops.IsFile(target) {
		lang.ReportErr("This is not a folder: %v", target)
	}

	files, err := ioutil.ReadDir(target)
	lang.CheckErr(err)
	result := map[string]bool{}
	for _, f := range files {
		fileName := f.Name()
		filePath := filepath.Join(target, fileName)
		filePath = strings.ReplaceAll(filePath, "\\", "/")
		if IgnoredFiles[fileName] {
			continue
		}
		if fileops.IsFolder(filePath) && children != DIRECT_CHILDREN {
			children := ListFiles(filePath, IgnoredFiles, children, filter)
			//result = append(result, children...)
			result = putAll(result, children)
			continue
		}

		if fileops.IsFile(filePath) {
			if filter(filePath) {
				//result = append(result, filePath)
				result[filePath] = true
			}
			continue
		}
	}
	if filter(target) {
		//result = append(result, target)
		result[target] = true
	}
	lang.CheckErr(err)
	return result
}

func putAll(result map[string]bool, children map[string]bool) map[string]bool {
	for k, v := range children {
		result[k] = v
	}
	return result
}

func GoPath(git string) string {
	return strings.ReplaceAll(filepath.Join(os.Getenv("GOPATH"), "src", git), "\\", "/")
}

func ClearProject(target string, ignore map[string]bool) {
	pin.D("clear", target)
	files, err := ioutil.ReadDir(target)
	lang.CheckErr(err)

	for _, f := range files {
		fileName := f.Name()
		filePath := filepath.Join(target, fileName)
		if ignore[fileName] {
			pin.D("  skip", filePath)
			continue
		}
		pin.D("delete", filePath)
		err := os.RemoveAll(filePath)
		lang.CheckErr(err)
	}
	pin.D("")

}

func ShortenFileNames(input map[string]bool) (short2long map[string]string) {
	short2long = map[string]string{}
	for k, _ := range input {
		s := filepath.Base(k)
		short2long[s] = k
	}
	return
}
