# 本地验证

本分支只使用 GitHub 存储源码，不运行 GitHub Actions。同步上游时不要引入 workflows 或 composite actions。构建与验证入口为 `scripts/build-local.sh` 和 `scripts/check-local.sh`，均在本机执行。
