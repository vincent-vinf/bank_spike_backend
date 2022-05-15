# golang秒杀系统后端

## 架构设计

### 微服务划分

* 用户
  * 获取token

* 秒杀
  * 核心的秒杀逻辑

* 准入
  * 对指定的秒杀，根据灵活的规则配置来判断用户是否有资格

* 管理
  * 管理秒杀活动(CRUD)

* 订单
  * 处理订单


```bash
docker build --target access -t access-server:0.0.1 .
docker build --target spike -t spike-server:0.0.1 .
docker build --target user -t user-server:0.0.1 . 
docker build --target admin -t admin-server:0.0.1 . 
```