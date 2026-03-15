#!/bin/sh
set -e

# 从共享数据中读取域名列表（由 Go 后端维护）
# 如果文件不存在，创建空文件
touch /etc/postfix/virtual_domains

# 运行升级配置（兼容新版 Postfix）
postfix upgrade-configuration 2>/dev/null || true

# 修复权限
postfix set-permissions 2>/dev/null || true
newaliases 2>/dev/null || true

echo "Starting Postfix in foreground mode..."
exec postfix start-fg
