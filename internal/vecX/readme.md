# VecX SDK: 高性能本地向量化与记忆库 - 使用与维护文档

## 1. 概述

`VecX` 是一个高性能、自包含的本地向量化与记忆存储 SDK。它专为需要长期、语义化记忆的 AI Agent 项目而设计。

本 SDK 采用 Go 语言封装，提供简洁的 API 接口，内部集成了以下核心技术栈，将所有复杂性都对用户屏蔽：

*   **向量生成**: 使用 `ONNX Runtime` 调用 `paraphrase-multilingual-mpnet-base-v2` 模型，将文本转换为高质量的 768 维向量。
*   **高性能分词**: 通过一个由 Go 在后台管理的**长连接 Python 服务**，利用 `transformers` 库进行精确、高效的文本分词，避免了重复启动 Python 进程的巨大开销。
*   **持久化记忆**: 使用 **Qdrant** 向量数据库作为记忆存储后端，通过 Docker 在本地运行，确保了记忆的持久化、高性能检索和未来的可扩展性。

最终，使用者只需通过几行 Go 代码，即可实现毫秒级的记忆存储和语义搜索。

## 2. 功能特性

*   **简洁的 API**: 只需 `New()`, `AddMemory()`, `SearchSimilarMemories()`, `Close()` 四个核心函数即可完成所有操作。
*   **高性能**: 一次性初始化成本后，后续所有向量化和搜索请求均在毫秒级别完成。
*   **高质量语义理解**: 内置 `mpnet` 多语言模型，能精准捕捉中文及其他多种语言的语义，搜索结果优于轻量级模型。
*   **持久化记忆**: 所有记忆都存储在 Qdrant 数据库中，并通过 Docker 数据卷持久化到本地文件系统，重启程序或容器后记忆不会丢失。
*   **完全封装**: 所有依赖（模型、脚本、动态库、Python 环境）都封装在 `iinternal/vecX` 包内，实现了高度的模块化和便携性。

## 3. 环境与依赖设置

在调用 SDK 之前，请确保您的开发环境满足以下要求。**这些步骤只需在项目初次设置时执行一次。**

### 3.1. 前置要求

*   **Go**: 版本 1.18 或更高。
*   **Docker**: 已安装并正在运行。
*   **uv** (可选但推荐): 一个极速的 Python 包管理器，用于创建虚拟环境。

### 3.2. 启动 Qdrant 服务

`VecX` SDK 需要一个正在运行的 Qdrant 实例来存储记忆。请在项目根目录 (`niurou/`) 打开终端，运行以下命令启动 Qdrant：

```bash
docker run -d -p 6333:6333 -p 6334:6334 \
    -v $(pwd)/qdrant_storage:/qdrant/storage \
    qdrant/qdrant
```
*   `-d` 参数会让容器在后台运行。
*   `-v` 参数会将数据库数据持久化到项目根目录下的 `qdrant_storage` 文件夹。

### 3.3. 设置 Python 虚拟环境

SDK 内部的 Python 分词服务需要一个独立的环境和依赖。

1.  **进入 SDK 目录**:
    ```bash
    cd iinternal/vecX/
    ```

2.  **创建虚拟环境**:
    ```bash
    uv venv
    ```

3.  **激活环境**:
    ```bash
    source .venv/bin/activate
    ```

4.  **安装所有依赖**:
    ```bash
    uv pip install transformers torch numpy sentence-transformers accelerate
    ```

5.  **回到项目根目录**:
    ```bash
    cd ../..
    ```

## 4. SDK 使用指南 (API)

### 4.1. 快速上手

下面是一个完整的使用示例，展示了如何初始化 SDK、添加记忆并进行搜索。

```go
// main.go

package main

import (
	"context"
	"fmt"
	"log"

	"niurou/iinternal/vecX" // 导入您自己的 SDK 包
)

func main() {
	// 1. 初始化 VecX 服务 (一行代码完成所有复杂操作)
	// New() 会自动加载模型、启动 Python 服务、连接数据库。
	vecService, err := vecX.New()
	if err != nil {
		log.Fatalf("初始化 VecX 服务失败: %v", err)
	}
	// 使用 defer 确保在程序结束时，所有后台服务都能被优雅地关闭
	defer vecService.Close()

	ctx := context.Background()

	// --- 演示流程 ---
	fmt.Println("\n--- 步骤 1: 使用 SDK 添加记忆 ---")
	_, _ = vecService.AddMemory(ctx, "我最喜欢的颜色是蓝色。")
	_, _ = vecService.AddMemory(ctx, "我的创始人是一名 Go 语言开发者。")
	
	fmt.Println("\n--- 步骤 2: 使用 SDK 进行语义搜索 ---")
	query := "关于我的创造者，你了解什么？"
	searchResults, err := vecService.SearchSimilarMemories(ctx, query, 2)
	if err != nil {
		log.Fatalf("搜索失败: %v", err)
	}

	fmt.Println("\n--- 步骤 3: 展示搜索结果 ---")
	log.Printf("为查询 \"%s\" 找到 %d 个相似结果:", query, len(searchResults))
	for i, point := range searchResults {
		fmt.Printf("  %d. Score: %.4f (越大越相似)\n", i+1, point.GetScore())
		fmt.Printf("     Text: %s\n", point.GetPayload()["text"].GetStringValue())
	}
}
```

### 4.2. API 详解

#### `vecX.New() (Service, error)`
初始化 `VecX` SDK 的所有服务。这是您需要调用的第一个函数。它会加载 ONNX 模型、启动后台 Python 分词服务并等待其就绪、连接到 Qdrant 数据库并确保集合存在。成功时返回一个 `Service` 接口实例，失败时返回错误。

#### `service.AddMemory(ctx context.Context, memoryText string) (string, error)`
将一段文本添加到 Agent 的记忆库中。它会自动为文本生成向量，并将其与文本内容、时间戳等元数据一起存储。成功时返回为该条记忆生成的唯一 ID (UUID)，失败时返回错误。

#### `service.SearchSimilarMemories(ctx context.Context, queryText string, topK uint64) ([]*qdrant.ScoredPoint, error)`
根据一段查询文本，从记忆库中搜索语义最相似的 `topK` 条记忆。返回的结果是一个 `ScoredPoint` 切片，每个 `ScoredPoint` 都包含了相似度分数、唯一 ID 和您存储的元数据（Payload）。

#### `service.Close()`
优雅地关闭并清理所有后台服务，包括 Python 子进程、ONNX Runtime 环境和 Qdrant 的 gRPC 连接。**强烈建议使用 `defer service.Close()`** 来确保在您的程序退出时总是能调用此函数，避免产生僵尸进程。

## 5. 维护与扩展

### 5.1. 清空/重置记忆库

如果您想彻底清空所有记忆，最简单的方法是重置 Qdrant 的持久化数据：
1.  停止 Qdrant 容器: `docker stop <container_id>`
2.  删除数据文件夹: `rm -rf qdrant_storage`
3.  重新启动 Qdrant 容器。

### 5.2. 更换 AI 模型

SDK 的设计使得更换模型非常简单：
1.  **导出新模型**: 使用 `optimum-cli` 导出一个新的 ONNX 模型文件夹。
2.  **替换文件**: 将 `iinternal/vecX/` 目录下的旧模型文件夹（如 `mpnet_onnx`）替换为新模型文件夹。
3.  **修改 SDK 代码 (`sdk.go`)**:
    *   在 `const` 部分，更新 `vectorSize` 以匹配新模型的输出维度。
    *   在 `New()` 函数中，更新 `modelPath` 指向新的 `.onnx` 文件。
    *   检查新模型的输入要求。如果它需要 `token_type_ids`，您需要同时修改 `sdk.go` 和 `tokenizer.py` 以提供这个字段。如果它不需要，请确保代码中也没有提供。
4.  **修改 Tokenizer 脚本 (`tokenizer.py`)**:
    *   更新 `AutoTokenizer.from_pretrained(...)` 的路径以指向新模型文件夹。