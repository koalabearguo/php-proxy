# index_v3_2_6.php
- 该文件兼容goagent php模式;
- goagent php客户端均可使用，不限于这里以golang重构的版本

# index.php
- 该文件v3.3.0(包括)之后已经不兼容goagent php模式;
- 为了优化服务端的传输延时和服务端性能，最新的index.php支持chunked内容编码
- 使用该版本服务端文件时，需要使用php-proxy v2.1.4(包括)以上版本
