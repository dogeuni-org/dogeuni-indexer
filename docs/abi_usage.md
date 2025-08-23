# Cardity ABI 使用指南

## 🎯 **ABI 的本质与设计理念**

### **ABI 是什么？**
- **ABI (Application Binary Interface)** 是 Cardity 编译器 (`cardityc`) 在编译时生成的接口定义文件
- **链下工件**：ABI 不需要上链，但可以存储在索引中供查询使用
- **查询工具**：ABI 用于帮助用户发现和查询索引中的合约内容

### **为什么 ABI 不上链？**
1. **链上存储成本**：ABI 是纯文本，上链会增加不必要的存储成本
2. **链下更灵活**：开发者可以随时更新 ABI 而不需要重新部署合约
3. **隐私控制**：开发者可以选择哪些合约公开 ABI，哪些保持私有

### **ABI 在索引中的作用**
- **合约发现**：通过方法名、事件名等语义化信息发现合约
- **接口查询**：让用户能够通过 ABI 信息查询索引内容
- **用户体验**：提供更友好的合约搜索和发现方式

## 🔧 **ABI 的正确使用流程**

### **1. 编译时生成 ABI**
```bash
# 使用 cardityc 编译器
./build/cardityc my_protocol.car

# 生成的文件：
# - my_protocol.carc (链上部署)
# - my_protocol.abi.json (链下使用)
```

### **2. 部署合约**
```bash
# 只部署 .carc 文件到链上
# ABI 文件保留在本地
```

### **3. ABI 管理（链下生成，索引存储）**
```bash
# ABI 由开发者管理，可以：
# - 保存在本地开发环境
# - 上传到 GitHub 等代码仓库
# - 分享给其他开发者
# - 用于生成 SDK 和文档
# - 存储在索引中供查询使用
```

## 📡 **索引服务查询 API**

### **基础查询（基于链上数据）**
```bash
# 通过合约 ID 查询
GET /v4/cardity/contract/{contract_id}

# 通过协议查询
POST /v4/cardity/contracts
{
  "protocol": "USDTLikeToken",
  "version": "1.0.0"
}

# 通过合约引用查询
POST /v4/cardity/contracts
{
  "contract_ref": "USDTLikeToken@1.0.0"
}
```

### **ABI 增强查询（通过 ABI 发现合约）**
```bash
# 通过 ABI Hash 查询
POST /v4/cardity/abi/search
{
  "abi_hash": "your_abi_hash"
}

# 通过方法名搜索
POST /v4/cardity/abi/search
{
  "method_name": "transfer"
}

# 通过事件名搜索
POST /v4/cardity/abi/search
{
  "event_name": "Transfer"
}
```

### **ABI 管理**
```bash
# 查询 ABI（如果索引中有）
GET /v4/cardity/abi/{contract_id}

# 通过 ABI 搜索合约
POST /v4/cardity/abi/search

# 查看 ABI 统计
GET /v4/cardity/abi/stats
```

## 🚀 **最佳实践**

### **对于合约开发者**
1. **编译时生成 ABI**：使用 `cardityc` 生成完整的 ABI
2. **本地管理 ABI**：将 ABI 保存在项目目录中
3. **版本控制**：将 ABI 纳入 Git 版本管理
4. **文档化**：为 ABI 提供使用说明和示例
5. **索引存储**：将 ABI 存储在索引中供其他用户查询

### **对于 dApp 开发者**
1. **获取 ABI**：从合约开发者处获取 ABI 文件
2. **本地集成**：将 ABI 集成到 dApp 中
3. **动态加载**：根据 ABI 动态生成交互界面
4. **版本管理**：确保 ABI 版本与部署的合约匹配
5. **索引查询**：通过索引服务查询合约的 ABI 信息

### **对于索引服务**
1. **存储 ABI**：存储开发者提供的 ABI 信息
2. **提供查询**：通过 ABI 信息帮助用户发现合约
3. **接口增强**：利用 ABI 提供更好的合约搜索体验
4. **不强制要求**：ABI 不是索引的必需部分，但有助于用户体验

## 🔍 **实际使用示例**

### **场景 1：开发者部署合约**
```bash
# 1. 编译合约
cardityc my_token.car

# 2. 部署 .carc 文件到链上
# 3. 将 ABI 保存在项目目录中
# 4. 将 ABI 提交到 Git 仓库
# 5. 将 ABI 存储在索引中供查询
```

### **场景 2：用户查询合约**
```bash
# 1. 通过索引服务查询合约基础信息
GET /v4/cardity/contract/{id}

# 2. 通过 ABI 信息搜索相关合约
POST /v4/cardity/abi/search
{
  "method_name": "transfer"
}

# 3. 获取合约的 ABI 信息
GET /v4/cardity/abi/{id}
```

### **场景 3：dApp 集成**
```javascript
// 1. 从索引服务获取 ABI
const abi = await fetch('/v4/cardity/abi/' + contractId);

// 2. 使用 ABI 生成交互界面
generateUI(abi);

// 3. 通过索引服务查询合约状态
const contract = await fetch('/v4/cardity/contract/' + contractId);

// 4. 通过 ABI 搜索相关合约
const similarContracts = await fetch('/v4/cardity/abi/search', {
  method: 'POST',
  body: JSON.stringify({ method_name: 'transfer' })
});
```

## 📝 **总结**

Cardity 的 ABI 设计体现了以下原则：
1. **链上最小化**：只存储必要的 `.carc` 文件
2. **链下管理**：ABI 由开发者管理，但可以存储在索引中
3. **查询增强**：ABI 用于增强索引的查询和发现功能
4. **用户体验**：通过 ABI 提供更好的合约发现体验

**重要说明**：
- **ABI 不上链**：ABI 是链下工件，不需要上链
- **ABI 可索引**：ABI 可以存储在索引中供查询使用
- **查询工具**：ABI 用于通过接口信息查询索引内容
- **合约发现**：通过 ABI 信息帮助用户发现相关合约

这种设计确保了：
- 链上存储效率
- 开发者对 ABI 的完全控制
- 索引服务的查询增强
- 用户的合约发现体验
- 系统的清晰职责分离
