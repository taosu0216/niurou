package main

import (
	"context"
	"fmt"
	"log"

	"niurou/internal/memManager" // <-- 只导入顶层管理器！
)

func main() {
	// 1. 初始化 Memory Manager (一行代码启动所有底层服务)
	memory, err := memManager.New()
	if err != nil {
		log.Fatalf("初始化记忆管理器失败: %v", err)
	}
	defer memory.Close()

	ctx := context.Background()

	// --- 演示流程 ---
	fmt.Println("\n--- 步骤 1: 添加初始记忆 ---")
	memoryIdToUpdate, err := memory.AddMemory(ctx, "我最喜欢的运动是篮球。")
	if err != nil {
		log.Fatalf("添加记忆失败: %v", err)
	}
	_, _ = memory.AddMemory(ctx, "我的创始人是一名 Go 语言开发者。")
	fmt.Println("--------------------------------------------------")

	// --- 步骤 2: 在更新前进行搜索 ---
	fmt.Println("\n--- 步骤 2: 首次搜索（更新前） ---")
	queryBeforeUpdate := "我喜欢什么球类运动？"
	searchResults, err := memory.HybridSearch(ctx, queryBeforeUpdate, 2)
	if err != nil {
		log.Fatalf("混合搜索失败: %v", err)
	}
	log.Printf("为查询 \"%s\" 找到 %d 个相关结果:", queryBeforeUpdate, len(searchResults))
	for _, text := range searchResults {
		fmt.Printf("  - %s\n", text)
	}
	fmt.Println("--------------------------------------------------")

	// --- 步骤 3: 更新一条记忆 ---
	fmt.Println("\n--- 步骤 3: 更新记忆 ---")
	newMemoryText := "我最喜欢的运动是足球，因为它的策略性更强。"
	err = memory.UpdateMemory(ctx, memoryIdToUpdate, newMemoryText)
	if err != nil {
		log.Fatalf("更新记忆失败: %v", err)
	}
	fmt.Println("--------------------------------------------------")

	// --- 步骤 4: 在更新后再次搜索 ---
	fmt.Println("\n--- 步骤 4: 再次搜索（更新后） ---")
	queryAfterUpdate := "我为什么喜欢足球？"
	searchResults, err = memory.HybridSearch(ctx, queryAfterUpdate, 2)
	if err != nil {
		log.Fatalf("混合搜索失败: %v", err)
	}
	log.Printf("为查询 \"%s\" 找到 %d 个相关结果:", queryAfterUpdate, len(searchResults))
	for _, text := range searchResults {
		fmt.Printf("  - %s\n", text)
	}
	fmt.Println("--------------------------------------------------")

	// --- 步骤 5: 删除一条记忆 ---
	fmt.Println("\n--- 步骤 5: 删除记忆 ---")
	err = memory.DeleteMemory(ctx, memoryIdToUpdate)
	if err != nil {
		log.Fatalf("删除记忆失败: %v", err)
	}
	fmt.Println("--------------------------------------------------")

	// --- 步骤 6: 在删除后最终搜索 ---
	fmt.Println("\n--- 步骤 6: 最终搜索（删除后） ---")
	queryAfterDelete := "我喜欢什么运动？"
	searchResults, err = memory.HybridSearch(ctx, queryAfterDelete, 2)
	if err != nil {
		log.Fatalf("混合搜索失败: %v", err)
	}
	log.Printf("为查询 \"%s\" 找到 %d 个相关结果:", queryAfterDelete, len(searchResults))
	if len(searchResults) == 0 {
		log.Println("  [成功] 未找到关于“运动”的记忆，因为它已被删除。")
	} else {
		log.Println("  [警告] 仍然找到了已删除的记忆，请检查删除逻辑。")
		for _, text := range searchResults {
			fmt.Printf("  - %s\n", text)
		}
	}
	fmt.Println("--------------------------------------------------")

	log.Println("\n完整记忆生命周期测试完成！")
}
