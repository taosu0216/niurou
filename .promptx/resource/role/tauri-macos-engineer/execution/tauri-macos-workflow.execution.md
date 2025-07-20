<execution>
  <constraint>
    ## Tauri macOS 开发技术约束
    - **Rust 版本要求**：最低支持 Rust 1.70+，推荐使用最新稳定版
    - **macOS 版本支持**：最低支持 macOS 10.15 Catalina
    - **Xcode 依赖**：需要 Xcode Command Line Tools 用于编译
    - **代码签名要求**：发布应用必须有有效的 Apple Developer 证书
    - **安全限制**：严格的 CSP 配置，最小化 API 权限暴露
  </constraint>

  <rule>
    ## 强制性开发规则
    - **安全第一**：所有 Tauri 命令必须经过权限验证
    - **错误处理**：所有 Rust 命令必须返回 Result 类型
    - **资源管理**：及时释放文件句柄、网络连接等系统资源
    - **版本控制**：tauri.conf.json 配置变更必须版本控制
    - **测试覆盖**：核心业务逻辑必须有单元测试覆盖
  </rule>

  <guideline>
    ## 开发指导原则
    - **渐进式开发**：从简单功能开始，逐步增加复杂性
    - **平台优先**：优先实现 macOS 特有功能，再考虑跨平台兼容
    - **性能监控**：持续监控应用性能指标
    - **用户反馈**：及时收集和响应用户体验反馈
    - **文档同步**：代码变更同步更新技术文档
  </guideline>

  <process>
    ## Tauri macOS 开发完整流程
    
    ### Phase 1: 环境准备 (30分钟)
    ```bash
    # 安装 Rust 和 Tauri CLI
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
    cargo install tauri-cli
    
    # 验证 macOS 开发环境
    xcode-select --install
    ```
    
    ### Phase 2: 项目初始化 (15分钟)
    ```bash
    # 创建 Tauri 项目
    cargo tauri init
    
    # 配置 tauri.conf.json
    # - 设置应用元数据
    # - 配置窗口属性
    # - 设置安全策略
    ```
    
    ### Phase 3: 架构设计 (60分钟)
    
    #### 前端架构设计
    ```mermaid
    graph TD
        A[前端 UI] --> B[Tauri API 调用]
        B --> C[Rust 后端命令]
        C --> D[系统 API 调用]
        D --> E[macOS 原生功能]
    ```
    
    #### Rust 后端设计
    ```rust
    #[tauri::command]
    async fn handle_file_operation(path: String) -> Result<String, String> {
        // 实现文件操作逻辑
        // 错误处理和权限检查
    }
    ```
    
    ### Phase 4: 核心功能开发 (2-4周)
    
    #### 开发优先级
    1. **基础功能**：窗口管理、基础 UI
    2. **业务逻辑**：核心功能实现
    3. **系统集成**：文件系统、网络、通知
    4. **macOS 特性**：菜单栏、Dock、快捷键
    
    #### 代码组织结构
    ```
    src-tauri/
    ├── src/
    │   ├── main.rs          # 应用入口
    │   ├── commands/        # Tauri 命令模块
    │   ├── utils/           # 工具函数
    │   └── models/          # 数据模型
    ├── tauri.conf.json      # Tauri 配置
    └── Cargo.toml           # Rust 依赖
    ```
    
    ### Phase 5: macOS 原生集成 (1-2周)
    
    #### 系统托盘集成
    ```rust
    use tauri::{SystemTray, SystemTrayMenu, SystemTrayMenuItem};
    
    let tray_menu = SystemTrayMenu::new()
        .add_item(SystemTrayMenuItem::new("显示", "show"))
        .add_item(SystemTrayMenuItem::new("退出", "quit"));
    
    let system_tray = SystemTray::new().with_menu(tray_menu);
    ```
    
    #### 菜单栏应用模式
    ```json
    {
      "tauri": {
        "windows": [{
          "decorations": false,
          "alwaysOnTop": true,
          "skipTaskbar": true
        }]
      }
    }
    ```
    
    ### Phase 6: 性能优化 (1周)
    
    #### 编译优化配置
    ```toml
    [profile.release]
    lto = true
    codegen-units = 1
    panic = "abort"
    strip = true
    ```
    
    #### 资源优化策略
    - 静态资源压缩
    - 懒加载非关键模块
    - 内存使用监控
    
    ### Phase 7: 打包发布 (2-3天)
    
    #### 代码签名配置
    ```json
    {
      "tauri": {
        "bundle": {
          "macOS": {
            "signingIdentity": "Developer ID Application: Your Name",
            "providerShortName": "YourTeamID"
          }
        }
      }
    }
    ```
    
    #### 发布流程
    ```bash
    # 构建发布版本
    cargo tauri build
    
    # 公证应用（需要 Apple ID）
    xcrun notarytool submit target/release/bundle/macos/YourApp.app.zip \
      --apple-id your-apple-id@example.com \
      --password your-app-password \
      --team-id YourTeamID
    ```
  </process>

  <criteria>
    ## 质量评价标准
    
    ### 功能完整性
    - ✅ 核心功能正常运行
    - ✅ 错误处理完善
    - ✅ macOS 原生特性集成
    - ✅ 用户体验流畅
    
    ### 性能指标
    - ✅ 启动时间 < 2秒
    - ✅ 内存占用 < 100MB
    - ✅ CPU 使用率 < 5%（空闲时）
    - ✅ 应用包大小 < 50MB
    
    ### 安全标准
    - ✅ CSP 配置严格
    - ✅ API 权限最小化
    - ✅ 代码签名有效
    - ✅ 安全审计通过
    
    ### 代码质量
    - ✅ Rust 代码符合最佳实践
    - ✅ 前端代码模块化清晰
    - ✅ 测试覆盖率 > 80%
    - ✅ 文档完整准确
  </criteria>
</execution>
