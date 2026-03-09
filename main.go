package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	appcontainer "icctv-http-service/container"
	"icctv-http-service/middlewares"
	"icctv-http-service/routes"

	"github.com/joho/godotenv"
)

func main() {
	// 尝试加载 .env 文件，如果文件不存在也不会报错（适用于生产环境直接注入环境变量的情况）
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not loaded: %v", err)
	} else {
		log.Println("✓ .env file loaded successfully")
	}

	// 调试：打印关键环境变量
	log.Printf("DB_DRIVER=%s", os.Getenv("DB_DRIVER"))
	log.Printf("DB_HOST=%s", os.Getenv("DB_HOST"))
	log.Printf("DB_PORT=%s", os.Getenv("DB_PORT"))

	container, err := appcontainer.Build()
	if err != nil {
		log.Fatalf("failed to build container: %v", err)
	}

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	// Routes
	routes.SetupRoutes(mux, container.Controllers, container.Middlewares)

	// 应用 CORS 中间件
	corsMiddleware := middlewares.NewCORSMiddleware()
	handler := corsMiddleware.Handle(mux)

	// 启动后台健康检查服务
	container.HealthChecker.Start()

	// 优雅关闭处理
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down...")
		container.HealthChecker.Stop()
		os.Exit(0)
	}()

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("OrangePi HTTP service listening on %s", addr)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}