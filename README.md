# url-download-speed-exporter

一个简单的监控指定url下载速度exporter

可用于监控服务器自身的实际使用场景下的入口带宽，也可以用于监控目标服务器的实际使用场景下的出口带宽

为提高数据准确性，建议使用至少2MB大小的url

## 使用方法
`/exporter --target={$url_1} --target={$url_2} --target={$url_3} ...`

比如使用docker
```docker run --rm -it -p 8080:8080 thebeet/url-download-speed-exporter --target=https://dl-cdn.alpinelinux.org/alpine/v3.16/releases/x86_64/alpine-minirootfs-3.16.1-x86_64.tar.gz --target=https://dl-cdn.alpinelinux.org/alpine/v3.16/releases/x86/alpine-minirootfs-3.16.1-x86.tar.gz```
