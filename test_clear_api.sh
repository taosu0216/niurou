#!/bin/bash

# test_clear_api.sh - 测试清空数据的HTTP API

echo "🧪 测试清空数据的HTTP API..."

# 检查服务是否运行
echo "📝 检查服务状态..."
curl -s http://localhost:8080/health > /dev/null
if [ $? -ne 0 ]; then
    echo "❗️ 服务未运行，请先启动服务: ./niurou"
    exit 1
fi

echo "✅ 服务正在运行"

# 调用清空数据API
echo "🗑️ 调用清空数据API..."
response=$(curl -s -X DELETE http://localhost:8080/api/v1/clear-all)

echo "📊 API响应:"
echo "$response" | jq . 2>/dev/null || echo "$response"

# 检查响应是否成功
if echo "$response" | grep -q '"success":true'; then
    echo "✅ 清空数据成功！"
else
    echo "❗️ 清空数据失败！"
    exit 1
fi

echo "🎉 HTTP API测试完成！"
