# php-proxy
最近在学习golang，闲来无聊就找个项目把它用golang写了一遍，嗯，golang真香；
本项目主要实现的是GoAgent的php代理模式，协议兼容GoAgent php模式；
本项目主要用于个人学习，研究免费php空间的用途，严禁用于非法用途，后果自负

### 特性改进
- 在连接php server时，https模式支持TLS sni的发送，可以用来穿过CDN，尤其是cloudflare
- 支持自定义TLS SNI的发送,可以用来欺骗xxx
- 代理模式添加支持HTTP OPTIONS请求，chrome浏览器会用OPTIONS方法
- 修复了curl请求出错时，返回的数据还是加密（简单XOR运算）的小问题
- 修复了hostsdeny匹配时，返回的数据还是加密（简单XOR运算）的小问题
- 屏蔽websocket协议，这种双向的通信协议，支持不好
- 支持HTTP2，server与client保持一个连接，端口复用性能提升
- 由于php-proxy.crt/key根证书公钥私钥公开了，这里在检测到以公开根证书作为根的中间人(php server)，就会断开与服务器的通信
- 为了安全性以及使用的灵活性，添加支持自定义CA
- 支持`HTTP3`(dev分支，用于尝鲜测试),测试可以跟cloudflare CDN通信

### 协议分析
- 简单的来讲就是把客户端请求的数据（头+Body）,打包POST到php server，格式如下：
|2 byte(header length)| + |header(deflate压缩)| + |body| = POST Body,其中header中的第一行的url路径是绝对路径;
返回的响应Body就是代理到的所有数据（头+Body）,经过解密后发送到浏览器客户端
- GoAgent由于python ssl版本的问题，在向https server发送请求时，并不会发送tls中的sni扩展，当然Host头还是会发的，不过这样好像不能穿过CDN，试过cloudflare，请求不成功
- 这种模式的代理相当于经过了三级的代理，本机算一级，php server算一级，curl算一级，所以延时有点高，小流量使用还行吧
- 这种代理只能代理http/https连接，当然curl应该是支持ftp的，不过CONNECT请求信息中，我们并不能知道接下来要走什么协议，默认是处理成https了
- 这个简单的XOR加密是真的不错，虽然不能保证数据安全，但是放到有广告的php免费空间中，乱码中主机商不知道怎么插入广告了。。。具有广告过滤的功能了,哈哈哈

### 使用
1. 把php-proxy.crt CA证书导入到系统中
2. 把index.php上传到一个(免费的)php空间中，位置名称随意,记下php文件的网络地址(这里最好用一个CDN加速，例如cloudflare，因为免费空间基本速度不快)
3. 在右侧release下载对应平台的可执行文件(这里也可以自己根据go文件生成,cmd命令窗口/linux终端下，切换到当前目录，直接go build \*.go([参考](https://github.com/koalabearguo/php-proxy/wiki))就可以了，如果构建失败，请升级go版本)
4. cmd命令窗口/linux终端执行php-proxy -s php文件的网络地址,这时本地127.0.0.1:8081就是一个http proxy
```
#windows监听127.0.0.1:8080
php-proxy.exe -s https://xxx.xx/free/index.php -l 127.0.0.1:8080

#linux默认监听127.0.0.1:8081
#php 地址https://xxx.xx/free/index.php
php-proxy -s https://xxxx.xx/proxy/index.php
```
5. 设置浏览器的http proxy地址127.0.0.1:8081
6. 在浏览器中输入你想浏览的网页...
7. 更多使用详情可以执行php-proxy -h获取
```
  -d    enable debug mode for debug
  -k    insecure connect to php server(ignore certs verify/middle attack)
  -l string
        Local listen address(HTTP Proxy address) (default "127.0.0.1:8081")
  -p string
        php server password (default "123456")
  -s string
        php fetchserver path(http/https) (default "https://a.bc.com/php-proxy/index.php")
  -sni string
        HTTPS sni extension ServerName(default fetchserver hostname)
```
8. v1.0(不包括)之后的版本支持json格式的配置文件(php-proxy.json)，当命令行有参数时，不使用配置文件，并会把命令行的数据写入json配置文件;
如果命令行没有参数,则从配置文件中读取配置信息,如果读取失败，则使用内部的默认参数
```
{
"fetchserver": "https://a.bc.com/go/index.php",
"password": "123456",
"sni": "a.bc.com",
"listen": "127.0.0.1:8081",
"debug": false,
"insecure": false
}

```
9. v1.1.2(包括)之后的版本支持自定义CA，只要把crt文件名字命名为php-proxy.crt，key文件命名为php-proxy.key，放到同一个目录下，程序会自动识别;
我这里自己生成的CA文件，为了安全，不建议大家使用，推荐大家使用自定义的CA,同时要注意自己的CA私钥不要随意泄露给别人;如果可执行文件目录下没有php-proxy.crt和
php-proxy.key文件，则使用内部预留的CA(也就是我自己生成的CA),不推荐这样使用
10. v1.1.3(包括)之后的版本，可以对抗DNS污染(不能对抗IP封锁哈,IP封锁了就要套CDN了)，或者指定DNS解析,比如你的域名helloworld.com被污染了,原先的配置
(php-proxy.json)可以修改成这样,一定要指定sni，否则大部分情况下是不工作的(除了独立服务器，没有使用virtual host类似功能的)
```
#当然修改系统的hosts文件，也可以完成相似的功能，不过在不是管理员时可能没有权限
"fetchserver": "https://域名解析正确的IP/go/index.php",
"sni": "helloworld.com",
```
另外隐藏的功能就是，假如你用了cloudflare的cdn，那么它给你域名分配的IP，一般有两个,这两个IP一般不是最优的，可以借助其他工具([better-cloudflare-ip](https://github.com/badafans/better-cloudflare-ip))查找其他的CDN ip,找到性能好的，
通过这种指定IP的模式，可以更好的使用CDN加速
另外提一下，如果IP没被封锁，但TLS SNI被盯上了，这个好办，换个域名就好了（好多地方可以免费申请域名）

11. v2.0.0版本，重写了数据处理过程，修复了下载问题，性能大幅提升，感谢[10362227](https://github.com/10362227)所做的对比与测试
### 注意事项
- 由于我自己生成的php-proxy.key/crt私钥和公钥的公开，如果导入到系统中，可能会导致一些钓鱼网站的恶意使用;在访问一些以Php-Proxy CA签发的https网站，本机浏览器
会直接信任这种网站,可能会造成隐私泄露;如果你用的chrome浏览器，建议php-proxy.crt证书不导到系统中，在chrome快捷方式目标后面加上--ignore-certificate-errors
这样chrome也是可以用的,只是地址栏中会显示红色,这不影响使用;如果确实你需要导入CA到系统中，则在不使用时，建议从根证书系统中删除;
所以为了安全起见，建议自己动手生成CA;当然你觉得也没啥隐私可泄露的，这也无所谓了

### TODO
- 增加请求头添加的配置，也许可以用来放到国内外(免费)的php空间，做免流代理
- 增加智能代理功能

### 感谢
- GoAgent项目，让我学习了Python，php
- [go-httpproxy/httpproxy](https://github.com/go-httpproxy/httpproxy)项目，SSL中间人拦截模式以及Ca签发是从这里学的
