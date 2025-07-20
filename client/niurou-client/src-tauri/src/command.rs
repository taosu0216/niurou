use serde::{Deserialize, Serialize};

const GO_BACKEND_URL: &str = "http://localhost:8080";

#[derive(Serialize, Deserialize, Debug)]
pub struct ChatRequest {
    pub message: String,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct ChatResponse {
    pub reply: String,
    pub timestamp: Option<String>,
}

#[tauri::command]
pub async fn chat(message: String) -> Result<ChatResponse, String> {
    let client = reqwest::Client::new();
    
    let chat_request = ChatRequest { message };
    
    let response = client
        .post(&format!("{}/api/v1/chat", GO_BACKEND_URL))
        .json(&chat_request)
        .send()
        .await
        .map_err(|e| format!("网络请求失败: {}", e))?;

    if !response.status().is_success() {
        return Err(format!("服务器错误: {}", response.status()));
    }

    let result = response
        .json::<ChatResponse>()
        .await
        .map_err(|e| format!("解析响应失败: {}", e))?;

    Ok(result)
}
