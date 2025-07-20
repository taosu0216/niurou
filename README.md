# 🧠 Niurou - 智能记忆AI助手

一个具备长期记忆能力的智能AI助手，支持知识图谱构建、向量检索、实时记忆更新和智能记忆回收。

## ✨ 核心特性

### 🎯 智能记忆管理
- **长期记忆存储** - 基于Neo4j图数据库和Qdrant向量数据库
- **智能记忆回收** - 自动判断对话价值，选择性保存有意义的内容
- **实时记忆更新** - 对话中即时修正、补充记忆信息
- **去重机制** - 避免重复记忆累积，保持数据库整洁

### 🔍 混合检索系统
- **图谱检索** - 基于实体关系的精确查询
- **向量检索** - 基于语义相似度的模糊匹配
- **混合搜索** - 结合两种检索方式，提供最佳搜索结果

### 🤖 智能对话能力
- **工具调用** - 自动使用记忆搜索和更新工具
- **上下文理解** - 基于历史记忆提供个性化回答
- **知识提取** - 自动从对话中提取结构化知识

## 🚀 快速开始

### 环境要求
- Go 1.21+
- Docker & Docker Compose
- Neo4j 数据库
- Qdrant 向量数据库

### 安装步骤

1. **克隆项目**
```bash
git clone <repository-url>
cd niurou
```

2. **启动数据库服务**
```bash
# 启动Neo4j
docker run -d -p 7474:7474 -p 7687:7687 \
    -v $(pwd)/neo4j_data:/data \
    -e NEO4J_AUTH=neo4j/password \
    --name neo4j-agent-db neo4j:latest

# 启动Qdrant
docker run -d -p 6333:6333 -p 6334:6334 \
    -v $(pwd)/qdrant_storage:/qdrant/storage \
    --name qdrant-agent qdrant/qdrant
```

3. **编译运行**
```bash
# 编译
go build -o niurou main.go

# 运行
./niurou
```

## 📁 项目结构

```
niurou/
├── main.go                 # 应用入口
├── internal/               # 内部包
│   ├── agent/             # AI Agent核心逻辑
│   ├── configger/         # 配置管理
│   ├── graphDB/           # Neo4j图数据库
│   ├── llm/               # LLM集成和提示词
│   ├── memManager/        # 记忆管理器
│   ├── server/            # HTTP服务器
│   ├── service/           # 业务服务层
│   ├── tools/             # AI工具集合
│   └── vecX/              # 向量数据库
├── test_*.go              # 测试文件
├── test_*.sh              # 测试脚本
└── docs/                  # 文档目录
```

## 🔧 API文档

### 聊天接口
- `POST /api/v1/chat` - 发送消息
- `GET /api/v1/status` - 获取服务状态

### 管理接口
- `DELETE /api/v1/clear-all` - 清空所有数据
- `GET /health` - 健康检查

详细API文档请参考 [API_USAGE.md](API_USAGE.md)

## 🧪 测试

```bash
# 测试记忆回收功能
go run test_memory_recovery.go

# 测试记忆更新功能
go run test_update_memory.go

# 测试去重功能
go run test_deduplication.go

# 测试清空功能
./test_clear_api.sh
```

## 🛠️ 开发指南

每个模块都有详细的README文档：
- [Agent模块](internal/agent/README.md)
- [记忆管理器](internal/memManager/README.md)
- [工具集合](internal/tools/README.md)
- [LLM集成](internal/llm/README.md)

## 📝 许可证

MIT License