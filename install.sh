
#!/bin/bash
set -e

echo "Initializing MySQL Space Exporter..."

# 创建配置文件
if [ ! -f .env ]; then
    echo "Creating .env file from example..."
    cp .env.example .env
fi

# 赋予执行权限
chmod +x build.sh push.sh

# 构建镜像
echo "Building Docker image..."
./build.sh

echo "Installation complete!"
echo "Please edit .env file with your configuration before running."
echo "To start the exporter, run: docker-compose up -d"
