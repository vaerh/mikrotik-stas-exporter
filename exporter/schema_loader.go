package exporter

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
)

func LoadResSchemas(ctx context.Context, basedir string) ([]ResSchema, error) {
	var files []string
	err := filepath.Walk(basedir, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			if strings.HasSuffix(f.Name(), ".yaml") {
				absolutefilepath, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				files = append(files, absolutefilepath)
			}
		}
		return err
	})

	if err != nil {
		return nil, err
	}

	var res []ResSchema

	for _, file := range files {
		s, err := SchemaParser(file)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("")
			continue
		}

		res = append(res, *s)
	}

	return res, nil
}
