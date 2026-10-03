# 完整WHOIS查询系统需求文档

## 1. 系统概述

开发一个支持多域名后缀、具备代理能力和验证码处理的WHOIS查询系统，提供命令行和API两种使用方式。

## 2. 功能需求

### 2.1 核心查询功能

- 支持.cn/.com/.net等主流域名后缀
- 自动识别域名类型选择对应WHOIS服务器
- 支持单域名查询和批量查询模式
- 返回结构化WHOIS数据

### 2.2 代理支持

- 支持HTTP/HTTPS/SOCKS5代理协议
- 代理认证（用户名/密码）
- 代理自动轮换和故障转移
- 代理性能测试和评分

### 2.3 验证码处理

- 自动检测和下载验证码图片
- 支持人工输入和自动识别两种模式
- 验证码提交和验证结果处理
- 验证码失败自动重试机制

### 2.4 会话管理

- 自动维护JSESSIONID等会话信息
- Cookie持久化和恢复
- 请求头自动生成和管理

## 3. 非功能需求

### 3.1 性能需求

- 单次查询响应时间 < 3秒
- 支持50+并发查询
- 内存占用 < 500MB（100并发时）

### 3.2 可靠性

- 自动重试失败查询（可配置次数）
- 网络中断自动恢复
- 数据完整性校验

### 3.3 安全性

- 代理认证信息加密存储
- 查询频率自动限制
- 敏感信息脱敏处理

## 4. 技术规范

### 4.1 架构设计

```plaintext
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  控制模块   │───▶│ 查询引擎   │───▶│ WHOIS适配器 │
└─────────────┘    └─────────────┘    └─────────────┘
       ▲                  ▲                  ▲
       │                  │                  │
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ 代理管理    │    │验证码处理器│    │会话管理器   │
└─────────────┘    └─────────────┘    └─────────────┘
```

### 4.2 数据流程

1. 输入域名列表
2. 初始化代理和会话
3. 发起WHOIS查询
4. 处理可能的验证码
5. 解析和返回结果

### 4.3 接口定义

```go
type WhoisClient interface {
    Query(domain string) (*WhoisResult, error)
    SetProxy(proxy string) error
    SetCaptchaHandler(handler CaptchaHandler)
}

type CaptchaHandler interface {
    Resolve(imageData []byte) (string, error)
}
```

## 5. 部署要求

### 5.1 运行环境

- Go 1.18+
- Linux/Windows/macOS
- 网络连接WHOIS服务器

### 5.2 依赖项

- 代理支持库
- 图像处理库（验证码识别）
- 并发控制库

## 6. 测试计划

### 6.1 测试用例

- 单域名查询测试
- 批量查询压力测试
- 代理切换测试
- 验证码处理测试

### 6.2 验收标准

- 所有域名类型查询成功率 > 95%
- 验证码识别准确率 > 80%
- 代理切换成功率 100%

## 7. 附录

### 7.1 术语表

- WHOIS: 域名注册信息查询协议
- gTLD: 通用顶级域名
- CNIC: 中国互联网络信息中心

### 7.2 参考文档

- IANA WHOIS协议规范
- CNIC WHOIS接口文档
- 代理服务器配置指南



验证码可能会随着页面html返回：

```html
<div id="fancybox-content" style="border-width: 0px; width: 402px; height: auto;"><div style="width:auto;height:auto;overflow: auto;position:relative;">
		<div class="popup">
			<div class="popup_title">
				<span>验证码</span>
				<div style="margin-right: 5px;">
					<a href="javascript:void(0)" onclick="closeWin()"></a>
				</div>
			</div>
			<div class="popup_concent">
				<div class="fb_node_container">
					<div class="yhw_news">如果您需要继续查询，请输入验证码</div>
					<div class="yhw_input_bar">
						<div class="yhw_input" style="border-color: rgb(197, 197, 197);">
							<span> <input type="text" name="validate" id="validate" maxlength="4" class="input" onkeydown="return submitValidate(event);" style="border: 1px solid red;"> </span>
						</div>
						<div class="tip" id="validate_hint" style="display: block; margin-left: 16px;">
							<span></span>验证码错误
						</div>
					</div>
					<div class="yhw_validate_bar">
						<div class="validate_pt">
							<img src="ImageServlet?t=0.5657076744654301" alt="valideCode" id="validateImg" class="img" width="120" height="50" onclick="this.src='ImageServlet?t=' + Math.random();">
						</div>
						<div class="validate_ft">
							<font color="#100c6b">看不清？请点击图片</font>
						</div>
					</div>
				</div>
			</div>
			<div class="yhw_validate_bottom">
				<table width="100%">
					<tbody><tr>
						<td width="70%"> </td>
						<td width="30%">
							<a id="subb" href="javascript:void(0)" onclick="queryValidateCode()">
								<span class="buttona">
									确定
								</span>
							</a>
							
							<a href="javascript:void(0)" onclick="closeWin()"><span class="buttona">
						 		取消
							</span>
							</a>
						</td>
					</tr>
				</tbody></table>
			</div>
		</div>
	</div></div>
```



提交验证码的请求：

```bash
curl 'https://webwhois.cnnic.cn/AjaxValidateCodeServlet?validate=1111' \
  -H 'Accept: text/plain, */*; q=0.01' \
  -H 'Accept-Language: zh-CN,zh;q=0.9' \
  -H 'Cache-Control: no-cache' \
  -H 'Connection: keep-alive' \
  -b 'whoisweb_num=0; JSESSIONID=C22D5BAF3E83A4DD7995FE8989DBE110' \
  -H 'Pragma: no-cache' \
  -H 'Referer: https://webwhois.cnnic.cn/WhoisServlet?queryType=Domain&domain=alipay.cn' \
  -H 'Sec-Fetch-Dest: empty' \
  -H 'Sec-Fetch-Mode: cors' \
  -H 'Sec-Fetch-Site: same-origin' \
  -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36' \
  -H 'X-Requested-With: XMLHttpRequest' \
  -H 'sec-ch-ua: "Chromium";v="134", "Not:A-Brand";v="24", "Google Chrome";v="134"' \
  -H 'sec-ch-ua-mobile: ?0' \
  -H 'sec-ch-ua-platform: "macOS"'
```





