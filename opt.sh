# 安装 crd
make install
# 卸载 crd
make uninstall
# 打包镜像
make docker-build IMG=124.71.131.239:8889/zhijian/opdemo:v1.0.0
# 推送镜像
make docker-push IMG=124.71.131.239:8889/zhijian/opdemo:v1.0.0
# 部署 operator 到集群中 引号必须添加
make deploy IMG="124.71.131.239:8889/zhijian/opdemo:v1.0.0"
# 卸载部署
make undeploy
