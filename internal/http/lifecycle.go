package http

import (
	"context"
	"github.com/gofiber/fiber/v3"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"sproxy/internal/config"
)

type LifecycleDependency struct {
	fx.In
	Lc     fx.Lifecycle
	App    *fiber.App
	Config *config.Config
	Logger *zap.SugaredLogger
}

func httpLifecycle(dep LifecycleDependency) {
	lc, app, cfg, log := dep.Lc, dep.App, dep.Config, dep.Logger
	lc.Append(fx.StartStopHook(
		func() {
			go func() {
				localAddress := "http://127.0.0.1:" + cfg.Http.GetPort()
				log.Debugf("Http Listening on %s", localAddress)
				lo.Must0(app.Listen(
					":"+cfg.Http.GetPort(),
					fiber.ListenConfig{
						DisableStartupMessage: true,
						EnablePrintRoutes:     false,
						EnablePrefork:         cfg.Http.Prefork,
						ShutdownTimeout:       1000,
					},
				), "sproxy start fail")
			}()
		},
		func(ctx context.Context) error {
			return app.ShutdownWithContext(ctx)
		},
	),
	)
}

//func httpLifecycle(dep LifecycleDependency) {
//	lc, app, cfg, log := dep.Lc, dep.App, dep.Config, dep.Logger
//
//	var ln net.Listener
//	var wg sync.WaitGroup
//
//	lc.Append(fx.Hook{
//		OnStart: func(ctx context.Context) error {
//			var err error
//			addr := ":" + cfg.Http.GetPort()
//
//			// 创建共享 listener
//			ln, err = net.Listen("tcp", addr)
//			if err != nil {
//				return fmt.Errorf("failed to listen: %w", err)
//			}
//
//			// 设置最大可用核数
//			runtime.GOMAXPROCS(runtime.NumCPU())
//
//			log.Debugf("Http Listening on http://127.0.0.1:%s with %d threads", cfg.Http.GetPort(), runtime.NumCPU())
//
//			// 启动多个 listener goroutine（NumCPU 个）
//			for i := 0; i < runtime.NumCPU(); i++ {
//				wg.Add(1)
//				go func(id int) {
//					defer wg.Done()
//					err := app.Listener(ln)
//					if err != nil && !errors.Is(err, net.ErrClosed) {
//						log.Errorf("listener #%d error: %v", id, err)
//					}
//				}(i)
//			}
//
//			return nil
//		},
//
//		OnStop: func(ctx context.Context) error {
//			// 关闭 fiber 和 listener
//			if err := app.ShutdownWithContext(ctx); err != nil {
//				log.Errorf("fiber shutdown error: %v", err)
//			}
//
//			// 等待所有 listener 协程退出
//			wg.Wait()
//			log.Debugf("Http server stopped")
//			return nil
//		},
//	})
//}
