# index_v3_2_6.php
- 该文件兼容goagent php模式;
- goagent php客户端均可使用，不限于这里以golang重构的版本

# index.php
- 该文件v3.3.0(包括)之后已经不兼容goagent php模式;
- 为了优化服务端的传输延时和服务端性能，最新的index.php支持chunked内容编码
- 使用该版本服务端文件时，建议使用php-proxy v2.1.4(包括)以上版本

# index.js
- 该文件是部署在Cloudflare Wokers上使用的，目前基本功能已经完成,属于测试阶段。
- 使用该版本服务端文件时，建议使用php-proxy v2.1.4(包括)以上版本
- 由于workers上没有广告，所以基本的XOR加密去掉了。
- 脚本实现参考项目[GotoX](https://github.com/SeaHOH/GotoX),非常感谢作者，如有侵权，删
- 为什么实现了js版本？因为Workers快啊，太快了,体验太好了!
- 下面是配置参考，fetchserver填入Workers地址就可以了
```
{
"fetchserver": "https://inner-free-9539.domain.workers.dev/",
"password": "123456",
"sni": "",
"listen": "127.0.0.1:8080",
"debug": false,
"insecure": false,
"autoproxy": false,
"user-agent": "",
"proxy": ""
}
```
