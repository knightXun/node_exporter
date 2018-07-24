1. 监控系统日志 参考 kubernetes/node-problem-detector
2. 监控kubernetes, etcd, flannel, docker进程的运行状态, 采集其日志的error,warning级别的日志,上传为时序数据
3. 邮件报警,可配置 取缔Monit,不能报警的上传给上游
4. monit功能加强
5. 监控信息时序化
6. 采集节点kubelet的event 参考heapster
7. 采集节点manifest 进程的health
8. docker images的情况
9. 主要目录 /var /root 的使用情况
10. 支持ceph的监控
10. plugin化,后期支持ceph的监控
11. gossip
12. 开放升级安装的接口,插件化,由上游触发进行升级工作
13. 集群健康状态展示