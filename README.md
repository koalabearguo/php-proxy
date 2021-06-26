# php-proxy
最近在学习golang，闲来无聊就找个项目把它用golang写了一遍，嗯，golang真香；
本项目主要实现的是GoAgent的php代理模式，协议兼容GoAgent php模式；
本项目主要用于个人学习，研究免费php空间的用途。

### 特性改进
- 在连接php server时，https模式支持TLS sni的发送，可以用来穿过CDN，尤其是cloudflare
- 代理模式添加支持HTTP OPTIONS请求，chrome浏览器会用OPTIONS方法
- 修复了curl请求出错时，返回的数据还是加密（简单XOR运算）的小问题
- 修复了hostsdeny匹配时，返回的数据还是加密（简单XOR运算）的小问题
- 屏蔽websocket升级请求头的发送
- 支持HTTP2，server与client保持一个连接，端口复用性能提升

### 协议分析
- 简单的来讲就是把客户端请求的数据（头+Body）,打包POST到php server，格式如下：
|2 byte(header length)| + |header(deflate压缩)| + |body| = POST Body,其中header中的第一行的url路径是绝对路径
- 这种代理模式实际上用的是HTTP/1.0模式，每次请求接收完成后就关闭了socket，没有connection：keep-alive，所以没有正向代理性能好，例如squid。更没有http2的端口复
用效果好
- GoAgent由于python ssl版本的问题，在向https server发送请求时，并不会发送tls中的sni扩展，当然Host头还是会发的，不过这样好像不能穿过CDN，试过cloudflare，请求不成功
- 这种模式的代理相当于经过了三级的代理，本机算一级，php server算一级，curl算一级，所以延时有点高，小流量使用还行吧
- 这种代理只能代理http/https连接，当然curl应该是支持ftp的，不过CONNECT请求信息中，我们并不能知道接下来要走什么协议，默认是处理成https了
- 这个简单的XOR加密是真的不错，虽然不能保证数据安全，但是放到有广告的php免费空间中，乱码中主机商不知道怎么插入广告了。。。具有广告过滤的功能了,哈哈哈

### TODO
- 增加请求头添加的配置，也许可以用来放到国内外(免费)的php空间，做免流代理
- 初次写golang，软件架构估计设计也不合理，慢慢改进

### 感谢
- GoAgent项目，让我学习了Python，php
- go-httpproxy/httpproxy项目，SSL中间人拦截模式以及Ca签发是从这里学的
