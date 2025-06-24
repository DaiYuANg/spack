package scanner

import (
	"context"
	"golang.org/x/sync/errgroup"
	"io/fs"
	"path/filepath"
	"sproxy/pkg"
)

type Scanner struct {
	root       string
	processors []FileProcessor
	maxWorkers int
}

func (s *Scanner) Run(ctx context.Context) error {
	sem := make(chan struct{}, s.maxWorkers)
	eg, ctx := errgroup.WithContext(ctx)

	//// 预构造所有感兴趣的扩展集合，方便过滤
	//extSet := make(map[string]struct{})
	//for _, p := range s.processors {
	//	for _, e := range p.Extensions() {
	//		extSet[e] = struct{}{}
	//	}
	//}

	walkFunc := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		mime, err := pkg.DetectMime(path)
		if err != nil {
			return err
		}
		println(path)
		println(mime.String())
		println(mime.Extension())
		//ext := strings.ToLower(filepath.Ext(path))
		//if _, ok := extSet[ext]; !ok {
		//	return nil
		//}

		//rel, err := filepath.Rel(s.root, path)
		//if err != nil {
		//	return err
		//}

		sem <- struct{}{}
		eg.Go(func() error {
			defer func() { <-sem }()

			//for _, proc := range s.processors {
			//	if contains(proc.Extensions(), ext) {
			//		if err := proc.Process(ctx, path, rel); err != nil {
			//			return fmt.Errorf("processor %T failed: %s", proc, err)
			//		}
			//	}
			//}
			return nil
		})

		return nil
	}

	if err := filepath.WalkDir(s.root, walkFunc); err != nil {
		return err
	}
	return eg.Wait()
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
