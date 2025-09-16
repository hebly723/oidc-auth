# 邀请码系统技术文档

## 概述

本文档描述了OIDC认证系统中新增的邀请码功能模块。该功能允许用户生成邀请码，新用户在登录时可以使用邀请码，系统会记录邀请关系并支持向外部服务上报邀请码使用事件。

## 功能需求

### 核心功能
1. **邀请码生成**：用户可以生成唯一的邀请码
2. **邀请码验证**：新用户登录时可以使用邀请码建立邀请关系
3. **使用记录追踪**：记录邀请码的使用情况和邀请关系
4. **API接口**：提供RESTful API管理邀请码

### 业务规则
- 邀请码长度：8位字符，包含大写字母和数字
- 邀请码有效期：30天
- 用户注册后可使用邀请码的有效期：1天
- 每个用户只能使用一次邀请码

## 系统架构

### 模块组成

```mermaid
graph TB
    A[HTTP请求] --> B[Router路由层]
    B --> C[Handler处理层]
    C --> D[Service业务层]
    D --> E[Repository数据层]
    E --> F[Database数据库]
    
    C --> G[Middleware中间件]
    
    %% 原有模块样式
    classDef existing fill:#e1f5fe,stroke:#01579b,stroke-width:2px,color:#000
    
    %% 新增邀请码相关模块样式
    classDef inviteCode fill:#fff3e0,stroke:#e65100,stroke-width:3px,color:#000
    
    %% 应用样式到原有模块
    class A,B,C,D,E,F,G existing
    
    %% 新增的邀请码专用模块
    C1[InviteCodeHandler]:::inviteCode
    C2[InviteCodeServer]:::inviteCode
    D1[InviteCodeService]:::inviteCode
    E1[InviteCode Model]:::inviteCode
    E2[InviteCodeUsage Model]:::inviteCode
    M1[InviteCode Middleware]:::inviteCode
    
    %% 连接关系
    C --> C1
    C --> C2
    D --> D1
    E --> E1
    E --> E2
    G --> M1
    
    %% 邀请码模块内部关系
    C1 --> D1
    D1 --> E1
    D1 --> E2
    
    %% 图例
    subgraph Legend["图例说明"]
        L1[原有模块]:::existing
        L3[新增邀请码模块]:::inviteCode
    end
```

### 模块分类详细说明

#### 🔵 原有模块（蓝色）
系统原有的核心架构组件，为OIDC认证系统的基础框架：
- **HTTP请求**：客户端发起的HTTP请求入口
- **Router路由层**：负责请求路由分发和中间件管理
- **Handler处理层**：处理具体的业务请求逻辑
- **Service业务层**：核心业务逻辑处理
- **Repository数据层**：数据访问和持久化操作
- **Database数据库**：数据存储层
- **Middleware中间件**：请求预处理和后处理逻辑

#### 🟠 新增邀请码专用模块（橙色）
专门为邀请码功能开发的业务模块：
- **InviteCodeHandler**：邀请码相关API请求处理器
- **InviteCodeServer**：邀请码服务端处理逻辑
- **InviteCodeService**：邀请码核心业务逻辑服务
- **InviteCode Model**：邀请码数据模型
- **InviteCodeUsage Model**：邀请码使用记录数据模型
- **InviteCode Middleware**：邀请码相关中间件处理

### 数据模型

```mermaid
erDiagram
    InviteCode {
        uuid id PK
        string code UK
        uuid user_id FK
        timestamp created_at
        timestamp updated_at
    }
    
    InviteCodeUsage {
        uuid id PK
        uuid invite_code_id FK
        uuid user_id FK
        timestamp used_at
        timestamp created_at
        timestamp updated_at
    }
    
    AuthUser {
        uuid id PK
        string username
        timestamp created_at
    }
    
    InviteCode ||--o{ InviteCodeUsage : "一对多"
    AuthUser ||--o{ InviteCode : "生成"
    AuthUser ||--o{ InviteCodeUsage : "使用"
```

## 核心流程

### 邀请码生成流程

```mermaid
sequenceDiagram
    participant U as 用户
    participant H as Handler
    participant S as InviteCodeService
    participant D as Database
    
    U->>H: POST /invite/generate
    H->>H: 验证请求参数
    H->>D: 验证用户是否存在
    D-->>H: 返回用户信息
    H->>S: 调用生成邀请码
    S->>S: 生成随机8位码
    S->>D: 检查邀请码是否重复
    D-->>S: 返回检查结果
    alt 邀请码重复
        S->>S: 重新生成邀请码
    end
    S->>D: 保存邀请码记录
    D-->>S: 返回保存结果
    S-->>H: 返回邀请码信息
    H-->>U: 返回生成结果
```

### 邀请码使用流程

```mermaid
sequenceDiagram
    participant U as 新用户
    participant L as LoginHandler
    participant M as Middleware
    participant IC as InviteCodeHandler
    participant IS as InviteCodeService
    participant D as Database
    
    U->>L: 登录请求(带invite_code参数)
    L->>L: 执行OAuth登录流程
    L->>M: 获取邀请码处理器
    M-->>L: 返回处理器实例
    L->>IC: 验证邀请码
    IC->>IS: 调用验证服务
    IS->>D: 检查邀请码是否存在
    D-->>IS: 返回邀请码信息
    IS->>IS: 验证邀请码有效期
    IS->>IS: 验证用户注册时间
    IS->>D: 检查用户是否已使用过邀请码
    D-->>IS: 返回使用记录
    IS->>D: 创建邀请码使用记录
    D-->>IS: 返回创建结果
    IS-->>IC: 返回使用记录
    IC-->>L: 返回处理结果
    L-->>U: 完成登录
```

## API接口设计

### 生成邀请码

**请求**
```http
POST /oidc-auth/api/v1/invite/generate
Content-Type: application/json

{
    "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**响应**
```json
{
    "code": 200,
    "message": "Invite code generated successfully",
    "data": {
        "code": "ABC12345",
        "created_at": "2024-01-15 10:30:00"
    }
}
```

### 获取邀请码列表

**请求**
```http
GET /oidc-auth/api/v1/invite/list?user_id=550e8400-e29b-41d4-a716-446655440000
```

**响应**
```json
{
    "code": 200,
    "message": "Success",
    "data": {
        "codes": [
            {
                "id": "660e8400-e29b-41d4-a716-446655440001",
                "code": "ABC12345",
                "created_at": "2024-01-15 10:30:00"
            }
        ]
    }
}
```

### 登录时使用邀请码

**请求**
```http
GET /oidc-auth/api/v1/plugin/login?provider=casdoor&invite_code=ABC12345
```


## 数据库设计

### 邀请码表 (invite_codes)

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | UUID | PRIMARY KEY | 主键 |
| code | VARCHAR(32) | UNIQUE, NOT NULL | 邀请码 |
| user_id | UUID | NOT NULL, INDEX | 生成用户ID |
| created_at | TIMESTAMPTZ | NOT NULL | 创建时间 |
| updated_at | TIMESTAMPTZ | NOT NULL | 更新时间 |

### 邀请码使用记录表 (invite_code_usages)

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | UUID | PRIMARY KEY | 主键 |
| invite_code_id | UUID | NOT NULL, INDEX | 邀请码ID |
| user_id | UUID | NOT NULL, INDEX | 使用用户ID |
| used_at | TIMESTAMPTZ | NOT NULL | 使用时间 |
| created_at | TIMESTAMPTZ | NOT NULL | 创建时间 |
| updated_at | TIMESTAMPTZ | NOT NULL | 更新时间 |

### 索引设计

```sql
-- 邀请码表索引
CREATE UNIQUE INDEX idx_invite_codes_code ON invite_codes(code);
CREATE INDEX idx_invite_codes_user_id ON invite_codes(user_id);

-- 使用记录表索引
CREATE INDEX idx_invite_code_usages_invite_code_id ON invite_code_usages(invite_code_id);
CREATE INDEX idx_invite_code_usages_user_id ON invite_code_usages(user_id);
```

## 错误处理

### 常见错误码

| 错误码 | 说明 | 处理方式 |
|--------|------|----------|
| 400 | 请求参数错误 | 检查请求参数格式 |
| 404 | 用户或邀请码不存在 | 验证用户ID和邀请码 |
| 409 | 邀请码已被使用 | 提示用户邀请码已使用 |
| 410 | 邀请码已过期 | 提示用户邀请码过期 |
| 500 | 服务器内部错误 | 检查日志排查问题 |

### 业务异常处理

1. **邀请码重复**：自动重新生成
2. **邀请码过期**：返回明确错误信息
3. **用户已使用邀请码**：阻止重复使用
4. **数据库连接失败**：返回服务不可用错误

## 性能考虑

### 优化策略

1. **数据库索引**：为常用查询字段建立索引
2. **连接池**：使用数据库连接池提高并发性能
3. **缓存策略**：可考虑对热点邀请码进行缓存

### 监控指标

1. **邀请码生成速率**：监控邀请码生成的QPS
2. **邀请码使用率**：统计邀请码的使用情况
3. **响应时间**：监控API接口的响应时间
4. **错误率**：监控各类错误的发生频率

## 部署说明

### 环境要求

- Go 1.19+
- PostgreSQL 12+

### 配置步骤

1. 执行数据库迁移，创建相关表结构
2. 重启服务使配置生效
3. 验证API接口功能正常

### 监控部署

建议部署以下监控：
- 数据库性能监控
- API接口响应时间监控
- 错误日志告警

## 安全考虑

### 安全措施

1. **邀请码随机性**：使用加密安全的随机数生成器
2. **参数验证**：严格验证所有输入参数
3. **权限控制**：验证用户权限后才能生成邀请码
4. **防重放攻击**：邀请码只能使用一次
5. **日志记录**：记录所有关键操作的审计日志

### 数据保护

1. **敏感数据加密**：考虑对邀请码进行加密存储
2. **访问控制**：限制对邀请码数据的访问权限
3. **数据备份**：定期备份邀请码相关数据
4. **数据清理**：定期清理过期的邀请码数据

## 测试策略

### 单元测试

- 邀请码生成逻辑测试
- 邀请码验证逻辑测试
- 数据库操作测试

### 集成测试

- API接口测试
- 登录流程集成测试
- 错误处理测试

### 性能测试

- 并发邀请码生成测试
- 大量邀请码使用测试
- 数据库性能测试

## 版本历史

| 版本 | 日期 | 变更内容 |
|------|------|----------|
| 1.0.0 | 2024-01-15 | 初始版本，实现基础邀请码功能 |

## 相关文档

- [API参考文档](api-reference.md)
- [数据库设计文档](database-design.md)
- [认证流程文档](authentication-flow.md)
- [部署指南](deployment-guide.md)