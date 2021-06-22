# php-proxy
最近在学习golang，闲来无聊就找个项目把它用golang写了一遍，嗯，golang真香；
本项目主要实现的是GoAgent的php代理模式，协议兼容GoAgent php模式；
本项目主要用于个人学习，研究免费php空间的用途。

### 特性改进
- 在连接php server时，https模式支持TLS sni的发送，可以用来穿过CDN，尤其是cloudflare
- 代理模式添加支持HTTP OPTIONS请求，chrome浏览器会用OPTIONS方法
- 修复了curl请求出错时，返回的数据还是加密（简单XOR运算）的小问题
- 接收模式支持Transfer-Encoding：chunked

### 协议分析
- 简单的来讲就是把客户端请求的数据（头+Body）,打包POST到php server，格式如下：
|2 byte(header length)| + |header(deflate压缩)| + |body| = POST Body
- 这种代理模式实际上用的是HTTP/1.0模式，每次请求接收完成后就关闭了socket，没有connection：keep-alive，所有没有正向代理性能好，例如squid。更没有http2的端口复
用效果好
- GoAgent由于python版本的问题，在向https server发送请求时，并不会发送tls 中的sni扩展，当然Host头还是会发的，不过这样好像不能穿过CDN，试过cloudflare，请求不成功
- 这种模式的代理相当于经过了三级的代理，本机算一级，php server算一级，curl算一级，所以延时有点高，小流量使用还行吧

### TODO
- 目前的话只用curl测试过，发送接收数据没问题，可能还有bug，慢慢完善
- 现在的ca还是用的GoAgent，后面会自己再生成一份
- 增加请求头添加的配置，也许可以用来放到国内外(免费)的php空间，做免流代理
- 性能改进，看看能不能做到keep-alive，http端口复用什么的
- 初次写golang，软件架构估计设计也不合理，慢慢改进

### 感谢
- GoAgent项目，让我学习了Python，php
- go-httpproxy/httpproxy项目，SSL中间人拦截模式以级Ca签发是从这里学的
