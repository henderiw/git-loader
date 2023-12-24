package v1alpha1

import "path"

func (r *SchemaSpec) GetBasePath(baseDir string) string {
	return path.Join(baseDir, r.Provider, r.Version)
}

func (r *SchemaSpec) GetNewSchemaBase(basePath string) SchemaSpecSchema {
	basePath = r.GetBasePath(basePath)

	return SchemaSpecSchema{
		Models:   getNewBase(basePath, r.Schema.Models),
		Includes: getNewBase(basePath, r.Schema.Includes),
		Excludes: r.Schema.Excludes,
	}
}

func getNewBase(basePath string, in []string) []string {
	str := make([]string, 0, len(in))
	for _, s := range in {

		str = append(str, path.Join(basePath, s))
		//str = append(str, fmt.Sprintf("./%s", path.Join(basePath, s)))
	}
	return str
}
