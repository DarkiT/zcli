#!/bin/bash

# 获取当前时间
BUILD_TIME=$(date +"%Y-%m-%d %H:%M:%S")

# 获取Git信息
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
GIT_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "unknown")

# 设置输出文件名
OUTPUT="zcli"

# 显示构建信息
echo "正在构建 $OUTPUT..."
echo "构建时间: $BUILD_TIME"
echo "Git提交: $GIT_COMMIT"
echo "Git分支: $GIT_BRANCH"
echo "Git标签: $GIT_TAG"

# 使用ldflags注入编译时间和Git信息
# 注意: 请将下面的包路径替换为您的实际包路径
go build -ldflags "\
-X 'github.com/你的用户名/你的项目/zcli.BuildTimeStr=$BUILD_TIME' \
-X 'github.com/你的用户名/你的项目/zcli.GitCommit=$GIT_COMMIT' \
-X 'github.com/你的用户名/你的项目/zcli.GitBranch=$GIT_BRANCH' \
-X 'github.com/你的用户名/你的项目/zcli.GitTag=$GIT_TAG'" \
-o $OUTPUT main.go

echo "构建完成: $OUTPUT" 