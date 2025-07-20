# 清空数据API使用说明

## 🗑️ 一键清空所有记忆数据

### API端点
```
DELETE http://localhost:8080/api/v1/clear-all
```

### 使用方法

#### 1. 使用curl命令
```bash
curl -X DELETE http://localhost:8080/api/v1/clear-all
```

#### 2. 使用curl并格式化输出
```bash
curl -X DELETE http://localhost:8080/api/v1/clear-all | jq .
```

#### 3. 使用测试脚本
```bash
./test_clear_api.sh
```

### 响应格式

#### 成功响应 (200 OK)
```json
{
  "success": true,
  "message": "所有记忆数据已成功清空",
  "timestamp": "2025-07-20T14:30:00Z"
}
```

#### 失败响应 (500 Internal Server Error)
```json
{
  "success": false,
  "error": "清空数据失败: 具体错误信息"
}
```

### 功能说明

- **完全清空**: 删除Neo4j中的所有节点和关系
- **重置向量库**: 删除并重新创建Qdrant集合
- **独立操作**: 不依赖LLM或Agent，直接操作数据库
- **安全警告**: ⚠️ 这是不可逆操作，会永久删除所有数据！

### 使用场景

1. **开发测试**: 快速清空测试数据
2. **数据重置**: 重新开始记忆收集
3. **故障恢复**: 清理损坏的数据
4. **隐私保护**: 彻底删除敏感信息

### 注意事项

- 确保在正确的环境中执行
- 重要数据请提前备份
- 操作不可撤销，请谨慎使用
