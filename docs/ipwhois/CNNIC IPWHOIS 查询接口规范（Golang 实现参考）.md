# CNNIC IPWHOIS 查询接口规范（Golang 实现参考）

查询页面：
https://ipwhois.cnnic.net.cn/

## 基础请求信息
```go
const (
    BaseURL     = "http://ipwhois.cnnic.net.cn/bns/query/Query/ipwhoisQuery.do"
    UserAgent   = "Mozilla/5.0 (compatible; Go-http-client)" // 建议自定义UA
    Timeout     = 10 * time.Second
)
```

## 查询类型枚举
```go
type QueryType string

const (
    IPv4Query    QueryType = "ipv4"
    PersonQuery  QueryType = "person"
    IPv6Query    QueryType = "ipv6"
    ASNQuery     QueryType = "asn"
    RouteQuery   QueryType = "route"
    NetnameQuery QueryType = "netname"
)
```

## 请求参数规范

### 通用参数结构
```go
type WhoisRequest struct {
    QueryValue  string    // 查询内容
    QueryOption QueryType // 查询类型枚举
}
```

### 各类型参数要求
1. **IPv4查询**  
   - `QueryOption`: `ipv4`  
   - `QueryValue`格式要求：
     - 单地址：`210.72.0.0`
     - CIDR表示法：`210.72.0.0/19`
     - 地址范围：`210.72.0.0 - 210.72.31.255`

2. **联系人查询**  
   - `QueryOption`: `person`  
   - `QueryValue`示例：`IPAS1-CN`（需符合CNNIC联系人ID格式）

3. **IPv6查询**  
   - `QueryOption`: `ipv6`  
   - `QueryValue`格式要求：
     - 压缩格式：`2001:F88::`
     - CIDR表示法：`2001:F88::/32`

4. **AS号码查询**  
   - `QueryOption`: `asn`  
   - `QueryValue`接受格式：
     - 纯数字：`9811`
     - AS前缀格式：`AS9811`

5. **路由注册查询**  
   - `QueryOption`: `route`  
   - `QueryValue`要求：CIDR格式  
     示例：`211.144.211.0/24`

6. **网络名称查询**  
   - `QueryOption`: `netname`  
   - `QueryValue`示例：`CSTNET`（需准确匹配注册名称）

## 注意事项
1. 防重复提交机制：
   - 原系统通过`checkSubmit()`函数防止重复请求
   - 建议实现请求间隔控制（≥1秒）

2. 请求头要求：
   - 必须设置`User-Agent`
   - 建议添加`Accept-Language: zh-CN,zh`

3. 响应处理：
   - 编码为UTF-8
   - 响应内容为HTML格式（非JSON）

4. 错误处理：
   - 需要处理HTTP状态码≠200的情况
   - 注意DNS解析超时问题

5. 合规要求：
   - 遵守CNNIC的查询频率限制
   - 禁止商业性高频查询

6. 请求时要能够支持设置代理IP，并且代理IP是一个可选性，不设置的话也能够请求成功；
