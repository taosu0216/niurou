// internal/graceful/shutdown.go
package graceful

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ShutdownManager 优雅退出管理器
type ShutdownManager struct {
	shutdownFuncs []ShutdownFunc
	mu            sync.Mutex
	shutdownCh    chan os.Signal
	done          chan struct{}
}

// ShutdownFunc 关闭函数类型
type ShutdownFunc func(ctx context.Context) error

// New 创建新的优雅退出管理器
func New() *ShutdownManager {
	return &ShutdownManager{
		shutdownFuncs: make([]ShutdownFunc, 0),
		shutdownCh:    make(chan os.Signal, 1),
		done:          make(chan struct{}),
	}
}

// RegisterShutdownFunc 注册关闭函数
func (sm *ShutdownManager) RegisterShutdownFunc(fn ShutdownFunc) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.shutdownFuncs = append(sm.shutdownFuncs, fn)
}

// Start 启动优雅退出监听
func (sm *ShutdownManager) Start() {
	// 监听系统信号
	signal.Notify(sm.shutdownCh, 
		syscall.SIGINT,  // Ctrl+C
		syscall.SIGTERM, // 终止信号
		syscall.SIGQUIT, // Ctrl+\
	)
	
	go sm.waitForShutdown()
}

// waitForShutdown 等待关闭信号
func (sm *ShutdownManager) waitForShutdown() {
	sig := <-sm.shutdownCh
	log.Printf("🛑 收到退出信号: %v", sig)
	
	// 开始优雅关闭流程
	sm.performShutdown()
	
	// 通知主程序可以退出
	close(sm.done)
}

// performShutdown 执行关闭流程
func (sm *ShutdownManager) performShutdown() {
	log.Println("🔄 开始优雅关闭流程...")
	
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	sm.mu.Lock()
	funcs := make([]ShutdownFunc, len(sm.shutdownFuncs))
	copy(funcs, sm.shutdownFuncs)
	sm.mu.Unlock()
	
	// 并发执行所有关闭函数
	var wg sync.WaitGroup
	for i, fn := range funcs {
		wg.Add(1)
		go func(index int, shutdownFunc ShutdownFunc) {
			defer wg.Done()
			
			log.Printf("🔄 执行关闭函数 %d...", index+1)
			if err := shutdownFunc(ctx); err != nil {
				log.Printf("❗️ 关闭函数 %d 执行失败: %v", index+1, err)
			} else {
				log.Printf("✅ 关闭函数 %d 执行成功", index+1)
			}
		}(i, fn)
	}
	
	// 等待所有关闭函数完成或超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Println("✅ 所有关闭函数执行完成")
	case <-ctx.Done():
		log.Println("⚠️ 关闭流程超时，强制退出")
	}
	
	log.Println("🏁 优雅关闭流程完成")
}

// Wait 等待关闭完成
func (sm *ShutdownManager) Wait() {
	<-sm.done
}

// Stop 手动触发关闭
func (sm *ShutdownManager) Stop() {
	select {
	case sm.shutdownCh <- syscall.SIGTERM:
	default:
		// 如果通道已满，说明已经在关闭中
	}
}

// CreateShutdownContext 创建可取消的上下文
func CreateShutdownContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// WaitForSignal 等待系统信号（简化版本）
func WaitForSignal() os.Signal {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	return <-sigCh
}

// WithTimeout 为关闭操作添加超时
func WithTimeout(fn ShutdownFunc, timeout time.Duration) ShutdownFunc {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		
		done := make(chan error, 1)
		go func() {
			done <- fn(ctx)
		}()
		
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// LogShutdownFunc 包装关闭函数并添加日志
func LogShutdownFunc(name string, fn ShutdownFunc) ShutdownFunc {
	return func(ctx context.Context) error {
		log.Printf("🔄 开始关闭 %s...", name)
		start := time.Now()
		
		err := fn(ctx)
		
		duration := time.Since(start)
		if err != nil {
			log.Printf("❗️ 关闭 %s 失败 (耗时: %v): %v", name, duration, err)
		} else {
			log.Printf("✅ 关闭 %s 成功 (耗时: %v)", name, duration)
		}
		
		return err
	}
}
