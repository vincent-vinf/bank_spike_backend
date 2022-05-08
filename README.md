# golang秒杀系统后端

## 架构设计

### 微服务划分

* 用户
* 秒杀
* 准入
* 管理

```bash
docker build --target access -t access-server:0.0.1 .
docker build --target spike -t spike-server:0.0.1 .
docker build --target user -t user-server:0.0.1 . 
docker build --target admin -t admin-server:0.0.1 . 
```